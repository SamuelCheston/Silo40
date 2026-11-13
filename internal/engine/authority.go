package engine

import (
	"math"
	"sort"

	"silo40/internal/model"
)

// AuthorityConfig centralizes the first authority-skeleton parameters.
// PoliceObedienceResponseRate is TODO(balance): the design specifies a causal
// response but not its tuning constant yet.
type AuthorityConfig struct {
	NewGameW                    float64
	LegacyConservativeW         float64
	BaseW                       float64
	KWeight                     float64
	DWeight                     float64
	PWeight                     float64
	XWeight                     float64
	EWeight                     float64
	WHalfLifeMonths             float64
	BailiffWeight               float64
	ITInternalWeight            float64
	SigmoidMidpoint             float64
	SigmoidWidth                float64
	MayorOpenW                  float64
	MayorReturnW                float64
	MayorMinLegitimacy          float64
	MayorEntryMonths            int
	MayorReturnMonths           int
	MayorAttachmentThreshold    float64
	MayorHighWCap               float64
	AttachmentInertia           float64
	PactOpenW                   float64
	PactMinQ                    float64
	PactMinFacts                int
	PactEntryMonths             int
	PactAttachmentThreshold     float64
	PoliceObedienceResponseRate float64
}

func DefaultAuthorityConfig() AuthorityConfig {
	return AuthorityConfig{
		NewGameW: 70, LegacyConservativeW: 50,
		BaseW: 50, KWeight: 25, DWeight: 12, PWeight: 10, XWeight: 12, EWeight: 40,
		WHalfLifeMonths: 8, BailiffWeight: 0.65, ITInternalWeight: 0.35,
		SigmoidMidpoint: 38, SigmoidWidth: 3.5,
		MayorOpenW: 30, MayorReturnW: 45, MayorMinLegitimacy: 0.35,
		MayorEntryMonths: 2, MayorReturnMonths: 3, MayorAttachmentThreshold: 0.60,
		MayorHighWCap: 0.075, AttachmentInertia: 0.6,
		PactOpenW: 18, PactMinQ: 0.85, PactMinFacts: 2, PactEntryMonths: 3,
		PactAttachmentThreshold:     0.55,
		PoliceObedienceResponseRate: 0.35, // TODO(balance): requires play measurement.
	}
}

// AuthorityInputs are the observable v8 sources. OrderDeviation is reserved
// for a later Order batch and intentionally has no effect on W yet.
type AuthorityInputs struct {
	BailiffLoyalty    float64
	ITInternalLoyalty float64
	Exposure          float64
	Delivery          float64
	Personnel         float64
	ExternalSignal    float64
	OrderDeviation    float64
}

// InitializeAuthorityState initializes a new game or a migrated v7 silo.
// Migrated W is explicitly conservative rather than presented as reconstructed history.
func InitializeAuthorityState(silo *model.Silo, migrated bool) {
	if silo == nil {
		return
	}
	cfg := DefaultAuthorityConfig()
	initializeMayorHolder(silo)
	silo.SetMayorLegitimacy(silo.MayorLegitimacy())
	w := cfg.NewGameW
	source := "new_game"
	if migrated {
		w = cfg.LegacyConservativeW
		source = "v7_conservative_w50"
	}
	silo.Authority = model.ShadowAuthority{
		Initialized: true, InitializationSource: source, W: w,
		PoliceObedience: silo.MayorLegitimacy(),
		Judicial:        model.JudicialAttachment{IT: 0.85, Mayor: 0.10, Pact: 0.05, Regime: "IT"},
	}
	j := &silo.Authority.Judicial
	j.TargetIT, j.TargetMayor, j.TargetPact = j.IT, j.Mayor, j.Pact
	s := 1 / (1 + math.Exp(-(w-cfg.SigmoidMidpoint)/cfg.SigmoidWidth))
	j.CommandIT = j.IT * s
	j.CommandMayor = j.Mayor * (0.75 + 0.25*silo.MayorLegitimacy())
	mirrorAuthorityRelations(silo)
}

// AssignMayorHolder attaches personal legitimacy to an existing named resident.
func AssignMayorHolder(silo *model.Silo, holderName string) bool {
	if silo == nil {
		return false
	}
	residents := append([]model.Resident(nil), silo.Residents...)
	sort.SliceStable(residents, func(i, j int) bool { return residents[i].ID < residents[j].ID })
	for i := range residents {
		resident := &residents[i]
		if resident.Name != holderName || !resident.Alive {
			continue
		}
		if silo.Mayor.HolderResidentID != resident.ID {
			resetJudicialForNewMayor(silo)
		}
		silo.Mayor.HolderResidentID = resident.ID
		silo.Mayor.HolderName = resident.Name
		silo.SetMayorLegitimacy(silo.MayorLegitimacy())
		return true
	}
	return false
}

// AdvanceAuthorityMonthly reads current observables and advances exactly one month.
func AdvanceAuthorityMonthly(silo *model.Silo) {
	if silo == nil {
		return
	}
	inputs := AuthorityInputs{
		BailiffLoyalty:    professionLoyalty(silo, "Judicial"),
		ITInternalLoyalty: professionLoyalty(silo, "IT"),
		Exposure:          clamp01(silo.SafeguardRisk),
		Delivery:          1 - 2*professionPanic(silo, "Judicial"),
		Personnel:         0, // No personnel-continuity observable exists in v8.
		ExternalSignal:    0, // Rare external signals are a later content input.
	}
	AdvanceAuthority(silo, inputs)
}

// AdvanceAuthority applies the deterministic formulas without consuming RNG.
func AdvanceAuthority(silo *model.Silo, inputs AuthorityInputs) {
	cfg := DefaultAuthorityConfig()
	AdvanceAuthorityWithConfig(silo, inputs, cfg)
}

func AdvanceAuthorityWithConfig(silo *model.Silo, inputs AuthorityInputs, cfg AuthorityConfig) {
	if silo == nil {
		return
	}
	if !silo.Authority.Initialized {
		InitializeAuthorityState(silo, false)
	}
	refreshMayorHolder(silo)

	a := &silo.Authority
	a.K = clamp01(cfg.BailiffWeight*clamp01(inputs.BailiffLoyalty) + cfg.ITInternalWeight*clamp01(inputs.ITInternalLoyalty))
	a.E = clamp01(inputs.Exposure)
	a.D = clampSigned(inputs.Delivery)
	a.P = clampSigned(inputs.Personnel)
	a.X = clampSigned(inputs.ExternalSignal)
	a.OrderDeviation = clampSigned(inputs.OrderDeviation) // Reserved; not used below.

	wEq := clamp(cfg.BaseW+cfg.KWeight*a.K+cfg.DWeight*a.D+cfg.PWeight*a.P+cfg.XWeight*a.X-cfg.EWeight*a.E, 0, 100)
	alpha := 1 - math.Pow(2, -1/cfg.WHalfLifeMonths)
	a.W = clamp(a.W+alpha*(wEq-a.W), 0, 100)

	l := silo.MayorLegitimacy()
	a.PoliceObedience = clamp01(a.PoliceObedience + cfg.PoliceObedienceResponseRate*(l-a.PoliceObedience))
	advanceJudicialAttachment(silo, cfg)
	mirrorAuthorityRelations(silo)
}

func advanceJudicialAttachment(silo *model.Silo, cfg AuthorityConfig) {
	a := &silo.Authority
	j := &a.Judicial
	l := silo.MayorLegitimacy()
	rawMayorOpen := a.W <= cfg.MayorOpenW && l >= cfg.MayorMinLegitimacy
	if rawMayorOpen {
		j.MayorLowMonths++
	} else {
		j.MayorLowMonths = 0
	}
	if a.W >= cfg.MayorReturnW {
		j.MayorHighMonths++
	} else {
		j.MayorHighMonths = 0
	}
	if j.Regime == "MAYOR" && j.MayorHighMonths >= cfg.MayorReturnMonths {
		j.Regime = "IT"
		j.MayorLowMonths = 0
		j.MayorHighMonths = 0
	}
	mayorOpen := rawMayorOpen || j.Regime == "MAYOR"

	pactOpen := j.PactQualified && a.W <= cfg.PactOpenW && j.PublicEvidenceQ >= cfg.PactMinQ && j.IndependentPublicFacts >= cfg.PactMinFacts
	if pactOpen {
		j.PactLowMonths++
	} else {
		j.PactLowMonths = 0
	}
	if !j.PactLocked && j.PactLowMonths >= cfg.PactEntryMonths && j.Pact >= cfg.PactAttachmentThreshold {
		j.PactLocked = true
		j.Regime = "PACT"
	}

	s := 1 / (1 + math.Exp(-(a.W-cfg.SigmoidMidpoint)/cfg.SigmoidWidth))
	f := 1 - s
	mayorMultiplier := 0.75 + 0.25*l
	var targetIT, targetMayor, targetPact float64
	if j.PactLocked {
		targetPact = 1
	} else if !pactOpen {
		h := 0.0
		if mayorOpen {
			h = 1
		}
		targetMayor = h*f*mayorMultiplier + (1-h)*cfg.MayorHighWCap*l
		targetIT = 1 - targetMayor
	} else {
		h := 0.0
		if mayorOpen {
			h = 1
		}
		zIT := s + (1-h)*f
		zMayor := h * f * mayorMultiplier
		zPact := 1.5 * f * j.PublicEvidenceQ
		z := zIT + zMayor + zPact
		if z > 0 {
			targetIT, targetMayor, targetPact = zIT/z, zMayor/z, zPact/z
		} else {
			targetIT = 1
		}
	}

	migration := cfg.AttachmentInertia
	j.IT = (1-migration)*j.IT + migration*targetIT
	j.Mayor = (1-migration)*j.Mayor + migration*targetMayor
	j.Pact = (1-migration)*j.Pact + migration*targetPact
	normalizeAttachment(j)
	if j.Regime != "MAYOR" && j.MayorLowMonths >= cfg.MayorEntryMonths && j.Mayor >= cfg.MayorAttachmentThreshold {
		j.Regime = "MAYOR"
		j.MayorHighMonths = 0
	}
	j.TargetIT, j.TargetMayor, j.TargetPact = targetIT, targetMayor, targetPact
	j.CommandIT = j.IT * s
	j.CommandMayor = j.Mayor * mayorMultiplier
	j.CommandPact = j.Pact * j.PublicEvidenceQ
	if j.PactLocked {
		j.CommandPact = j.Pact
	}
}

func normalizeAttachment(j *model.JudicialAttachment) {
	j.IT = clamp01(j.IT)
	j.Mayor = clamp01(j.Mayor)
	j.Pact = clamp01(j.Pact)
	sum := j.IT + j.Mayor + j.Pact
	if sum <= 0 {
		j.IT, j.Mayor, j.Pact = 1, 0, 0
		return
	}
	j.IT /= sum
	j.Mayor /= sum
	j.Pact /= sum
}

func mirrorAuthorityRelations(silo *model.Silo) {
	for i := range silo.Professions {
		profession := &silo.Professions[i]
		if profession.Relations == nil {
			profession.Relations = map[string]float64{}
		}
		switch profession.Name {
		case "Police":
			profession.Relations["Mayor"] = silo.Authority.PoliceObedience
		case "Judicial":
			profession.Relations["IT"] = silo.Authority.Judicial.CommandIT
			profession.Relations["Mayor"] = silo.Authority.Judicial.CommandMayor
			profession.Relations["pact"] = silo.Authority.Judicial.CommandPact
		}
	}
}

func initializeMayorHolder(silo *model.Silo) {
	if silo.Mayor.Initialized && silo.Mayor.HolderResidentID != 0 {
		return
	}
	residents := sortedResidents(silo)
	for i := range residents {
		if residents[i].Profession == "Mayor" && residents[i].Alive {
			silo.Mayor.HolderResidentID = residents[i].ID
			silo.Mayor.HolderName = residents[i].Name
			break
		}
	}
	if silo.Mayor.HolderName == "" {
		silo.Mayor.HolderName = "Mayor"
	}
}

func refreshMayorHolder(silo *model.Silo) {
	currentID := silo.Mayor.HolderResidentID
	residents := sortedResidents(silo)
	for i := range residents {
		if residents[i].ID == currentID && residents[i].Alive {
			return
		}
	}
	previousID := currentID
	silo.Mayor.HolderResidentID = 0
	silo.Mayor.HolderName = ""
	initializeMayorHolder(silo)
	if previousID != silo.Mayor.HolderResidentID {
		resetJudicialForNewMayor(silo)
	}
}

func resetJudicialForNewMayor(silo *model.Silo) {
	j := &silo.Authority.Judicial
	released := j.Mayor * 0.75
	j.Mayor *= 0.25
	if !j.PactLocked {
		j.IT += released
	}
	j.Regime = "SWING"
	j.MayorLowMonths = 0
	j.MayorHighMonths = 0
	normalizeAttachment(j)
}

func sortedResidents(silo *model.Silo) []model.Resident {
	residents := append([]model.Resident(nil), silo.Residents...)
	sort.SliceStable(residents, func(i, j int) bool { return residents[i].ID < residents[j].ID })
	return residents
}

func professionLoyalty(silo *model.Silo, name string) float64 {
	professionID := uint(0)
	for i := range silo.Professions {
		if silo.Professions[i].Name == name {
			professionID = silo.Professions[i].ID
			break
		}
	}
	cohorts := append([]model.PopulationCohort(nil), silo.Cohorts...)
	sort.SliceStable(cohorts, func(i, j int) bool { return cohorts[i].ID < cohorts[j].ID })
	total, weight := 0.0, 0.0
	for i := range cohorts {
		if cohorts[i].ProfessionID != professionID || cohorts[i].Count <= 0 {
			continue
		}
		w := float64(cohorts[i].Count)
		total += clamp01(cohorts[i].Loyalty) * w
		weight += w
	}
	if weight == 0 {
		return 0.8
	}
	return total / weight
}

func professionPanic(silo *model.Silo, name string) float64 {
	for i := range silo.Professions {
		if silo.Professions[i].Name == name {
			return clamp01(silo.Professions[i].PanicValue)
		}
	}
	return 0
}

func clampSigned(value float64) float64 { return clamp(value, -1, 1) }

func clamp(value, low, high float64) float64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

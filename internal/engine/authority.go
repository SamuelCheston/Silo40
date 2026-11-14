package engine

import (
	"math"
	"sort"

	"silo40/internal/model"
)

// AuthorityConfig centralizes the first authority-skeleton parameters.
// PoliceObedienceResponseRate is TODO(balance): the design specifies a causal
// response but not its tuning constant yet.
//
// G12 ruling: the Judicial never attaches to the Mayor — it attaches only to
// IT or to the pact. The low/high-W windows below therefore move the Judicial
// between IT control and the contested state (SWING); the Mayor keeps an
// influence modifier only (MayorOpenW / MayorMinLegitimacy / MayorHighWCap).
type AuthorityConfig struct {
	NewGameW            float64
	LegacyConservativeW float64
	BaseW               float64
	KWeight             float64
	DWeight             float64
	PWeight             float64
	XWeight             float64
	EWeight             float64
	WHalfLifeMonths     float64
	BailiffWeight       float64
	ITInternalWeight    float64
	SigmoidMidpoint     float64
	SigmoidWidth        float64
	// MayorOpenW is the W at/below which IT's grip on the Judicial loosens
	// (and the Mayor's influence modifier opens). It doubles as the "low W"
	// bar for PERSONAL_RULE qualification.
	MayorOpenW float64
	// MayorReturnW is the W at/above which the dark order re-asserts the
	// Judicial (contested state returns to IT control).
	MayorReturnW float64
	// MayorMinLegitimacy floors both the Mayor's influence modifier and the
	// PERSONAL_RULE legitimacy read.
	MayorMinLegitimacy float64
	// MayorEntryMonths / MayorReturnMonths are the sustained-month windows for
	// the IT→SWING and SWING→IT regime transitions.
	MayorEntryMonths  int
	MayorReturnMonths int
	// PersonalRuleMinObedience is the police-obedience floor for PERSONAL_RULE.
	// TODO(balance): mirrors the pre-ruling 0.60 attachment-committed bar; the
	// ruling moves that bar from Judicial attachment to police obedience and
	// leaves the constant for sim tuning.
	PersonalRuleMinObedience    float64
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
		MayorEntryMonths: 2, MayorReturnMonths: 3, PersonalRuleMinObedience: 0.60,
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
	// G12: IT/Pact form the attachment split (summing to 1); the Mayor field
	// starts as a small influence modifier, not an attachment share.
	silo.Authority = model.ShadowAuthority{
		Initialized: true, InitializationSource: source, W: w,
		PoliceObedience: silo.MayorLegitimacy(),
		Judicial:        model.JudicialAttachment{IT: 0.90, Mayor: 0.10, Pact: 0.10, Regime: "IT"},
	}
	j := &silo.Authority.Judicial
	j.TargetIT, j.TargetMayor, j.TargetPact = j.IT, j.Mayor, j.Pact
	s := 1 / (1 + math.Exp(-(w-cfg.SigmoidMidpoint)/cfg.SigmoidWidth))
	j.CommandIT = j.IT * s
	j.CommandMayor = j.Mayor
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
// Exposure is the belief-derived conspiracy exposure high-water mark (§3.3),
// NOT safeguard risk; public-evidence Q and the independent-fact count come
// from the judicial belief layer and the public-evidence registry (§3.7).
func AdvanceAuthorityMonthly(silo *model.Silo) {
	if silo == nil {
		return
	}
	info := ComputeInformationEnvironment(silo)
	judicial := &silo.Authority.Judicial
	judicial.PublicEvidenceQ = info.PublicEvidenceQ
	judicial.IndependentPublicFacts = info.QualifyingFacts
	judicial.PublicFactKeys = info.QualifyingFactIDs
	if info.QualifyingFacts > 0 {
		// Structural qualification: public evidence exists. The numeric gates
		// (W/Q/fact count) still decide the pact transition.
		judicial.PactQualified = true
	}
	// §3.2/§5.1：K 只来自二元联盟 (法警 + IT 内部) 的维护忠诚；Police、Mayor
	// 幕僚与其他部门不进入 K。
	InitializeItAuthority(silo)
	alliance := &silo.ItAuthority
	// §4 人事连续性 P：IT 内部倒戈与法警指挥链断裂是负向人事压力
	// (TODO(balance): 门槛与斜率为 sim 起点)。
	personnel := 0.0
	if alliance.ITInternalLoyalty < 0.4 {
		personnel -= (0.4 - alliance.ITInternalLoyalty) * 2.5
	}
	if alliance.BailiffLoyalty < 0.3 {
		personnel -= (0.3 - alliance.BailiffLoyalty) * 2.0
	}
	inputs := AuthorityInputs{
		BailiffLoyalty:    clamp01(alliance.BailiffLoyalty + alliance.BailiffFear),
		ITInternalLoyalty: clamp01(alliance.ITInternalLoyalty + alliance.ITInternalFear),
		Exposure:          info.Exposure,
		Delivery:          1 - 2*professionPanic(silo, "Judicial"),
		Personnel:         personnel,
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

	// §3.2 冲击队列 Σδe：只消费一次，单月合计 cap ±12。
	if len(a.Shocks) > 0 {
		shock := 0.0
		for _, delta := range a.Shocks {
			shock += delta
		}
		a.Shocks = nil
		shock = clamp(shock, -12, 12)
		a.W = clamp(a.W+shock, 0, 100)
	}

	l := silo.MayorLegitimacy()
	a.PoliceObedience = clamp01(a.PoliceObedience + cfg.PoliceObedienceResponseRate*(l-a.PoliceObedience))
	advanceJudicialAttachment(silo, cfg)
	mirrorAuthorityRelations(silo)
}

// advanceJudicialAttachment implements the G12 ruling: the Judicial attaches
// only to IT or to the pact. The Mayor never owns it — the Mayor keeps an
// influence modifier (pressure / replacement / per-case relaxation) that never
// feeds Regime and stays out of the IT/pact split. Regime moves IT → SWING
// when IT's grip loosens (sustained low W) and SWING → IT when the dark order
// re-asserts itself (sustained high W); the pact lock is a one-way fact.
func advanceJudicialAttachment(silo *model.Silo, cfg AuthorityConfig) {
	a := &silo.Authority
	j := &a.Judicial
	l := silo.MayorLegitimacy()

	// Legacy saves may still carry the pre-ruling MAYOR regime; normalize it
	// into the contested state on sight (no save-version bump this round).
	if j.Regime == "MAYOR" {
		j.Regime = "SWING"
	}

	if a.W <= cfg.MayorOpenW {
		j.MayorLowMonths++
	} else {
		j.MayorLowMonths = 0
	}
	if a.W >= cfg.MayorReturnW {
		j.MayorHighMonths++
	} else {
		j.MayorHighMonths = 0
	}
	if j.Regime != "PACT" {
		if j.Regime == "IT" && j.MayorLowMonths >= cfg.MayorEntryMonths {
			j.Regime = "SWING"
			j.MayorHighMonths = 0
		} else if j.Regime == "SWING" && j.MayorHighMonths >= cfg.MayorReturnMonths {
			j.Regime = "IT"
			j.MayorLowMonths = 0
		}
	}

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
	var targetIT, targetPact float64
	switch {
	case j.PactLocked:
		targetPact = 1
	case !pactOpen:
		targetIT = 1
	default:
		zIT := s
		zPact := 1.5 * f * j.PublicEvidenceQ
		if z := zIT + zPact; z > 0 {
			targetIT, targetPact = zIT/z, zPact/z
		} else {
			targetIT = 1
		}
	}

	// Mayor influence modifier (not an attachment share).
	targetMayor := cfg.MayorHighWCap * l
	if a.W <= cfg.MayorOpenW && l >= cfg.MayorMinLegitimacy {
		targetMayor = f * mayorMultiplier
	}

	migration := cfg.AttachmentInertia
	j.IT = (1-migration)*j.IT + migration*targetIT
	j.Pact = (1-migration)*j.Pact + migration*targetPact
	j.Mayor = (1-migration)*j.Mayor + migration*targetMayor
	normalizeAttachment(j)
	j.TargetIT, j.TargetMayor, j.TargetPact = targetIT, targetMayor, targetPact
	j.CommandIT = j.IT * s
	j.CommandMayor = j.Mayor
	j.CommandPact = j.Pact * j.PublicEvidenceQ
	if j.PactLocked {
		j.CommandPact = j.Pact
	}
}

// normalizeAttachment keeps the IT/pact attachment split summing to 1. The
// Mayor field is an influence modifier (G12) and is deliberately excluded from
// the split.
func normalizeAttachment(j *model.JudicialAttachment) {
	j.Mayor = clamp01(j.Mayor)
	j.IT = clamp01(j.IT)
	j.Pact = clamp01(j.Pact)
	sum := j.IT + j.Pact
	if sum <= 0 {
		j.IT, j.Pact = 1, 0
		return
	}
	j.IT /= sum
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

// resetJudicialForNewMayor drops the outgoing holder's influence (G12: there
// is no Mayor attachment share to release) and returns the Judicial to the
// contested state unless the pact already locked it — the charter lock is a
// one-way fact and survives Mayor succession.
func resetJudicialForNewMayor(silo *model.Silo) {
	j := &silo.Authority.Judicial
	j.Mayor *= 0.25
	if !j.PactLocked {
		j.Regime = "SWING"
	}
	j.MayorLowMonths = 0
	j.MayorHighMonths = 0
	normalizeAttachment(j)
}

func sortedResidents(silo *model.Silo) []model.Resident {
	residents := append([]model.Resident(nil), silo.Residents...)
	sort.SliceStable(residents, func(i, j int) bool { return residents[i].ID < residents[j].ID })
	return residents
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

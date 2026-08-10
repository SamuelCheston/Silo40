package engine

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"

	"silo40/internal/model"
)

func initResidentsAndFactions(silo *model.Silo) {
	if silo == nil {
		return
	}
	silo.Cohorts = initPopulationCohorts(silo)
	refreshProfessionPopulationFromCohorts(silo)
	silo.Residents = initKeyResidents(silo)
	RebuildImplicitFactions(silo)
}

func initPopulationCohorts(silo *model.Silo) []model.PopulationCohort {
	cohorts := make([]model.PopulationCohort, 0, len(silo.Professions)*3)
	nextID := uint(1)
	for i := range silo.Professions {
		prof := &silo.Professions[i]
		splits := initialCohortSplits(prof.Population)
		for idx, count := range splits {
			if count <= 0 {
				continue
			}
			loyalty := clamp01(0.42 + rand.Float64()*0.28 + classBias(prof.ClassType, 0.08) + float64(idx-1)*0.08)
			influence := clamp01(float64(prof.PowerLevel)/10.0 + classBias(prof.ClassType, 0.10) + rand.Float64()*0.12 - float64(idx)*0.03)
			ideologies := map[string]float64{
				model.IdeologyProForeign: clamp01(prof.Ideologies[model.IdeologyProForeign] + (rand.Float64()-0.5)*0.18 + float64(idx-1)*0.05),
				model.IdeologyDemocracy:  clamp01(prof.Ideologies[model.IdeologyDemocracy] + (rand.Float64()-0.5)*0.16 + float64(idx-1)*0.04),
			}
			cohort := model.PopulationCohort{
				ID:                 nextID,
				SiloID:             silo.ID,
				ProfessionID:       prof.ID,
				Name:               fmt.Sprintf("%s Cohort %d", prof.Name, idx+1),
				Count:              count,
				HomeZone:           prof.Zone,
				Loyalty:            loyalty,
				Influence:          influence,
				ActionPoints:       18 + rand.Float64()*20,
				PoliticalPrestige:  influence * 20,
				OrganizationFactor: 0.85 + rand.Float64()*0.35,
				PanicSensitivity:   0.85 + rand.Float64()*0.30,
				Ideologies:         ideologies,
				KnownFragments:     initResidentFragments(prof.Name),
			}
			cohort.Tags = buildCohortTags(&cohort, prof, silo)
			cohorts = append(cohorts, cohort)
			nextID++
		}
	}
	return cohorts
}

func initialCohortSplits(population int) []int {
	switch {
	case population <= 4:
		return []int{population}
	case population <= 60:
		first := int(math.Round(float64(population) * 0.6))
		if first <= 0 {
			first = 1
		}
		return []int{first, population - first}
	default:
		first := int(math.Round(float64(population) * 0.5))
		second := int(math.Round(float64(population) * 0.3))
		third := population - first - second
		return []int{first, second, third}
	}
}

func initKeyResidents(silo *model.Silo) []model.Resident {
	if silo == nil {
		return nil
	}
	residents := make([]model.Resident, 0, len(silo.Cohorts)+len(silo.Professions))
	nextID := uint(1)
	for i := range silo.Cohorts {
		cohort := &silo.Cohorts[i]
		prof := profByID(silo, cohort.ProfessionID)
		if prof == nil {
			continue
		}
		resident := model.Resident{
			ID:                 nextID,
			SiloID:             silo.ID,
			Name:               fmt.Sprintf("%s Delegate %02d", prof.Name, i+1),
			CohortID:           uintPtr(cohort.ID),
			ProfessionID:       prof.ID,
			Profession:         prof.Name,
			HomeFloor:          randomFloorForZone(prof.Zone),
			Loyalty:            cohort.Loyalty,
			Ideologies:         copyIdeologies(cohort.Ideologies),
			Influence:          clamp01(cohort.Influence + rand.Float64()*0.08),
			ActionPoints:       cohort.ActionPoints,
			SuspicionLevel:     0,
			PoliticalPrestige:  cohort.PoliticalPrestige,
			PropagandaLevel:    0,
			OrganizationFactor: cohort.OrganizationFactor,
			KnownFragments:     append([]string{}, cohort.KnownFragments...),
			Relations:          initResidentRelations(silo, prof),
			Alive:              true,
		}
		resident.Tags = buildResidentTags(&resident, prof, silo)
		residents = append(residents, resident)
		nextID++
	}
	return residents
}

func initResidentFragments(profession string) []string {
	var fragments []string
	for _, frag := range model.ALL_FRAGMENTS {
		if strings.HasPrefix(frag, profession+"_") && rand.Float64() < 0.18 {
			fragments = append(fragments, frag)
		}
	}
	if len(fragments) == 0 {
		for _, frag := range model.ALL_FRAGMENTS {
			if strings.HasPrefix(frag, profession+"_") {
				return []string{frag}
			}
		}
	}
	return fragments
}

func initResidentRelations(silo *model.Silo, prof *model.Profession) map[string]float64 {
	relations := map[string]float64{}
	for i := range silo.Professions {
		other := &silo.Professions[i]
		base := prof.Relations[other.Name]
		if other.Name == prof.Name && base == 0 {
			base = 0.35
		}
		drift := (rand.Float64() - 0.5) * 0.12
		relations[other.Name] = clamp01(base + drift)
	}
	return relations
}

func classBias(classType string, delta float64) float64 {
	if classType == "ELITE" {
		return delta
	}
	return -delta * 0.35
}

func randomFloorForZone(zone string) int {
	switch zone {
	case "Upper":
		return 1 + rand.Intn(30)
	case "Mid":
		return 31 + rand.Intn(60)
	case "Lower":
		return 91 + rand.Intn(54)
	default:
		return 1 + rand.Intn(144)
	}
}

func skillTagForProfession(name string) string {
	switch name {
	case "Mayor":
		return "skill:governance"
	case "Judicial":
		return "skill:law"
	case "IT":
		return "skill:systems"
	case "Police":
		return "skill:security"
	case "Medical":
		return "skill:care"
	case "Supply":
		return "skill:logistics"
	case "Mechanical":
		return "skill:maintenance"
	case "Mines":
		return "skill:extraction"
	case "Agricultural":
		return "skill:growth"
	default:
		return "skill:general"
	}
}

func stanceTag(loyalty, ideology float64) string {
	switch {
	case loyalty >= 0.7 && ideology < 0.35:
		return "stance:order"
	case ideology >= 0.7:
		return "stance:truth"
	case loyalty < 0.35:
		return "stance:reform"
	default:
		return "stance:survival"
	}
}

func driveTag(resident *model.Resident, prof *model.Profession, silo *model.Silo) string {
	switch {
	case resident.Influence >= 0.75:
		return "drive:ambition"
	case prof.PanicValue >= 0.5 || silo.Cohesion < 0.45:
		return "drive:stability"
	case prof.ClassType == "COMMONER":
		return "drive:solidarity"
	default:
		return "drive:care"
	}
}

func buildResidentTags(resident *model.Resident, prof *model.Profession, silo *model.Silo) []string {
	tags := []string{
		"profession:" + strings.ToLower(prof.Name),
		"class:" + strings.ToLower(prof.ClassType),
		"zone:" + strings.ToLower(prof.Zone),
		skillTagForProfession(prof.Name),
		stanceTag(resident.Loyalty, resident.Ideologies[model.IdeologyProForeign]),
		driveTag(resident, prof, silo),
	}
	if resident.Loyalty >= 0.72 {
		tags = append(tags, "status:trusted")
	} else if resident.Loyalty < 0.30 {
		tags = append(tags, "status:dissident")
	}
	if resident.SuspicionLevel >= 0.45 {
		tags = append(tags, "status:watchlisted")
	}
	if resident.Influence >= 0.78 {
		tags = append(tags, "status:prominent")
	}
	if resident.IsRepresentative {
		tags = append(tags, "role:representative")
	}
	return dedupeStrings(tags)
}

func buildCohortTags(cohort *model.PopulationCohort, prof *model.Profession, silo *model.Silo) []string {
	proxy := model.Resident{
		Loyalty:            cohort.Loyalty,
		Ideologies:         cohort.Ideologies,
		Influence:          cohort.Influence,
		ActionPoints:       cohort.ActionPoints,
		PoliticalPrestige:  cohort.PoliticalPrestige,
		OrganizationFactor: cohort.OrganizationFactor,
	}
	return buildResidentTags(&proxy, prof, silo)
}

func dedupeStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func clamp01(v float64) float64 {
	return math.Max(0, math.Min(1, v))
}

func factionSignatureForResident(resident *model.Resident) (string, []string) {
	stance := "stance:survival"
	drive := "drive:care"
	for _, tag := range resident.Tags {
		if strings.HasPrefix(tag, "stance:") {
			stance = tag
		}
		if strings.HasPrefix(tag, "drive:") {
			drive = tag
		}
	}
	return stance + "|" + drive, []string{stance, drive}
}

func factionSignatureForCohort(cohort *model.PopulationCohort, prof *model.Profession, silo *model.Silo) (string, []string) {
	proxy := model.Resident{
		Loyalty:    cohort.Loyalty,
		Ideologies: cohort.Ideologies,
		Influence:  cohort.Influence,
	}
	proxy.Tags = buildResidentTags(&proxy, prof, silo)
	return factionSignatureForResident(&proxy)
}

func factionNameFromSignature(tags []string) string {
	words := make([]string, 0, len(tags))
	for _, tag := range tags {
		parts := strings.SplitN(tag, ":", 2)
		if len(parts) != 2 {
			continue
		}
		words = append(words, titleToken(parts[1]))
	}
	if len(words) == 0 {
		return "Unaffiliated Circle"
	}
	return strings.Join(words, " ") + " Circle"
}

func titleToken(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func factionCohesion(members []*model.PopulationCohort) float64 {
	if len(members) == 0 {
		return 0
	}
	totalWeight := 0.0
	total := 0.0
	for _, member := range members {
		weight := float64(member.Count)
		totalWeight += weight
		total += member.Loyalty * weight
	}
	if totalWeight == 0 {
		return 0
	}
	return total / totalWeight
}

func representativeScore(resident *model.Resident) float64 {
	return resident.Influence*0.40 +
		resident.Loyalty*0.20 +
		resident.OrganizationFactor*0.15 +
		resident.PoliticalPrestige/100.0*0.15 +
		resident.ActionPoints/100.0*0.10
}

func representativeCohortScore(cohort *model.PopulationCohort) float64 {
	scale := math.Log(float64(cohort.Count) + 1)
	return cohort.Influence*0.45 +
		cohort.Loyalty*0.15 +
		cohort.OrganizationFactor*0.20 +
		cohort.PoliticalPrestige/100.0*0.10 +
		scale*0.10
}

func refreshProfessionPopulationFromCohorts(silo *model.Silo) {
	if silo == nil {
		return
	}
	totalPopulation := 0
	for i := range silo.Professions {
		silo.Professions[i].Population = 0
	}
	for i := range silo.Cohorts {
		cohort := &silo.Cohorts[i]
		totalPopulation += cohort.Count
		if prof := profByID(silo, cohort.ProfessionID); prof != nil {
			prof.Population += cohort.Count
		}
	}
	silo.TotalPopulation = totalPopulation
}

func shouldRemainUnaffiliated(cohort *model.PopulationCohort) bool {
	if cohort.OrganizationFactor < 0.92 &&
		cohort.Loyalty >= 0.35 && cohort.Loyalty <= 0.68 &&
		cohort.Ideologies[model.IdeologyProForeign] < 0.45 &&
		cohort.Ideologies[model.IdeologyDemocracy] < 0.45 {
		return true
	}
	return false
}

// RebuildImplicitFactions 依据 cohort tags 动态聚类派系，并为每个派系选出 cohort/resident 双代表。
func RebuildImplicitFactions(silo *model.Silo) {
	if silo == nil {
		return
	}

	for i := range silo.Residents {
		silo.Residents[i].FactionID = nil
		silo.Residents[i].IsRepresentative = false
		if silo.Residents[i].CohortID == nil {
			continue
		}
		cohort := cohortByID(silo, *silo.Residents[i].CohortID)
		prof := profByID(silo, silo.Residents[i].ProfessionID)
		if cohort == nil || prof == nil {
			continue
		}
		silo.Residents[i].FactionID = cohort.FactionID
		silo.Residents[i].Tags = buildResidentTags(&silo.Residents[i], prof, silo)
	}

	for i := range silo.Cohorts {
		silo.Cohorts[i].FactionID = nil
		prof := profByID(silo, silo.Cohorts[i].ProfessionID)
		if prof == nil {
			continue
		}
		silo.Cohorts[i].Tags = buildCohortTags(&silo.Cohorts[i], prof, silo)
	}

	grouped := map[string][]*model.PopulationCohort{}
	tagIndex := map[string][]string{}
	for i := range silo.Cohorts {
		cohort := &silo.Cohorts[i]
		prof := profByID(silo, cohort.ProfessionID)
		if prof == nil || cohort.Count <= 0 || shouldRemainUnaffiliated(cohort) {
			continue
		}
		signature, tags := factionSignatureForCohort(cohort, prof, silo)
		grouped[signature] = append(grouped[signature], cohort)
		tagIndex[signature] = tags
	}

	signatures := make([]string, 0, len(grouped))
	for signature := range grouped {
		signatures = append(signatures, signature)
	}
	sort.Slice(signatures, func(i, j int) bool {
		leftSize := 0
		rightSize := 0
		for _, cohort := range grouped[signatures[i]] {
			leftSize += cohort.Count
		}
		for _, cohort := range grouped[signatures[j]] {
			rightSize += cohort.Count
		}
		if leftSize == rightSize {
			return signatures[i] < signatures[j]
		}
		return leftSize > rightSize
	})

	factions := make([]model.Faction, 0, len(signatures))
	for idx, signature := range signatures {
		members := grouped[signature]
		tags := tagIndex[signature]
		faction := model.Faction{
			ID:          uint(idx + 1),
			SiloID:      silo.ID,
			Name:        factionNameFromSignature(tags),
			Signature:   signature,
			Tags:        tags,
			MemberCount: 0,
			Cohesion:    factionCohesion(members),
		}

		var repCohort *model.PopulationCohort
		bestScore := -1.0
		totalInfluence := 0.0
		totalWeight := 0.0
		for _, cohort := range members {
			cohort.FactionID = uintPtr(faction.ID)
			score := representativeCohortScore(cohort)
			if score > bestScore {
				bestScore = score
				repCohort = cohort
			}
			weight := float64(cohort.Count)
			faction.MemberCount += cohort.Count
			totalInfluence += cohort.Influence * weight
			totalWeight += weight
		}
		if totalWeight > 0 {
			faction.Influence = totalInfluence / totalWeight
		}
		if repCohort != nil {
			faction.RepresentativeCohortID = uintPtr(repCohort.ID)
			rep := ensureRepresentativeResident(silo, repCohort)
			if rep != nil {
				rep.IsRepresentative = true
				rep.FactionID = uintPtr(faction.ID)
				if prof := profByID(silo, rep.ProfessionID); prof != nil {
					rep.Tags = buildResidentTags(rep, prof, silo)
				}
				faction.RepresentativeResidentID = rep.ID
				faction.RepresentativeName = rep.Name
			} else {
				faction.RepresentativeName = repCohort.Name
			}
		}
		factions = append(factions, faction)
	}

	for i := range silo.Residents {
		resident := &silo.Residents[i]
		if resident.CohortID == nil {
			continue
		}
		if cohort := cohortByID(silo, *resident.CohortID); cohort != nil {
			resident.FactionID = cohort.FactionID
		}
	}

	silo.Factions = factions
}

func ensureRepresentativeResident(silo *model.Silo, cohort *model.PopulationCohort) *model.Resident {
	if silo == nil || cohort == nil {
		return nil
	}
	for i := range silo.Residents {
		if silo.Residents[i].CohortID != nil && *silo.Residents[i].CohortID == cohort.ID && silo.Residents[i].Alive {
			return &silo.Residents[i]
		}
	}
	prof := profByID(silo, cohort.ProfessionID)
	if prof == nil {
		return nil
	}
	nextID := uint(len(silo.Residents) + 1)
	resident := model.Resident{
		ID:                 nextID,
		SiloID:             silo.ID,
		Name:               fmt.Sprintf("%s Speaker %02d", prof.Name, nextID),
		CohortID:           uintPtr(cohort.ID),
		ProfessionID:       prof.ID,
		Profession:         prof.Name,
		HomeFloor:          randomFloorForZone(prof.Zone),
		Loyalty:            cohort.Loyalty,
		Ideologies:         copyIdeologies(cohort.Ideologies),
		Influence:          clamp01(cohort.Influence + 0.08),
		ActionPoints:       cohort.ActionPoints,
		PoliticalPrestige:  cohort.PoliticalPrestige,
		OrganizationFactor: cohort.OrganizationFactor,
		KnownFragments:     append([]string{}, cohort.KnownFragments...),
		Relations:          initResidentRelations(silo, prof),
		Alive:              true,
	}
	resident.Tags = buildResidentTags(&resident, prof, silo)
	silo.Residents = append(silo.Residents, resident)
	return &silo.Residents[len(silo.Residents)-1]
}

func updatePopulationCohorts(silo *model.Silo, deltaYears float64) {
	if silo == nil || deltaYears <= 0 {
		return
	}
	for i := range silo.Cohorts {
		cohort := &silo.Cohorts[i]
		prof := profByID(silo, cohort.ProfessionID)
		if prof == nil || cohort.Count <= 0 {
			continue
		}
		loyaltyTarget := clamp01(silo.Legitimacy - prof.PanicValue*0.35*cohort.PanicSensitivity + classBias(prof.ClassType, 0.08))
		influenceTarget := clamp01(float64(prof.PowerLevel)/10.0 + classBias(prof.ClassType, 0.10))
		organizationTarget := clamp01(0.55 + (1.0-silo.Rebellion)*0.20 + prof.Productivity*0.20)

		cohort.Loyalty += (loyaltyTarget - cohort.Loyalty) * 0.45 * deltaYears
		for key := range cohort.Ideologies {
			profVal := prof.Ideologies[key]
			cohortVal := cohort.Ideologies[key]
			target := clamp01((cohortVal + profVal + silo.Rebellion*0.15) / 2.15)
			cohort.Ideologies[key] += (target - cohortVal) * 0.35 * deltaYears
		}
		cohort.Influence += (influenceTarget - cohort.Influence) * 0.20 * deltaYears
		cohort.OrganizationFactor += (organizationTarget - cohort.OrganizationFactor) * 0.22 * deltaYears
		cohort.ActionPoints = math.Min(70, cohort.ActionPoints+6*deltaYears)
		cohort.PoliticalPrestige = math.Max(0, cohort.PoliticalPrestige+(cohort.Influence*18-cohort.PoliticalPrestige)*0.30*deltaYears)
		cohort.Tags = buildCohortTags(cohort, prof, silo)
	}
}

func updateKeyResidents(silo *model.Silo, deltaYears float64) {
	if silo == nil || deltaYears <= 0 {
		return
	}
	for i := range silo.Residents {
		resident := &silo.Residents[i]
		if !resident.Alive || resident.CohortID == nil {
			continue
		}
		cohort := cohortByID(silo, *resident.CohortID)
		prof := profByID(silo, resident.ProfessionID)
		if cohort == nil || prof == nil {
			continue
		}
		resident.Loyalty += (cohort.Loyalty - resident.Loyalty) * 0.55 * deltaYears
		for key := range resident.Ideologies {
			target := cohort.Ideologies[key]
			resident.Ideologies[key] += (target - resident.Ideologies[key]) * 0.45 * deltaYears
		}
		resident.Influence += (cohort.Influence - resident.Influence) * 0.30 * deltaYears
		resident.ActionPoints = math.Min(70, resident.ActionPoints+6*deltaYears)
		resident.PoliticalPrestige = math.Max(0, resident.PoliticalPrestige+(cohort.PoliticalPrestige-resident.PoliticalPrestige)*0.35*deltaYears)
		if resident.IsRepresentative {
			resident.Influence = clamp01(resident.Influence + 0.04*deltaYears)
		}
		resident.Tags = buildResidentTags(resident, prof, silo)
	}
}

func applyPopulationDeathsToCohorts(silo *model.Silo, deaths int) {
	if silo == nil || deaths <= 0 || len(silo.Cohorts) == 0 {
		return
	}
	total := 0
	for _, cohort := range silo.Cohorts {
		total += cohort.Count
	}
	if total <= 0 {
		return
	}
	remainingDeaths := deaths
	for i := range silo.Cohorts {
		cohort := &silo.Cohorts[i]
		if cohort.Count <= 0 {
			continue
		}
		loss := int(math.Floor(float64(deaths) * float64(cohort.Count) / float64(total)))
		if loss > cohort.Count {
			loss = cohort.Count
		}
		cohort.Count -= loss
		remainingDeaths -= loss
	}
	for remainingDeaths > 0 {
		progressed := false
		for i := range silo.Cohorts {
			if silo.Cohorts[i].Count > 0 {
				silo.Cohorts[i].Count--
				remainingDeaths--
				progressed = true
				if remainingDeaths == 0 {
					break
				}
			}
		}
		if !progressed {
			break
		}
	}
	pruned := silo.Cohorts[:0]
	for i := range silo.Cohorts {
		if silo.Cohorts[i].Count > 0 {
			pruned = append(pruned, silo.Cohorts[i])
		}
	}
	silo.Cohorts = pruned
	refreshProfessionPopulationFromCohorts(silo)
}

func getFactionRepresentatives(silo *model.Silo) []*model.Resident {
	if silo == nil {
		return nil
	}
	reps := make([]*model.Resident, 0, len(silo.Factions))
	for i := range silo.Factions {
		repID := silo.Factions[i].RepresentativeResidentID
		if repID == 0 {
			continue
		}
		for j := range silo.Residents {
			if silo.Residents[j].ID == repID && silo.Residents[j].Alive {
				reps = append(reps, &silo.Residents[j])
				break
			}
		}
	}
	return reps
}

func cohortByID(silo *model.Silo, id uint) *model.PopulationCohort {
	for i := range silo.Cohorts {
		if silo.Cohorts[i].ID == id {
			return &silo.Cohorts[i]
		}
	}
	return nil
}

func copyIdeologies(src map[string]float64) map[string]float64 {
	if src == nil {
		return map[string]float64{}
	}
	dst := make(map[string]float64, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func uintPtr(v uint) *uint {
	return &v
}

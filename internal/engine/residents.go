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
	silo.Residents = initResidents(silo)
	RebuildImplicitFactions(silo)
}

func initResidents(silo *model.Silo) []model.Resident {
	residents := make([]model.Resident, 0, silo.TotalPopulation)
	nextID := uint(1)
	for i := range silo.Professions {
		prof := &silo.Professions[i]
		for idx := 0; idx < prof.Population; idx++ {
			loyalty := clamp01(0.42 + rand.Float64()*0.36 + classBias(prof.ClassType, 0.08))
			ideologyProForeign := clamp01(prof.Ideologies[model.IdeologyProForeign] + (rand.Float64()-0.5)*0.30)
			ideologyDemocracy := clamp01(prof.Ideologies[model.IdeologyDemocracy] + (rand.Float64()-0.5)*0.20)
			influence := clamp01(float64(prof.PowerLevel)/10.0 + classBias(prof.ClassType, 0.12) + rand.Float64()*0.18)
			resident := model.Resident{
				ID:           nextID,
				SiloID:       silo.ID,
				Name:         fmt.Sprintf("%s Resident %04d", prof.Name, idx+1),
				ProfessionID: prof.ID,
				Profession:   prof.Name,
				HomeFloor:    randomFloorForZone(prof.Zone),
				Loyalty:      loyalty,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: ideologyProForeign,
					model.IdeologyDemocracy:  ideologyDemocracy,
				},
				Influence:          influence,
				ActionPoints:       20 + rand.Float64()*25,
				SuspicionLevel:     0,
				PoliticalPrestige:  influence * 25,
				PropagandaLevel:    0,
				OrganizationFactor: 0.9 + rand.Float64()*0.35,
				KnownFragments:     initResidentFragments(prof.Name),
				Relations:          initResidentRelations(silo, prof),
				Alive:              true,
			}
			resident.Tags = buildResidentTags(&resident, prof, silo)
			residents = append(residents, resident)
			nextID++
		}
	}
	return residents
}

func initResidentFragments(profession string) []string {
	var fragments []string
	for _, frag := range model.ALL_FRAGMENTS {
		if strings.HasPrefix(frag, profession+"_") && rand.Float64() < 0.12 {
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

func factionCohesion(members []*model.Resident) float64 {
	if len(members) == 0 {
		return 0
	}
	total := 0.0
	for _, member := range members {
		total += member.Loyalty
	}
	return total / float64(len(members))
}

func representativeScore(resident *model.Resident) float64 {
	return resident.Influence*0.40 +
		resident.Loyalty*0.20 +
		resident.OrganizationFactor*0.15 +
		resident.PoliticalPrestige/100.0*0.15 +
		resident.ActionPoints/100.0*0.10
}

// RebuildImplicitFactions 依据 resident tags 动态聚类派系，并为每个派系选出代表人。
func RebuildImplicitFactions(silo *model.Silo) {
	if silo == nil {
		return
	}

	for i := range silo.Residents {
		silo.Residents[i].FactionID = 0
		silo.Residents[i].IsRepresentative = false
		prof := profByID(silo, silo.Residents[i].ProfessionID)
		if prof == nil {
			continue
		}
		silo.Residents[i].Tags = buildResidentTags(&silo.Residents[i], prof, silo)
	}

	grouped := map[string][]*model.Resident{}
	tagIndex := map[string][]string{}
	for i := range silo.Residents {
		resident := &silo.Residents[i]
		if !resident.Alive {
			continue
		}
		signature, tags := factionSignatureForResident(resident)
		grouped[signature] = append(grouped[signature], resident)
		tagIndex[signature] = tags
	}

	signatures := make([]string, 0, len(grouped))
	for signature := range grouped {
		signatures = append(signatures, signature)
	}
	sort.Slice(signatures, func(i, j int) bool {
		if len(grouped[signatures[i]]) == len(grouped[signatures[j]]) {
			return signatures[i] < signatures[j]
		}
		return len(grouped[signatures[i]]) > len(grouped[signatures[j]])
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
			MemberCount: len(members),
			Cohesion:    factionCohesion(members),
		}

		var rep *model.Resident
		bestScore := -1.0
		totalInfluence := 0.0
		for _, member := range members {
			score := representativeScore(member)
			if score > bestScore {
				bestScore = score
				rep = member
			}
			totalInfluence += member.Influence
		}
		if rep != nil {
			rep.IsRepresentative = true
			rep.FactionID = faction.ID
			if prof := profByID(silo, rep.ProfessionID); prof != nil {
				rep.Tags = buildResidentTags(rep, prof, silo)
			}
			faction.RepresentativeResidentID = rep.ID
			faction.RepresentativeName = rep.Name
		}
		for _, member := range members {
			member.FactionID = faction.ID
		}
		if len(members) > 0 {
			faction.Influence = totalInfluence / float64(len(members))
		}
		factions = append(factions, faction)
	}

	silo.Factions = factions
}

func updateResidentPopulationState(silo *model.Silo, deltaYears float64) {
	if silo == nil || deltaYears <= 0 {
		return
	}

	for i := range silo.Residents {
		resident := &silo.Residents[i]
		if !resident.Alive {
			continue
		}
		prof := profByID(silo, resident.ProfessionID)
		if prof == nil {
			continue
		}

		loyaltyTarget := clamp01(silo.Legitimacy - prof.PanicValue*0.35 + classBias(prof.ClassType, 0.08))
		influenceTarget := clamp01(float64(prof.PowerLevel)/10.0 + classBias(prof.ClassType, 0.10))

		resident.Loyalty += (loyaltyTarget - resident.Loyalty) * 0.45 * deltaYears
		for key := range resident.Ideologies {
			profVal := prof.Ideologies[key]
			resVal := resident.Ideologies[key]
			target := clamp01((resVal + profVal + silo.Rebellion*0.15) / 2.15)
			resident.Ideologies[key] += (target - resVal) * 0.35 * deltaYears
		}
		resident.Influence += (influenceTarget - resident.Influence) * 0.25 * deltaYears
		resident.ActionPoints = math.Min(70, resident.ActionPoints+6*deltaYears)

		if resident.IsRepresentative {
			resident.Influence = clamp01(resident.Influence + 0.04*deltaYears)
		}
		resident.PoliticalPrestige = math.Max(0, resident.PoliticalPrestige+(resident.Influence*18-resident.PoliticalPrestige)*0.30*deltaYears)
	}
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

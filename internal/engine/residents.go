package engine

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"

	"silo40/internal/model"
)

const (
	MIN_FACTION_SIZE            = 200
	BAD_RELATION_THRESHOLD      = 0.05
	NON_FACTION_NAME            = "Unaffiliated"
	RESIDENT_AMBITION_THRESHOLD = 0.62
	RESIDENT_PRESTIGE_THRESHOLD = 15.0
)

type implicitFactionGroup struct {
	signature string
	tags      []string
	cohorts   []*model.PopulationCohort
	profIDs   map[uint]bool
}

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
	cohorts := make([]model.PopulationCohort, 0, len(silo.Professions)*9)
	nextID := uint(1)

	levels := []struct {
		name  string
		level int // 0: Low, 1: Medium, 2: High
	}{
		{"Low", 0},
		{"Medium", 1},
		{"High", 2},
	}

	for i := range silo.Professions {
		prof := &silo.Professions[i]

		// 仅按亲外倾向 (ProForeign) 拆分 bucket
		proForeignVal := prof.Ideologies[model.IdeologyProForeign]

		// Low: [0, 0.3), Medium: [0.3, 0.7), High: [0.7, 1.0]
		var lProb, mProb, hProb float64
		if proForeignVal < 0.3 {
			lProb = 0.7 - proForeignVal
			mProb = 0.3 + proForeignVal
			hProb = 0.0
		} else if proForeignVal > 0.7 {
			lProb = 0.0
			mProb = 1.0 - proForeignVal
			hProb = proForeignVal
		} else {
			distToCenter := math.Abs(proForeignVal - 0.5)
			mProb = 0.8 - distToCenter*2
			sideProb := (1.0 - mProb) / 2
			if proForeignVal > 0.5 {
				hProb = sideProb + (proForeignVal-0.5)*2*sideProb
				lProb = 1.0 - mProb - hProb
			} else {
				lProb = sideProb + (0.5-proForeignVal)*2*sideProb
				hProb = 1.0 - mProb - lProb
			}
		}
		total := lProb + mProb + hProb
		probs := []float64{lProb / total, mProb / total, hProb / total}

		createdAny := false
		for l := 0; l < 3; l++ {
			count := int(math.Round(float64(prof.Population) * probs[l]))
			if count <= 0 {
				continue
			}
			createdAny = true

			profile := []string{}
			if l > 0 {
				profile = append(profile, fmt.Sprintf("%s:%s", model.IdeologyProForeign, levels[l].name))
			}

			if len(profile) == 0 {
				profile = append(profile, "Ideology:Neutral")
			}

			// 初始忠诚度不再受 Bucket 影响，而是基于地堡整体合法性和阶层偏见
			loyalty := clamp01(silo.Legitimacy + classBias(prof.ClassType, 0.08) + (rand.Float64()-0.5)*0.1)
			influence := 0.05
			bucketValues := []float64{0.15, 0.5, 0.85}
			cProForeign := bucketValues[l]

			cohort := model.PopulationCohort{
				ID:              nextID,
				SiloID:          silo.ID,
				ProfessionID:    prof.ID,
				Name:            fmt.Sprintf("%s %s", prof.Name, levels[l].name),
				Count:           count,
				IdeologyProfile: profile,
				HomeZone:        prof.Zone,
				Loyalty:         loyalty,
				Influence:       influence,
				ActionPoints:    20 + rand.Float64()*20,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: cProForeign,
					model.IdeologyLoyalty:    loyalty, // 忠诚度作为状态存入 Ideologies map
				},
				PanicSensitivity: 0.9 + rand.Float64()*0.25,
				KnownFragments:   initResidentFragments(prof.Name),
			}
			cohort.Tags = buildCohortTags(&cohort, prof, silo)
			cohorts = append(cohorts, cohort)
			nextID++
		}

		// 如果人数很少且没能分配到 Cohort
		if !createdAny && prof.Population > 0 {
			loyalty := clamp01(silo.Legitimacy + classBias(prof.ClassType, 0.08))
			cohort := model.PopulationCohort{
				ID:              nextID,
				SiloID:          silo.ID,
				ProfessionID:    prof.ID,
				Name:            fmt.Sprintf("%s Representative", prof.Name),
				Count:           prof.Population,
				IdeologyProfile: []string{"Ideology:Neutral"},
				HomeZone:        prof.Zone,
				Loyalty:         loyalty,
				Influence:       0.05,
				ActionPoints:    20 + rand.Float64()*20,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: proForeignVal,
					model.IdeologyLoyalty:    loyalty,
				},
				PanicSensitivity: 0.9 + rand.Float64()*0.25,
				KnownFragments:   initResidentFragments(prof.Name),
			}
			cohort.Tags = buildCohortTags(&cohort, prof, silo)
			cohorts = append(cohorts, cohort)
			nextID++
		}

		// 补差额逻辑
		currentTotal := 0
		var largestCohort *model.PopulationCohort
		maxCount := -1
		for j := range cohorts {
			if cohorts[j].ProfessionID == prof.ID {
				currentTotal += cohorts[j].Count
				if cohorts[j].Count > maxCount {
					maxCount = cohorts[j].Count
					largestCohort = &cohorts[j]
				}
			}
		}
		if largestCohort != nil && currentTotal != prof.Population {
			largestCohort.Count += (prof.Population - currentTotal)
		}
	}
	return cohorts
}

func initialCohortSplits(population int) []int {
	// 已废弃，改用 initPopulationCohorts 内部的意识形态组合拆分
	return []int{population}
}

func initKeyResidents(silo *model.Silo) []model.Resident {
	if silo == nil {
		return nil
	}
	residents := make([]model.Resident, 0, len(silo.Cohorts)+len(silo.Professions))
	nextID := uint(1)
	for i := range silo.Cohorts {
		cohort := &silo.Cohorts[i]
		prof := getProfessionByID(silo, cohort.ProfessionID)
		if prof == nil {
			continue
		}
		resident := model.Resident{
			ID:           nextID,
			SiloID:       silo.ID,
			Name:         fmt.Sprintf("%s Delegate %02d", prof.Name, i+1),
			CohortID:     uintPtr(cohort.ID),
			ProfessionID: prof.ID,
			Profession:   prof.Name,
			HomeFloor:    randomFloorForZone(prof.Zone),
			Ideologies:   copyIdeologies(cohort.Ideologies),
			Ambition:     generateResidentAmbition(prof, cohort.PoliticalPrestige, cohort.Ideologies),
			// 居民代表初始影响力跟随 Cohort 的“麻木”状态
			Influence:         cohort.Influence,
			ActionPoints:      cohort.ActionPoints,
			SuspicionLevel:    0,
			PoliticalPrestige: cohort.PoliticalPrestige,
			PropagandaLevel:   0,
			KnownFragments:    append([]string{}, cohort.KnownFragments...),
			Relations:         initResidentRelations(silo, prof),
			Alive:             true,
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

func stanceTag(loyalty, proForeign float64) string {
	// 基于三维意识形态综合判断立场
	// 忠诚度 (Loyalty) 为主导维度之一
	// Low: < 0.3 (Extremist), Medium: 0.3-0.7 (Moderate), High: >= 0.7 (Extremist)
	if loyalty >= 0.7 {
		return "stance:order"
	}
	if loyalty < 0.3 {
		return "stance:dissent"
	}

	// 亲外 (ProForeign) 维度
	if proForeign >= 0.7 {
		return "stance:truth"
	}
	if proForeign < 0.3 {
		return "stance:isolation"
	}

	return "stance:survival"
}

func driveTag(resident *model.Resident, prof *model.Profession, silo *model.Silo) string {
	switch {
	case resident.Ambition >= RESIDENT_AMBITION_THRESHOLD:
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
		stanceTag(
			resident.Ideologies[model.IdeologyLoyalty],
			resident.Ideologies[model.IdeologyProForeign],
		),
		driveTag(resident, prof, silo),
	}

	// 忠诚度状态标签
	lVal := resident.Ideologies[model.IdeologyLoyalty]
	if lVal >= 0.7 {
		tags = append(tags, "status:loyalist")
	} else if lVal < 0.3 {
		tags = append(tags, "status:rebel")
	} else if lVal >= 0.5 {
		tags = append(tags, "status:compliant")
	} else {
		tags = append(tags, "status:skeptic")
	}

	if resident.SuspicionLevel >= 0.45 {
		tags = append(tags, "status:watchlisted")
	}
	if resident.Influence >= 0.78 {
		tags = append(tags, "status:prominent")
	}
	if resident.Ambition >= RESIDENT_AMBITION_THRESHOLD {
		tags = append(tags, "status:ambitious")
	}
	if resident.IsRepresentative {
		tags = append(tags, "role:representative")
	}
	return dedupeStrings(tags)
}

func buildCohortTags(cohort *model.PopulationCohort, prof *model.Profession, silo *model.Silo) []string {
	proxy := model.Resident{
		Ideologies:        cohort.Ideologies,
		Influence:         cohort.Influence,
		ActionPoints:      cohort.ActionPoints,
		PoliticalPrestige: cohort.PoliticalPrestige,
	}
	return buildResidentTags(&proxy, prof, silo)
}

func ideologicalIntensity(proForeign float64) float64 {
	return clamp01(math.Abs(proForeign - 0.5))
}

func generateResidentAmbition(prof *model.Profession, prestige float64, ideologies map[string]float64) float64 {
	if prof == nil {
		return 0
	}
	proForeign := ideologies[model.IdeologyProForeign]
	return clamp01(
		0.22 +
			math.Min(prestige, 30)/30.0*0.40 +
			ideologicalIntensity(proForeign)*0.30 +
			rand.Float64()*0.08,
	)
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

func isPotentialLeader(res *model.Resident) bool {
	return res.Ambition >= RESIDENT_AMBITION_THRESHOLD && res.Influence >= 0.25
}

func hasPotentialLeaderInCohort(silo *model.Silo, cohortID uint) bool {
	for i := range silo.Residents {
		res := &silo.Residents[i]
		if res.CohortID != nil && *res.CohortID == cohortID {
			if isPotentialLeader(res) {
				return true
			}
		}
	}
	return false
}

func factionSignatureForResident(resident *model.Resident) (string, []string) {
	profile := make([]string, 0, 1)
	if resident.Ideologies[model.IdeologyProForeign] >= 0.7 {
		profile = append(profile, model.IdeologyProForeign+":High")
	} else if resident.Ideologies[model.IdeologyProForeign] >= 0.3 {
		profile = append(profile, model.IdeologyProForeign+":Medium")
	}

	// 即使 ideology 不够 High，如果该居民本身就是潜在领袖，也允许显示其意识形态标签
	isIdeologyHigh := resident.Ideologies[model.IdeologyProForeign] >= 0.7
	isPolitical := isIdeologyHigh || isPotentialLeader(resident)

	if !isPolitical {
		profile = []string{"ideology:status_quo"}
	}

	sort.Strings(profile)
	drive := "drive:care"
	for _, tag := range resident.Tags {
		if strings.HasPrefix(tag, "drive:") {
			drive = tag
		}
	}
	return strings.Join(profile, "|") + "|" + drive, append(profile, drive)
}

func factionSignatureForCohort(cohort *model.PopulationCohort, _ *model.Profession, silo *model.Silo) (string, []string) {
	// 判断该人群是否政治化
	isIdeologyHigh := cohort.Ideologies[model.IdeologyProForeign] >= 0.7
	isPolitical := isIdeologyHigh || hasPotentialLeaderInCohort(silo, cohort.ID)

	profile := factionProfileTags(cohort.IdeologyProfile, isPolitical)
	sort.Strings(profile)
	sig := strings.Join(profile, "|")

	// 补充 Drive 标签以增加区分度 (从标签中提取)
	drive := "drive:care"
	for _, tag := range cohort.Tags {
		if strings.HasPrefix(tag, "drive:") {
			drive = tag
			break
		}
	}

	fullSig := sig + "|" + drive
	return fullSig, append(append([]string{}, profile...), drive)
}

func factionProfileTags(profile []string, isPolitical bool) []string {
	filtered := make([]string, 0, len(profile))
	for _, tag := range profile {
		if strings.HasPrefix(tag, model.IdeologyLoyalty+":") {
			continue
		}
		if strings.HasPrefix(tag, model.IdeologyProForeign+":") {
			filtered = append(filtered, tag)
		}
	}
	// 如果不是政治化状态，且没有达到 High 门槛，则显示为 status_quo
	if !isPolitical && politicalFormationTier(filtered) == "" {
		return []string{"ideology:status_quo"}
	}
	return filtered
}

func politicalFormationTier(tags []string) string {
	if len(tags) < 1 {
		return ""
	}

	highCount := 0
	politicalCount := 0

	for _, tag := range tags {
		switch {
		case strings.HasPrefix(tag, model.IdeologyProForeign+":"):
			politicalCount++
			if strings.HasSuffix(tag, ":High") {
				highCount++
			}
			// Note: Medium ProForeign no longer triggers formation threshold (requires 70%+)
		}
	}

	if politicalCount < 1 {
		return ""
	}
	if highCount >= 1 {
		return "formation:high"
	}
	// Medium 级别不再直接返回 formation 标识，而是由上层结合领袖状态判定
	return ""
}

func factionNameFromSignature(tags []string) string {
	words := make([]string, 0, len(tags))
	for _, tag := range tags {
		parts := strings.SplitN(tag, ":", 2)
		if len(parts) != 2 {
			continue
		}
		// 转换 Ideology:Level 为 "Level Ideology"
		// 例如 pro_foreign:High -> High Proforeign
		if parts[0] == model.IdeologyProForeign || parts[0] == model.IdeologyLoyalty {
			words = append(words, fmt.Sprintf("%s %s", titleToken(parts[1]), titleToken(parts[0])))
		} else {
			words = append(words, titleToken(parts[1]))
		}
	}
	if len(words) == 0 {
		return "Independent Circle"
	}
	return strings.Join(words, " ") + " Circle"
}

func titleToken(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func factionCohesion(silo *model.Silo, members []*model.PopulationCohort, leader *model.Resident) float64 {
	if len(members) == 0 {
		return 0
	}

	// 1. 成员间关系基础值 (内部和谐度)
	// 计算阵营内不同职业之间的平均关系值
	internalRelSum := 0.0
	relPairs := 0
	profIDs := make(map[uint]bool)
	for _, m := range members {
		profIDs[m.ProfessionID] = true
	}

	ids := make([]uint, 0, len(profIDs))
	for id := range profIDs {
		ids = append(ids, id)
	}

	for i := 0; i < len(ids); i++ {
		p1 := getProfessionByID(silo, ids[i])
		if p1 == nil {
			continue
		}
		for j := i; j < len(ids); j++ {
			p2 := getProfessionByID(silo, ids[j])
			if p2 == nil {
				continue
			}
			if i == j {
				internalRelSum += 1.0 // 自身关系满分
			} else {
				// 双向关系平均
				internalRelSum += (p1.Relations[p2.Name] + p2.Relations[p1.Name]) / 2.0
			}
			relPairs++
		}
	}
	avgInternalRel := 1.0
	if relPairs > 0 {
		avgInternalRel = internalRelSum / float64(relPairs)
	}

	// 2. 领袖威望修正
	leaderBonus := 0.0
	if leader != nil {
		// 威望 (0-100) 转化为 0-0.4 的修正项
		leaderBonus = (leader.PoliticalPrestige / 100.0) * 0.4
	}

	// 3. 组织规模惩罚 (规模越大，凝聚力天然下降)
	totalPop := 0
	for _, m := range members {
		totalPop += m.Count
	}
	sizePenalty := math.Log10(float64(totalPop)/100.0+1.0) * 0.05

	// 最终凝聚力 = (内部关系 * 0.6) + (领袖威望修正) - (规模惩罚)
	cohesion := (avgInternalRel * 0.6) + leaderBonus - sizePenalty

	return clamp01(cohesion)
}

func representativeScore(resident *model.Resident) float64 {
	return resident.Influence*0.40 +
		resident.Ambition*0.20 +
		resident.Ideologies[model.IdeologyLoyalty]*0.20 +
		resident.PoliticalPrestige/100.0*0.15 +
		resident.ActionPoints/100.0*0.10
}

func representativeCohortScore(cohort *model.PopulationCohort) float64 {
	scale := math.Log(float64(cohort.Count) + 1)
	return cohort.PoliticalPrestige/20.0*0.45 +
		cohort.Ideologies[model.IdeologyLoyalty]*0.20 +
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
		if prof := getProfessionByID(silo, cohort.ProfessionID); prof != nil {
			prof.Population += cohort.Count
		}
	}
	silo.TotalPopulation = totalPopulation
}

func shouldRemainUnaffiliated(silo *model.Silo, cohort *model.PopulationCohort) bool {
	if cohort == nil {
		return true
	}
	// 判断该人群是否政治化：
	// 1. 意识形态值达到 70% 门槛
	// 2. 或者人群中产生了具备野心和影响力的潜在领袖
	isIdeologyHigh := cohort.Ideologies[model.IdeologyProForeign] >= 0.7
	isPolitical := isIdeologyHigh || hasPotentialLeaderInCohort(silo, cohort.ID)
	return !isPolitical
}

// RebuildImplicitFactions 依据 cohort 意识形态组合聚类派系，并为每个派系选出 cohort/resident 双代表。
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
		prof := getProfessionByID(silo, silo.Residents[i].ProfessionID)
		if cohort == nil || prof == nil {
			continue
		}
		silo.Residents[i].FactionID = cohort.FactionID
		silo.Residents[i].Tags = buildResidentTags(&silo.Residents[i], prof, silo)
	}

	for i := range silo.Cohorts {
		silo.Cohorts[i].FactionID = nil
		prof := getProfessionByID(silo, silo.Cohorts[i].ProfessionID)
		if prof == nil {
			continue
		}
		silo.Cohorts[i].Tags = buildCohortTags(&silo.Cohorts[i], prof, silo)
	}

	var groups []*implicitFactionGroup
	var unaffiliatedCohorts []*model.PopulationCohort

	for i := range silo.Cohorts {
		cohort := &silo.Cohorts[i]
		prof := getProfessionByID(silo, cohort.ProfessionID)
		if prof == nil || cohort.Count <= 0 {
			continue
		}
		if shouldRemainUnaffiliated(silo, cohort) {
			unaffiliatedCohorts = append(unaffiliatedCohorts, cohort)
			continue
		}
		sig, tags := factionSignatureForCohort(cohort, prof, silo)

		found := false
		for _, g := range groups {
			if g.signature == sig {
				// 检查该阵营中现有的部门是否与当前部门关系恶劣
				isHostile := false
				for existingProfId := range g.profIDs {
					// 同一职业的人群内部不视为敌对
					if existingProfId == prof.ID {
						continue
					}
					existingProf := getProfessionByID(silo, existingProfId)
					if existingProf != nil {
						// 检查双向关系，只要有一方关系低于阈值即视为不合
						rel1 := prof.Relations[existingProf.Name]
						rel2 := existingProf.Relations[prof.Name]
						if rel1 < BAD_RELATION_THRESHOLD || rel2 < BAD_RELATION_THRESHOLD {
							isHostile = true
							break
						}
					}
				}

				if !isHostile {
					g.cohorts = append(g.cohorts, cohort)
					g.profIDs[prof.ID] = true
					found = true
					break
				}
			}
		}

		if !found {
			groups = append(groups, &implicitFactionGroup{
				signature: sig,
				tags:      tags,
				cohorts:   []*model.PopulationCohort{cohort},
				profIDs:   map[uint]bool{prof.ID: true},
			})
		}
	}

	// 预先计算大小并过滤掉人数过少的阵营，将其归入“无阵营”
	validGroups := groups[:0]
	for _, g := range groups {
		size := 0
		for _, c := range g.cohorts {
			size += c.Count
		}
		if size >= MIN_FACTION_SIZE {
			validGroups = append(validGroups, g)
		} else {
			unaffiliatedCohorts = append(unaffiliatedCohorts, g.cohorts...)
		}
	}
	groups = validGroups

	sort.Slice(groups, func(i, j int) bool {
		sizeI := 0
		for _, c := range groups[i].cohorts {
			sizeI += c.Count
		}
		sizeJ := 0
		for _, c := range groups[j].cohorts {
			sizeJ += c.Count
		}
		if sizeI == sizeJ {
			return groups[i].signature < groups[j].signature
		}
		return sizeI > sizeJ
	})

	// 将“无阵营”群体作为一个特殊的组添加到最后
	if len(unaffiliatedCohorts) > 0 {
		groups = append(groups, &implicitFactionGroup{
			signature: "special:unaffiliated",
			tags:      []string{"status:unaffiliated"},
			cohorts:   unaffiliatedCohorts,
			profIDs:   make(map[uint]bool),
		})
	}

	// 统计每个 signature 出现的总次数，用于命名区分
	sigTotalCounts := make(map[string]int)
	for _, g := range groups {
		sigTotalCounts[g.signature]++
	}
	sigCurrentIndex := make(map[string]int)

	// 缓存现有阵营的状态以便保留
	existingFactionStates := make(map[string]struct {
		isPublic bool
		prestige float64
	})
	for _, f := range silo.Factions {
		existingFactionStates[f.Signature] = struct {
			isPublic bool
			prestige float64
		}{isPublic: f.IsPublic, prestige: f.Prestige}
	}

	factions := make([]model.Faction, 0, len(groups))
	for idx, g := range groups {
		members := g.cohorts
		tags := g.tags

		sigCurrentIndex[g.signature]++
		name := factionNameFromSignature(tags)
		if g.signature == "special:unaffiliated" {
			name = NON_FACTION_NAME
		} else if sigTotalCounts[g.signature] > 1 {
			name = fmt.Sprintf("%s (%d)", name, sigCurrentIndex[g.signature])
		}

		faction := model.Faction{
			ID:          uint(idx + 1),
			SiloID:      silo.ID,
			Name:        name,
			Signature:   g.signature,
			Tags:        tags,
			TagStats:    make(map[string]int),
			MemberCount: 0,
		}

		// 保留状态
		if state, ok := existingFactionStates[g.signature]; ok {
			faction.IsPublic = state.isPublic
			faction.Prestige = state.prestige
		}

		// 非政治阵营的影响力设为 0，确保不参与政治指标计算
		totalInfluence := 0.0
		if g.signature != "special:unaffiliated" {
			for _, cohort := range members {
				weight := float64(cohort.Count)
				totalInfluence += cohort.Influence * weight
			}
		}
		faction.Influence = totalInfluence

		var repCohort *model.PopulationCohort
		bestScore := -1.0
		for _, cohort := range members {
			cohort.FactionID = uintPtr(faction.ID)

			// 宏观统计：统计阵营内不同职业和标签的分布
			prof := getProfessionByID(silo, cohort.ProfessionID)
			if prof != nil {
				faction.TagStats["prof:"+prof.Name] += cohort.Count
			}
			for _, t := range cohort.Tags {
				faction.TagStats[t] += cohort.Count
			}

			score := representativeCohortScore(cohort)
			if score > bestScore {
				bestScore = score
				repCohort = cohort
			}
			faction.MemberCount += cohort.Count
		}

		var leader *model.Resident
		if repCohort != nil {
			faction.RepresentativeCohortID = uintPtr(repCohort.ID)
			rep := ensureRepresentativeResident(silo, repCohort)
			if rep != nil {
				rep.IsRepresentative = true
				rep.FactionID = uintPtr(faction.ID)
				if prof := getProfessionByID(silo, rep.ProfessionID); prof != nil {
					rep.Tags = buildResidentTags(rep, prof, silo)
				}
				faction.RepresentativeResidentID = rep.ID
				faction.RepresentativeName = rep.Name
				leader = rep
			} else {
				faction.RepresentativeName = repCohort.Name
			}
		}

		// 计算阵营凝聚力 (传入 Silo, 成员列表, 和领袖)
		faction.Cohesion = factionCohesion(silo, members, leader)

		factions = append(factions, faction)
	}

	// 同步 Resident 的 FactionID
	for i := range silo.Residents {
		resident := &silo.Residents[i]
		if resident.CohortID == nil {
			continue
		}
		if cohort := cohortByID(silo, *resident.CohortID); cohort != nil {
			resident.FactionID = cohort.FactionID
			resident.Ideologies[model.IdeologyLoyalty] += (cohort.Ideologies[model.IdeologyLoyalty] - resident.Ideologies[model.IdeologyLoyalty]) * 0.60
			if prof := getProfessionByID(silo, resident.ProfessionID); prof != nil {
				resident.Tags = buildResidentTags(resident, prof, silo)
			}
		}
	}

	silo.Factions = factions

	// 在阵营构建完成后应用忠诚度动态逻辑 (考虑公开度和威望)
	applyFactionLoyaltyDynamics(silo)
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
	prof := getProfessionByID(silo, cohort.ProfessionID)
	if prof == nil {
		return nil
	}
	nextID := uint(len(silo.Residents) + 1)
	resident := model.Resident{
		ID:           nextID,
		SiloID:       silo.ID,
		Name:         fmt.Sprintf("%s Speaker %02d", prof.Name, nextID),
		CohortID:     uintPtr(cohort.ID),
		ProfessionID: prof.ID,
		Profession:   prof.Name,
		HomeFloor:    randomFloorForZone(prof.Zone),
		Ideologies:   copyIdeologies(cohort.Ideologies),
		Ambition:     generateResidentAmbition(prof, cohort.PoliticalPrestige, cohort.Ideologies),
		// 新代表初始影响力跟随 Cohort 的“麻木”状态
		Influence:         cohort.Influence,
		ActionPoints:      cohort.ActionPoints,
		PoliticalPrestige: cohort.PoliticalPrestige,
		KnownFragments:    append([]string{}, cohort.KnownFragments...),
		Relations:         initResidentRelations(silo, prof),
		Alive:             true,
	}
	resident.Tags = buildResidentTags(&resident, prof, silo)
	silo.Residents = append(silo.Residents, resident)
	return &silo.Residents[len(silo.Residents)-1]
}

func updatePopulationCohorts(silo *model.Silo, deltaYears float64) {
	if silo == nil || deltaYears <= 0 {
		return
	}

	// 职业作为数据结构中的 primary 进行迭代
	for i := range silo.Professions {
		prof := &silo.Professions[i]

		for j := range silo.Cohorts {
			cohort := &silo.Cohorts[j]
			if cohort.ProfessionID != prof.ID || cohort.Count <= 0 {
				continue
			}

			loyaltyTarget := clamp01(silo.Legitimacy - prof.PanicValue*0.35*cohort.PanicSensitivity + classBias(prof.ClassType, 0.08))
			// 威望目标由权力等级、阶层偏见以及影响力（作为修正项）共同决定
			prestigeTarget := (float64(prof.PowerLevel)*1.8 + classBias(prof.ClassType, 2.5) + cohort.Influence*5.0)

			cohort.Ideologies[model.IdeologyLoyalty] += (loyaltyTarget - cohort.Ideologies[model.IdeologyLoyalty]) * 0.45 * deltaYears
			for key := range cohort.Ideologies {
				if key == model.IdeologyLoyalty {
					continue
				}
				profVal := prof.Ideologies[key]
				cohortVal := cohort.Ideologies[key]
				target := clamp01((cohortVal + profVal + silo.Rebellion*0.15) / 2.15)
				cohort.Ideologies[key] += (target - cohortVal) * 0.35 * deltaYears
			}

			// 影响力不再独立增长，而是趋向于权力等级的基础水平
			influenceTarget := clamp01(float64(prof.PowerLevel) / 10.0)
			cohort.Influence += (influenceTarget - cohort.Influence) * 0.15 * deltaYears

			cohort.ActionPoints = math.Min(70, cohort.ActionPoints+6*deltaYears)
			cohort.PoliticalPrestige = math.Max(0, cohort.PoliticalPrestige+(prestigeTarget-cohort.PoliticalPrestige)*0.30*deltaYears)
			cohort.Tags = buildCohortTags(cohort, prof, silo)
		}
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
		prof := getProfessionByID(silo, resident.ProfessionID)
		if cohort == nil || prof == nil {
			continue
		}
		resident.Ideologies[model.IdeologyLoyalty] += (cohort.Ideologies[model.IdeologyLoyalty] - resident.Ideologies[model.IdeologyLoyalty]) * 0.55 * deltaYears
		for key := range resident.Ideologies {
			if key == model.IdeologyLoyalty {
				continue
			}
			target := cohort.Ideologies[key]
			resident.Ideologies[key] += (target - resident.Ideologies[key]) * 0.45 * deltaYears
		}
		// 影响力不再简单同步 Cohort，而是由野心驱动缓慢增长
		// 基础增长率极低，高野心者增长较快
		influenceGrowth := (0.005 + resident.Ambition*0.05) * deltaYears
		resident.Influence = clamp01(resident.Influence + influenceGrowth)

		resident.ActionPoints = math.Min(70, resident.ActionPoints+6*deltaYears)
		resident.PoliticalPrestige = math.Max(0, resident.PoliticalPrestige+(cohort.PoliticalPrestige-resident.PoliticalPrestige)*0.35*deltaYears)
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

func applyFactionLoyaltyDynamics(silo *model.Silo) {
	if silo == nil {
		return
	}
	for i := range silo.Factions {
		faction := &silo.Factions[i]
		if faction.Signature == "special:unaffiliated" {
			continue
		}

		dissentBias := factionDissentBias(faction.Tags)
		orderBias := factionOrderBias(faction.Tags)
		formationBias := factionFormationBias(faction)

		// 遍历所有属于该阵营的 Cohorts
		for j := range silo.Cohorts {
			cohort := &silo.Cohorts[j]
			if cohort.FactionID == nil || *cohort.FactionID != faction.ID {
				continue
			}

			prof := getProfessionByID(silo, cohort.ProfessionID)
			if prof == nil {
				continue
			}

			baseTarget := clamp01(silo.Legitimacy - prof.PanicValue*0.35*cohort.PanicSensitivity + classBias(prof.ClassType, 0.08))
			target := baseTarget

			unrest := 1 - silo.Legitimacy
			target -= formationBias * unrest * (0.12 + 0.12*dissentBias)
			target += formationBias * silo.Legitimacy * 0.08 * orderBias

			cohort.Ideologies[model.IdeologyLoyalty] += (clamp01(target) - cohort.Ideologies[model.IdeologyLoyalty]) * 0.50
			cohort.Tags = buildCohortTags(cohort, prof, silo)
		}
	}
}

func factionFormationBias(f *model.Faction) float64 {
	base := 0.0
	switch politicalFormationTier(f.Tags) {
	case "formation:high":
		base = 1.0
	case "formation:medium":
		base = 0.7
	default:
		// 检查是否有领袖觉醒 (对应 "双向奔赴" 逻辑)
		if f.RepresentativeResidentID > 0 {
			base = 0.6 // 领袖触发的初始组织力稍低
		}
	}

	// 如果没有公开且威望低，组织力显著下降（NPC 感知不到，难以形成合力）
	if !f.IsPublic {
		// 威望可以恢复一部分组织力 (0.4 + prestige/50)
		base *= clamp01(0.4 + f.Prestige/50.0)
	}

	return clamp01(base)
}

const (
	FactionVisibilityHidden  = 0
	FactionVisibilityAware   = 1
	FactionVisibilityVisible = 2
)

// GetFactionVisibilityLevel 判定一个观测者（特工或 NPC）对目标阵营的感知等级
func GetFactionVisibilityLevel(silo *model.Silo, viewerProfID uint, viewerRelations map[string]float64, targetFaction *model.Faction) int {
	if targetFaction == nil {
		return FactionVisibilityHidden
	}
	if targetFaction.IsPublic || targetFaction.Signature == "special:unaffiliated" {
		return FactionVisibilityVisible
	}

	// 1. 如果观测者所属部门就在该阵营中，显然可见
	viewerProf := getProfessionByID(silo, viewerProfID)
	if viewerProf != nil {
		if count, ok := targetFaction.TagStats["prof:"+viewerProf.Name]; ok && count > 0 {
			return FactionVisibilityVisible
		}
	}

	// 2. 检查与阵营内任何部门的关系值
	maxRelation := 0.0
	for tag := range targetFaction.TagStats {
		if strings.HasPrefix(tag, "prof:") {
			targetProfName := tag[5:]
			if val, ok := viewerRelations[targetProfName]; ok {
				if val > maxRelation {
					maxRelation = val
				}
			}
		}
	}

	// 3. 门槛判定
	if maxRelation >= 0.4 {
		return FactionVisibilityVisible
	}
	if maxRelation >= 0.15 {
		return FactionVisibilityAware
	}
	return FactionVisibilityHidden
}

func factionDissentBias(tags []string) float64 {
	score := 0.0
	for _, tag := range tags {
		switch tag {
		case model.IdeologyProForeign + ":High":
			score += 0.6
		case model.IdeologyProForeign + ":Medium":
			score += 0.3
		}
	}
	return clamp01(score)
}

func factionOrderBias(tags []string) float64 {
	score := 0.0
	for _, tag := range tags {
		switch tag {
		case "ideology:status_quo":
			score += 0.7
		case model.IdeologyProForeign + ":Medium":
			score += 0.1
		}
	}
	return clamp01(score)
}

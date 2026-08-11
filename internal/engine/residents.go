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
	cohorts := make([]model.PopulationCohort, 0, len(silo.Professions)*27)
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

		// 统计同一个职业中拥有不同意识形态组合的人口数量
		// 我们按 3^3 = 27 种组合拆分
		pValues := []float64{
			prof.Ideologies[model.IdeologyProForeign],
			prof.Ideologies[model.IdeologyDemocracy],
			prof.Ideologies[model.IdeologyLoyalty],
		}

		// 为每个维度计算概率分布 (简化的正态分布映射)
		// Low: [0, 0.3), Medium: [0.3, 0.7), High: [0.7, 1.0]
		dimProbs := make([][]float64, 3)
		for d := 0; d < 3; d++ {
			val := pValues[d]
			// 简单的启发式分布：
			// 如果 val = 0.5, 大部分在 Medium
			// 如果 val > 0.7, 大部分在 High
			// 如果 val < 0.3, 大部分在 Low
			var lProb, mProb, hProb float64
			if val < 0.3 {
				lProb = 0.7 - val
				mProb = 0.3 + val
				hProb = 0.0
			} else if val > 0.7 {
				lProb = 0.0
				mProb = 1.0 - val
				hProb = val
			} else {
				// 0.3 <= val <= 0.7
				distToCenter := math.Abs(val - 0.5)
				mProb = 0.8 - distToCenter*2 // 0.5时为0.8, 0.3/0.7时为0.4
				sideProb := (1.0 - mProb) / 2
				if val > 0.5 {
					hProb = sideProb + (val-0.5)*2*sideProb
					lProb = 1.0 - mProb - hProb
				} else {
					lProb = sideProb + (0.5-val)*2*sideProb
					hProb = 1.0 - mProb - lProb
				}
			}
			total := lProb + mProb + hProb
			dimProbs[d] = []float64{lProb / total, mProb / total, hProb / total}
		}

		// 遍历 27 种组合
		for l1 := 0; l1 < 3; l1++ {
			for l2 := 0; l2 < 3; l2++ {
				for l3 := 0; l3 < 3; l3++ {
					prob := dimProbs[0][l1] * dimProbs[1][l2] * dimProbs[2][l3]
					count := int(math.Round(float64(prof.Population) * prob))
					
					if count <= 0 {
						continue
					}

					profile := []string{}
					if l1 > 0 { profile = append(profile, fmt.Sprintf("%s:%s", model.IdeologyProForeign, levels[l1].name)) }
					if l2 > 0 { profile = append(profile, fmt.Sprintf("%s:%s", model.IdeologyDemocracy, levels[l2].name)) }
					if l3 > 0 { profile = append(profile, fmt.Sprintf("%s:%s", model.IdeologyLoyalty, levels[l3].name)) }

					// 确保每个人都有意识形态，如果为空则标记为 Neutral
					if len(profile) == 0 {
						profile = append(profile, "Ideology:Neutral")
					}

					loyalty := clamp01(0.45 + rand.Float64()*0.25 + classBias(prof.ClassType, 0.08))
					influence := clamp01(float64(prof.PowerLevel)/10.0 + classBias(prof.ClassType, 0.10) + rand.Float64()*0.10)

					cohort := model.PopulationCohort{
						ID:              nextID,
						SiloID:          silo.ID,
						ProfessionID:    prof.ID,
						Name:            fmt.Sprintf("%s %s-%s-%s", prof.Name, levels[l1].name, levels[l2].name, levels[l3].name),
						Count:           count,
						IdeologyProfile: profile,
						HomeZone:        prof.Zone,
						Loyalty:         loyalty,
						Influence:       influence,
						ActionPoints:    20 + rand.Float64()*20,
						Ideologies: map[string]float64{
							model.IdeologyProForeign: pValues[0],
							model.IdeologyDemocracy:  pValues[1],
							model.IdeologyLoyalty:    pValues[2],
						},
						OrganizationFactor: 0.9 + rand.Float64()*0.30,
						PanicSensitivity:   0.9 + rand.Float64()*0.25,
						KnownFragments:     initResidentFragments(prof.Name),
					}
					cohort.Tags = buildCohortTags(&cohort, prof, silo)
					cohorts = append(cohorts, cohort)
					nextID++
				}
			}
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

func stanceTag(loyalty, proForeign, democracy float64) string {
	// 基于三维意识形态综合判断立场
	// 忠诚度 (Loyalty) 为主导维度之一
	// Low: < 0.3 (Extremist), Medium: 0.3-0.7 (Moderate), High: >= 0.7 (Extremist)
	if loyalty >= 0.7 {
		return "stance:order"
	}
	if loyalty < 0.3 {
		return "stance:dissent"
	}
	
	// 亲外 (ProForeign) 与 民主 (Democracy) 维度
	if proForeign >= 0.7 {
		return "stance:truth"
	}
	if proForeign < 0.3 {
		return "stance:isolation"
	}
	if democracy >= 0.7 {
		return "stance:liberty"
	}
	if democracy < 0.3 {
		return "stance:tradition"
	}
	
	return "stance:survival"
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
		stanceTag(
			resident.Ideologies[model.IdeologyLoyalty],
			resident.Ideologies[model.IdeologyProForeign],
			resident.Ideologies[model.IdeologyDemocracy],
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
	if resident.IsRepresentative {
		tags = append(tags, "role:representative")
	}
	return dedupeStrings(tags)
}

func buildCohortTags(cohort *model.PopulationCohort, prof *model.Profession, silo *model.Silo) []string {
	proxy := model.Resident{
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
	// 签名由意识形态组合决定，确保极端和中立隔离
	// IdeologyProfile 已经包含了 [Ideology:Level] 的信息
	profile := append([]string{}, cohort.IdeologyProfile...)
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

func factionNameFromSignature(tags []string) string {
	words := make([]string, 0, len(tags))
	for _, tag := range tags {
		parts := strings.SplitN(tag, ":", 2)
		if len(parts) != 2 {
			continue
		}
		// 转换 Ideology:Level 为 "Level Ideology"
		// 例如 pro_foreign:High -> High Proforeign
		if parts[0] == model.IdeologyProForeign || parts[0] == model.IdeologyDemocracy || parts[0] == model.IdeologyLoyalty {
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

func factionCohesion(members []*model.PopulationCohort) float64 {
	if len(members) == 0 {
		return 0
	}
	totalWeight := 0.0
	total := 0.0
	for _, member := range members {
		weight := float64(member.Count)
		totalWeight += weight
		total += member.Ideologies[model.IdeologyLoyalty] * weight
	}
	if totalWeight == 0 {
		return 0
	}
	return total / totalWeight
}

func representativeScore(resident *model.Resident) float64 {
	return resident.Influence*0.40 +
		resident.Ideologies[model.IdeologyLoyalty]*0.20 +
		resident.OrganizationFactor*0.15 +
		resident.PoliticalPrestige/100.0*0.15 +
		resident.ActionPoints/100.0*0.10
}

func representativeCohortScore(cohort *model.PopulationCohort) float64 {
	scale := math.Log(float64(cohort.Count) + 1)
	return cohort.Influence*0.45 +
		cohort.Ideologies[model.IdeologyLoyalty]*0.15 +
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
	// 按照用户要求，实际上所有人都会被划入阵营，因此取消“无阵营”逻辑
	return false
}

const (
	MIN_FACTION_SIZE       = 20
	BAD_RELATION_THRESHOLD = 0.3
)

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

	type factionGroup struct {
		signature string
		tags      []string
		cohorts   []*model.PopulationCohort
		profIds   map[uint]bool
	}
	var groups []*factionGroup

	for i := range silo.Cohorts {
		cohort := &silo.Cohorts[i]
		prof := profByID(silo, cohort.ProfessionID)
		if prof == nil || cohort.Count <= 0 {
			continue
		}
		sig, tags := factionSignatureForCohort(cohort, prof, silo)

		found := false
		for _, g := range groups {
			if g.signature == sig {
				// 检查该阵营中现有的部门是否与当前部门关系恶劣
				isHostile := false
				for existingProfId := range g.profIds {
					existingProf := profByID(silo, existingProfId)
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
					g.profIds[prof.ID] = true
					found = true
					break
				}
			}
		}

		if !found {
			groups = append(groups, &factionGroup{
				signature: sig,
				tags:      tags,
				cohorts:   []*model.PopulationCohort{cohort},
				profIds:   map[uint]bool{prof.ID: true},
			})
		}
	}

	// 预先计算大小并过滤掉人数过少的阵营
	validGroups := groups[:0]
	for _, g := range groups {
		size := 0
		for _, c := range g.cohorts {
			size += c.Count
		}
		if size >= MIN_FACTION_SIZE {
			validGroups = append(validGroups, g)
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

	// 统计每个 signature 出现的总次数，用于命名区分
	sigTotalCounts := make(map[string]int)
	for _, g := range groups {
		sigTotalCounts[g.signature]++
	}
	sigCurrentIndex := make(map[string]int)

	factions := make([]model.Faction, 0, len(groups))
	for idx, g := range groups {
		members := g.cohorts
		tags := g.tags
		
		sigCurrentIndex[g.signature]++
		name := factionNameFromSignature(tags)
		if sigTotalCounts[g.signature] > 1 {
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
			Cohesion:    factionCohesion(members),
		}

		var repCohort *model.PopulationCohort
		bestScore := -1.0
		totalInfluence := 0.0
		for _, cohort := range members {
			cohort.FactionID = uintPtr(faction.ID)
			
			// 宏观统计：统计阵营内不同职业和标签的分布
			prof := profByID(silo, cohort.ProfessionID)
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
			weight := float64(cohort.Count)
			faction.MemberCount += cohort.Count
			totalInfluence += cohort.Influence * weight
		}
		faction.Influence = totalInfluence // 阵营总影响力为成员影响力之和
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

	// 同步 Resident 的 FactionID
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

	// 职业作为数据结构中的 primary 进行迭代
	for i := range silo.Professions {
		prof := &silo.Professions[i]

		for j := range silo.Cohorts {
			cohort := &silo.Cohorts[j]
			if cohort.ProfessionID != prof.ID || cohort.Count <= 0 {
				continue
			}

			loyaltyTarget := clamp01(silo.Legitimacy - prof.PanicValue*0.35*cohort.PanicSensitivity + classBias(prof.ClassType, 0.08))
			influenceTarget := clamp01(float64(prof.PowerLevel)/10.0 + classBias(prof.ClassType, 0.10))
			organizationTarget := clamp01(0.55 + (1.0-silo.Rebellion)*0.20 + prof.Productivity*0.20)

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
			cohort.Influence += (influenceTarget - cohort.Influence) * 0.20 * deltaYears
			cohort.OrganizationFactor += (organizationTarget - cohort.OrganizationFactor) * 0.22 * deltaYears
			cohort.ActionPoints = math.Min(70, cohort.ActionPoints+6*deltaYears)
			cohort.PoliticalPrestige = math.Max(0, cohort.PoliticalPrestige+(cohort.Influence*18-cohort.PoliticalPrestige)*0.30*deltaYears)
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
		prof := profByID(silo, resident.ProfessionID)
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

package engine

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"silo40/internal/model"
)

// unaffiliatedSignature identifies the synthetic bucket holding all
// non-political cohorts. Internal key only; never a real faction signature.
const unaffiliatedSignature = "special:unaffiliated"

func initResidentsAndFactions(silo *model.Silo) {
	if silo == nil {
		return
	}
	silo.Cohorts = initPopulationCohorts(silo)
	refreshProfessionPopulationFromCohorts(silo)
	silo.Residents = initKeyResidents(silo)
	MaintainFactions(silo, nil)
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
			loyalty := clamp01(silo.MayorLegitimacy() + classBias(prof.ClassType, 0.08) + (rng.Float64()-0.5)*0.1)
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
				ActionPoints:    20 + rng.Float64()*20,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: cProForeign,
					model.IdeologyLoyalty:    loyalty, // 忠诚度作为状态存入 Ideologies map
				},
				PanicSensitivity: 0.9 + rng.Float64()*0.25,
				KnownFragments:   initResidentFragments(prof.Name),
			}
			cohort.Tags = buildCohortTags(&cohort, prof, silo)
			cohorts = append(cohorts, cohort)
			nextID++
		}

		// 如果人数很少且没能分配到 Cohort
		if !createdAny && prof.Population > 0 {
			loyalty := clamp01(silo.MayorLegitimacy() + classBias(prof.ClassType, 0.08))
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
				ActionPoints:    20 + rng.Float64()*20,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: proForeignVal,
					model.IdeologyLoyalty:    loyalty,
				},
				PanicSensitivity: 0.9 + rng.Float64()*0.25,
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
			Ambition:     generateResidentAmbition(prof, cohort.Ideologies),
			// 居民代表初始影响力跟随 Cohort 的“麻木”状态
			Influence:       cohort.Influence,
			ActionPoints:    cohort.ActionPoints,
			SuspicionLevel:  0,
			PropagandaLevel: 0,
			KnownFragments:  append([]string{}, cohort.KnownFragments...),
			Relations:       initResidentRelations(silo, prof),
			Alive:           true,
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
		if strings.HasPrefix(frag, profession+"_") && rng.Float64() < 0.18 {
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
		drift := (rng.Float64() - 0.5) * 0.12
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
		return 1 + rng.Intn(30)
	case "Mid":
		return 31 + rng.Intn(60)
	case "Lower":
		return 91 + rng.Intn(54)
	default:
		return 1 + rng.Intn(144)
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
	case resident.Ambition >= FactionConfig.AmbitionHighThreshold:
		return "drive:ambition"
	case prof.PanicValue >= FactionConfig.DriveStabilityPanicThreshold || silo.Cohesion < FactionConfig.DriveStabilityCohesionThreshold:
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
	if resident.Ambition >= FactionConfig.AmbitionHighThreshold {
		tags = append(tags, "status:ambitious")
	}
	if resident.IsRepresentative {
		tags = append(tags, "role:representative")
	}
	return dedupeStrings(tags)
}

func buildCohortTags(cohort *model.PopulationCohort, prof *model.Profession, silo *model.Silo) []string {
	proxy := model.Resident{
		Ideologies:   cohort.Ideologies,
		Influence:    cohort.Influence,
		ActionPoints: cohort.ActionPoints,
	}
	return buildResidentTags(&proxy, prof, silo)
}

func ideologicalIntensity(proForeign float64) float64 {
	return clamp01(math.Abs(proForeign - 0.5))
}

// generateResidentAmbition 开局定死的代表野心。输入轴为部门 PowerLevel
// (24 篇 §2)：不再读 cohort 威望——威望派生自代表 ambition，会形成循环依赖。
func generateResidentAmbition(prof *model.Profession, ideologies map[string]float64) float64 {
	if prof == nil {
		return 0
	}
	proForeign := ideologies[model.IdeologyProForeign]
	return clamp01(
		FactionConfig.AmbitionBase +
			math.Min(float64(prof.PowerLevel), 10)/10.0*FactionConfig.AmbitionPowerWeight +
			ideologicalIntensity(proForeign)*FactionConfig.AmbitionIntensityWeight +
			rng.Float64()*FactionConfig.AmbitionNoise,
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
		}
	}

	if highCount >= 1 {
		return "formation:high"
	}
	if politicalCount >= 1 {
		return "formation:medium"
	}
	return ""
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
	// 固定迭代顺序：浮点求和顺序必须确定，否则同种子下凝聚力会出现末位 ULP 漂移。
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

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
		// 威望 (0-100) 转化为 0-CohesionLeaderMaxBonus 的修正项
		leaderBonus = (DerivedPrestige(leader.Ambition, leader.Influence) / 100.0) * FactionConfig.CohesionLeaderMaxBonus
	}

	// 3. 组织规模惩罚 (规模越大，凝聚力天然下降)
	totalPop := 0
	for _, m := range members {
		totalPop += m.Count
	}
	sizePenalty := math.Log10(float64(totalPop)/100.0+1.0) * FactionConfig.CohesionSizePenaltyScale

	// 最终凝聚力 = (内部关系 * 权重) + (领袖威望修正) - (规模惩罚)
	cohesion := (avgInternalRel * FactionConfig.CohesionRelationWeight) + leaderBonus - sizePenalty

	return clamp01(cohesion)
}

// cohortPrestige 聚合单元没有自己的野心/影响力人格：威望借其存活代表居民
// 派生 (24 篇 §2)。无代表时为 0。
func cohortPrestige(silo *model.Silo, cohort *model.PopulationCohort) float64 {
	for i := range silo.Residents {
		res := &silo.Residents[i]
		if res.Alive && res.CohortID != nil && *res.CohortID == cohort.ID {
			return DerivedPrestige(res.Ambition, res.Influence)
		}
	}
	return 0
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

// MaintainFactions 事件驱动派系的维护管线 (24 篇 C7)：派系只能由
// FOUND_FACTION 创立，成员资格持久 (加入/退出见 C9)。本管线不聚类、
// 不产生新派系，只做：无阵营桶兜底、成员统计重算、筹备期解散、
// 领袖死亡接任、cohort 代表补员与归属修复。全程确定性：按切片序
// (ID 升序) 处理，补员掷骰顺序固定。logf 可为 nil (初始化路径)。
func MaintainFactions(silo *model.Silo, logf func(string)) {
	if silo == nil {
		return
	}

	unaffiliatedID := ensureUnaffiliatedFaction(silo)

	// 成员资格持久：只修复悬空引用 (派系已解散/不存在)，不做聚类重分配。
	for i := range silo.Cohorts {
		cohort := &silo.Cohorts[i]
		if cohort.FactionID == nil || factionByID(silo, *cohort.FactionID) == nil {
			cohort.FactionID = uintPtr(unaffiliatedID)
		}
		if prof := getProfessionByID(silo, cohort.ProfessionID); prof != nil {
			cohort.Tags = buildCohortTags(cohort, prof, silo)
		}
	}

	// 逐派系维护：统计重算 → 筹备期解散 → 领袖死亡接任。
	var dissolved []uint
	for i := range silo.Factions {
		f := &silo.Factions[i]
		members := factionMemberCohorts(silo, f.ID)
		recomputeFactionStats(silo, f, members)
		if f.Signature == unaffiliatedSignature {
			continue
		}
		switch {
		case factionGraceExpired(silo, f):
			dissolved = append(dissolved, f.ID)
			if logf != nil {
				logf(fmt.Sprintf("「%s」筹备期届满未凑够 %d 人，宣告解散。", f.Name, FactionConfig.MinFactionSize))
			}
		case !succeedFactionLeader(silo, f):
			dissolved = append(dissolved, f.ID)
			if logf != nil {
				logf(fmt.Sprintf("「%s」领袖死亡且无可接任者，宣告解散。", f.Name))
			}
		}
	}
	for _, id := range dissolved {
		dissolveFaction(silo, id, unaffiliatedID)
	}

	// cohort 代表补员：代表死亡时按 cohort ID 升序重建 (掷骰顺序固定)。
	// 新代表跟随 cohort 归属；既有代表的派系归属持久，不被 cohort 覆盖。
	for i := range silo.Cohorts {
		cohort := &silo.Cohorts[i]
		if cohort.Count <= 0 {
			continue
		}
		rep := ensureRepresentativeResident(silo, cohort)
		if rep != nil && (rep.FactionID == nil || factionByID(silo, *rep.FactionID) == nil) {
			rep.FactionID = uintPtr(*cohort.FactionID)
		}
	}

	// 代表标志以派系引用为准：IsRepresentative = 某派系的 RESIDENT 代表。
	repIDs := make(map[uint]bool)
	for i := range silo.Factions {
		f := &silo.Factions[i]
		if f.RepresentativeKind == string(ActorResident) && f.RepresentativeID > 0 {
			repIDs[f.RepresentativeID] = true
		}
	}
	for i := range silo.Residents {
		r := &silo.Residents[i]
		if r.FactionID != nil && factionByID(silo, *r.FactionID) == nil {
			r.FactionID = uintPtr(unaffiliatedID)
		}
		r.IsRepresentative = r.Alive && repIDs[r.ID]
		// 不挂 cohort 的个体 (18 号正典具名人物) 状态由链条节点权威维护，
		// 其自定义标签不可被派生标签覆写。
		if r.CohortID == nil {
			continue
		}
		if prof := getProfessionByID(silo, r.ProfessionID); prof != nil {
			r.Tags = buildResidentTags(r, prof, silo)
		}
	}
}

// ensureUnaffiliatedFaction 无阵营桶：持久 ID，缺失时以 max+1 新建。
func ensureUnaffiliatedFaction(silo *model.Silo) uint {
	for i := range silo.Factions {
		if silo.Factions[i].Signature == unaffiliatedSignature {
			return silo.Factions[i].ID
		}
	}
	nextID := uint(1)
	for i := range silo.Factions {
		if silo.Factions[i].ID >= nextID {
			nextID = silo.Factions[i].ID + 1
		}
	}
	silo.Factions = append(silo.Factions, model.Faction{
		ID:        nextID,
		SiloID:    silo.ID,
		Name:      FactionConfig.NonFactionName,
		Signature: unaffiliatedSignature,
		Tags:      []string{"status:unaffiliated"},
		TagStats:  map[string]int{},
	})
	return nextID
}

func factionMemberCohorts(silo *model.Silo, factionID uint) []*model.PopulationCohort {
	var members []*model.PopulationCohort
	for i := range silo.Cohorts {
		c := &silo.Cohorts[i]
		if c.FactionID != nil && *c.FactionID == factionID {
			members = append(members, c)
		}
	}
	return members
}

// recomputeFactionStats 从成员 cohort 重算组织统计；身份字段
// (Name/IsPublic/Prestige/Stance/Memory/Founder/代表) 持久不动。
func recomputeFactionStats(silo *model.Silo, f *model.Faction, members []*model.PopulationCohort) {
	f.MemberCount = 0
	f.TagStats = make(map[string]int)
	totalInfluence := 0.0
	for _, cohort := range members {
		f.MemberCount += cohort.Count
		if f.Signature != unaffiliatedSignature {
			totalInfluence += cohort.Influence * float64(cohort.Count)
		}
		if prof := getProfessionByID(silo, cohort.ProfessionID); prof != nil {
			f.TagStats["prof:"+prof.Name] += cohort.Count
		}
		for _, t := range cohort.Tags {
			f.TagStats[t] += cohort.Count
		}
	}
	if f.Signature == unaffiliatedSignature {
		f.Influence = 0
		f.Cohesion = 0
		f.Demands = nil
		return
	}
	f.Influence = totalInfluence
	var leader *model.Resident
	if f.RepresentativeKind == string(ActorResident) {
		if r := residentByID(silo, f.RepresentativeID); r != nil && r.Alive {
			leader = r
		}
	}
	f.Cohesion = factionCohesion(silo, members, leader)
	f.Demands = []string{"demand:" + factionDriveTag(members)}
}

// factionDriveTag 成员主导 drive：人数最多者，平局取 cohort ID 小者。
func factionDriveTag(members []*model.PopulationCohort) string {
	var best *model.PopulationCohort
	for _, m := range members {
		if best == nil || m.Count > best.Count || (m.Count == best.Count && m.ID < best.ID) {
			best = m
		}
	}
	if best != nil {
		for _, tag := range best.Tags {
			if strings.HasPrefix(tag, "drive:") {
				return strings.TrimPrefix(tag, "drive:")
			}
		}
	}
	return "care"
}

// factionGraceExpired 筹备期届满 (成立满 FactionGraceMonths 个月) 且
// 人数不足 MinFactionSize。成立年月缺失 (0) 时不判解散。
func factionGraceExpired(silo *model.Silo, f *model.Faction) bool {
	if f.FoundedYear == 0 || f.MemberCount >= FactionConfig.MinFactionSize {
		return false
	}
	ageMonths := (silo.CurrentYear-f.FoundedYear)*12 + (silo.CurrentMonth - f.FoundedMonth)
	return ageMonths >= FactionConfig.FactionGraceMonths
}

// succeedFactionLeader 领袖死亡/离任时自动接任 (24 篇 §1.5)：派系内
// 存活居民按 SuccessionPrestigeWeight×(prestige/100) + (1−w)×ambition
// 评分，平局取 ID 小者。返回 false = 无可接任者，派系应解散。
// PLAYER/NPC 领袖不死亡，恒有主。
func succeedFactionLeader(silo *model.Silo, f *model.Faction) bool {
	if f.RepresentativeKind == string(ActorPlayer) || f.RepresentativeKind == string(ActorNPC) {
		return true
	}
	if f.RepresentativeKind == string(ActorResident) {
		if r := residentByID(silo, f.RepresentativeID); r != nil && r.Alive &&
			r.FactionID != nil && *r.FactionID == f.ID {
			return true
		}
	}
	var best *model.Resident
	bestScore := -1.0
	for i := range silo.Residents {
		r := &silo.Residents[i]
		if !r.Alive || r.FactionID == nil || *r.FactionID != f.ID {
			continue
		}
		score := FactionConfig.SuccessionPrestigeWeight*(DerivedPrestige(r.Ambition, r.Influence)/100) +
			(1-FactionConfig.SuccessionPrestigeWeight)*r.Ambition
		if score > bestScore || (score == bestScore && best != nil && r.ID < best.ID) {
			best, bestScore = r, score
		}
	}
	if best == nil {
		return false
	}
	f.RepresentativeKind = string(ActorResident)
	f.RepresentativeID = best.ID
	f.RepresentativeName = best.Name
	best.IsRepresentative = true
	return true
}

// dissolveFaction 解散派系：成员 cohort 与居民回无阵营桶，派系除名。
func dissolveFaction(silo *model.Silo, factionID, unaffiliatedID uint) {
	for i := range silo.Cohorts {
		c := &silo.Cohorts[i]
		if c.FactionID != nil && *c.FactionID == factionID {
			c.FactionID = uintPtr(unaffiliatedID)
		}
	}
	for i := range silo.Residents {
		r := &silo.Residents[i]
		if r.FactionID != nil && *r.FactionID == factionID {
			r.FactionID = uintPtr(unaffiliatedID)
			r.IsRepresentative = false
		}
	}
	kept := silo.Factions[:0]
	for _, f := range silo.Factions {
		if f.ID != factionID {
			kept = append(kept, f)
		}
	}
	silo.Factions = kept
}

func residentByID(silo *model.Silo, id uint) *model.Resident {
	for i := range silo.Residents {
		if silo.Residents[i].ID == id {
			return &silo.Residents[i]
		}
	}
	return nil
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
		Ambition:     generateResidentAmbition(prof, cohort.Ideologies),
		// 新代表初始影响力跟随 Cohort 的“麻木”状态
		Influence:      cohort.Influence,
		ActionPoints:   cohort.ActionPoints,
		KnownFragments: append([]string{}, cohort.KnownFragments...),
		Relations:      initResidentRelations(silo, prof),
		Alive:          true,
	}
	resident.Tags = buildResidentTags(&resident, prof, silo)
	silo.Residents = append(silo.Residents, resident)
	return &silo.Residents[len(silo.Residents)-1]
}

// opennessLoyaltyErosion 亲外对忠诚天花板的侵蚀系数。推导见
// updatePopulationCohorts 内的 TODO(balance) 注释。
const opennessLoyaltyErosion = 0.7

// NudgeAllCohortIdeology 给全体存活 cohort 的某一思潮轴叠加固定增量 (夹到
// [0,1])。公开事实登记用：事实成为公共真相的时刻，对全堡思潮做一次方向性
// 推动。确定性：按 cohort 切片顺序迭代，常数量级，不消耗 RNG。
func NudgeAllCohortIdeology(silo *model.Silo, ideology string, delta float64) {
	if silo == nil || ideology == "" || delta == 0 {
		return
	}
	for i := range silo.Cohorts {
		cohort := &silo.Cohorts[i]
		if cohort.Count <= 0 {
			continue
		}
		if cohort.Ideologies == nil {
			cohort.Ideologies = map[string]float64{}
		}
		cohort.Ideologies[ideology] = clampUnit(cohort.Ideologies[ideology] + delta)
	}
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

			// 开放性侵蚀忠诚天花板 (《无人裁决》可达性)：已扎根的可核验事实越多
			// (cohort 亲外越高)，对旧叙事与指挥链的无条件服从就越被溶解。侵蚀项
			// 只随玩家持续投入积累的亲外上升——不投入则亲外停在 ~0.26，忠诚
			// 天花板几乎不受影响，封闭照旧。
			// TODO(balance): 0.7 的推导——覆灭前夕快照门槛为 亲外 ≥ 忠诚。无侵蚀时
			// 忠诚稳在 ~0.9，而流通杠杆现实上限约 0.6，门槛不可达；侵蚀系数取 0.7 时
			// 交叉点出现在 亲外 ≈ (正统性−恐慌项)/(1+0.7) ≈ 0.53–0.59（正统性 0.9–1.0），
			// 持续流通局亲外到 ~0.65 稳定越线；不玩局亲外 ~0.26、忠诚天花板仍 ~0.8+，
			// 差距 ~0.55 远未越线——侵蚀只奖励真实投入。
			loyaltyTarget := clamp01(silo.MayorLegitimacy() - prof.PanicValue*0.35*cohort.PanicSensitivity + classBias(prof.ClassType, 0.08) - opennessLoyaltyErosion*cohort.Ideologies[model.IdeologyProForeign])

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
		f := &silo.Factions[i]
		if f.RepresentativeKind != string(ActorResident) || f.RepresentativeID == 0 {
			continue
		}
		for j := range silo.Residents {
			if silo.Residents[j].ID == f.RepresentativeID && silo.Residents[j].Alive {
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

// ApplyFactionLoyaltyDynamics advances the faction-driven loyalty evolution:
// residents sync toward their cohort loyalty, then political cohorts drift
// toward the faction-adjusted target (formation tier, public status,
// prestige). Both steps are per-month steps time-scaled by deltaYears
// (monthlyTimeScale), so a zero-duration step (deltaYears <= 0) applies
// nothing: instant (0-month) actions must not accelerate political
// evolution. Called by the tick pipeline right after RebuildImplicitFactions.
func ApplyFactionLoyaltyDynamics(silo *model.Silo, deltaYears float64) {
	if silo == nil || deltaYears <= 0 {
		return
	}

	residentStep := monthlyTimeScale(FactionConfig.ResidentLoyaltySyncMonthlyStep, deltaYears)
	for i := range silo.Residents {
		resident := &silo.Residents[i]
		if resident.CohortID == nil {
			continue
		}
		if cohort := cohortByID(silo, *resident.CohortID); cohort != nil {
			resident.Ideologies[model.IdeologyLoyalty] += (cohort.Ideologies[model.IdeologyLoyalty] - resident.Ideologies[model.IdeologyLoyalty]) * residentStep
		}
	}

	dynamicsStep := monthlyTimeScale(FactionConfig.LoyaltyDynamicsMonthlyStep, deltaYears)
	for i := range silo.Factions {
		faction := &silo.Factions[i]
		if faction.Signature == unaffiliatedSignature {
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

			baseTarget := clamp01(silo.MayorLegitimacy() - prof.PanicValue*0.35*cohort.PanicSensitivity + classBias(prof.ClassType, 0.08))
			target := baseTarget

			unrest := 1 - silo.MayorLegitimacy()
			target -= formationBias * unrest * (FactionConfig.UnrestDissentBase + FactionConfig.UnrestDissentExtra*dissentBias)
			target += formationBias * silo.MayorLegitimacy() * FactionConfig.OrderLegitimacyGain * orderBias

			cohort.Ideologies[model.IdeologyLoyalty] += (clamp01(target) - cohort.Ideologies[model.IdeologyLoyalty]) * dynamicsStep
			cohort.Tags = buildCohortTags(cohort, prof, silo)
		}
	}
}

// monthlyTimeScale converts a per-month convergence step into the equivalent
// step for an arbitrary time slice: 1-(1-base)^(12*dt). At dt=1/12 it
// returns base exactly (monthly ticks keep the historical rate); at dt=0 it
// returns 0. The exponential form keeps the effective evolution rate
// independent of how time is sliced.
func monthlyTimeScale(monthlyBase, deltaYears float64) float64 {
	if deltaYears <= 0 {
		return 0
	}
	return 1 - math.Pow(1-monthlyBase, 12*deltaYears)
}

func factionFormationBias(f *model.Faction) float64 {
	base := 0.0
	switch politicalFormationTier(f.Tags) {
	case "formation:high":
		base = FactionConfig.FormationBiasHigh
	case "formation:medium":
		base = FactionConfig.FormationBiasMedium
	default:
		// 检查是否有领袖觉醒 (对应 "双向奔赴" 逻辑)
		if f.RepresentativeID > 0 {
			base = FactionConfig.FormationBiasLeader // 领袖触发的初始组织力稍低
		}
	}

	// 如果没有公开且威望低，组织力显著下降（NPC 感知不到，难以形成合力）
	if !f.IsPublic {
		// 威望可以恢复一部分组织力 (base + prestige/divisor)
		base *= clamp01(FactionConfig.FormationNonPublicBase + f.Prestige/FactionConfig.FormationPrestigeDivisor)
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
	if targetFaction.IsPublic || targetFaction.Signature == unaffiliatedSignature {
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
	if maxRelation >= FactionConfig.VisibilityRelationVisible {
		return FactionVisibilityVisible
	}
	if maxRelation >= FactionConfig.VisibilityRelationAware {
		return FactionVisibilityAware
	}
	return FactionVisibilityHidden
}

func factionDissentBias(tags []string) float64 {
	score := 0.0
	for _, tag := range tags {
		switch tag {
		case model.IdeologyProForeign + ":High":
			score += FactionConfig.DissentBiasHigh
		case model.IdeologyProForeign + ":Medium":
			score += FactionConfig.DissentBiasMedium
		}
	}
	return clamp01(score)
}

func factionOrderBias(tags []string) float64 {
	score := 0.0
	for _, tag := range tags {
		switch tag {
		case "ideology:status_quo":
			score += FactionConfig.OrderBiasStatusQuo
		case model.IdeologyProForeign + ":Medium":
			score += FactionConfig.OrderBiasMedium
		}
	}
	return clamp01(score)
}

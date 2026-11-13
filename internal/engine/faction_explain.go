package engine

// P1-S4 可解释归属：任意 cohort / 居民可产出"为何在此派系"的解释——
// 到本派系质心的特征距离 vs 到次近派系质心的距离 (设计稿 §二/§七)。
// 纯只读旁路：不修改任何状态、不消耗 RNG，行为中性。
//
// 解释如实反映聚类机制：归属由"距离 + 惯性"共同决定，因此存在惯性保留
// (本派系距离略大于次近派系、但因惯性半径留在原派系) 的合法情形——
// 解释中给出 margin 与惯性标记，而不是假装纯距离最优。

import (
	"fmt"
	"math"

	"silo40/internal/model"
)

// AffiliationExplanation 一个对象的归属解释。
type AffiliationExplanation struct {
	SubjectKind   string  `json:"subject_kind"` // cohort / resident
	SubjectID     uint    `json:"subject_id"`
	SubjectName   string  `json:"subject_name"`
	FactionID     uint    `json:"faction_id"`
	FactionName   string  `json:"faction_name"`
	OwnDistance   float64 `json:"own_distance"`    // 到本派系质心的特征距离
	NearestID     uint    `json:"nearest_id"`      // 次近政治派系 ID (0 = 无其他派系)
	NearestName   string  `json:"nearest_name"`
	NearestDist   float64 `json:"nearest_distance"`
	Margin        float64 `json:"margin"`          // NearestDist - OwnDistance (>0 = 本派系更近)
	HeldByInertia bool    `json:"held_by_inertia"` // margin<0 时：惯性保留
	Reason        string  `json:"reason"`
}

// cohortFeaturePoint cohort 在 (亲外, 忠诚) 平面上的位置 (解释用，不带权重)。
func cohortIdeologyPoint(cohort *model.PopulationCohort) (float64, float64) {
	return cohort.Ideologies[model.IdeologyProForeign], cohort.Ideologies[model.IdeologyLoyalty]
}

// factionIdeologyCentroid 派系成员人数加权的 (亲外, 忠诚) 质心。
func factionIdeologyCentroid(silo *model.Silo, factionID uint) (float64, float64, bool) {
	total := 0
	var sumPF, sumLoy float64
	for i := range silo.Cohorts {
		cohort := &silo.Cohorts[i]
		if cohort.FactionID == nil || *cohort.FactionID != factionID {
			continue
		}
		pf, loy := cohortIdeologyPoint(cohort)
		total += cohort.Count
		sumPF += pf * float64(cohort.Count)
		sumLoy += loy * float64(cohort.Count)
	}
	if total == 0 {
		return 0, 0, false
	}
	return sumPF / float64(total), sumLoy / float64(total), true
}

// ExplainCohortAffiliation 解释一个 cohort 的归属 (只读)。
func ExplainCohortAffiliation(silo *model.Silo, cohortID uint) (AffiliationExplanation, error) {
	if silo == nil {
		return AffiliationExplanation{}, fmt.Errorf("nil silo")
	}
	var cohort *model.PopulationCohort
	for i := range silo.Cohorts {
		if silo.Cohorts[i].ID == cohortID {
			cohort = &silo.Cohorts[i]
			break
		}
	}
	if cohort == nil {
		return AffiliationExplanation{}, fmt.Errorf("cohort %d not found", cohortID)
	}
	expl := AffiliationExplanation{
		SubjectKind: "cohort",
		SubjectID:   cohort.ID,
		SubjectName: cohort.Name,
	}
	pf, loy := cohortIdeologyPoint(cohort)
	fillAffiliationDistances(silo, factionByIDOrUnaffiliated(silo, cohort.FactionID), pf, loy, &expl)
	return expl, nil
}

// ExplainResidentAffiliation 解释一个居民的归属 (按其自身思潮，只读)。
// 具名正典人物 (Silo 18 链条) 也走这条路径。
func ExplainResidentAffiliation(silo *model.Silo, residentName string) (AffiliationExplanation, error) {
	if silo == nil {
		return AffiliationExplanation{}, fmt.Errorf("nil silo")
	}
	var resident *model.Resident
	for i := range silo.Residents {
		if silo.Residents[i].Name == residentName {
			resident = &silo.Residents[i]
			break
		}
	}
	if resident == nil {
		return AffiliationExplanation{}, fmt.Errorf("resident %q not found", residentName)
	}
	expl := AffiliationExplanation{
		SubjectKind: "resident",
		SubjectID:   resident.ID,
		SubjectName: resident.Name,
	}
	pf := resident.Ideologies[model.IdeologyProForeign]
	loy := resident.Ideologies[model.IdeologyLoyalty]
	fillAffiliationDistances(silo, factionByIDOrUnaffiliated(silo, resident.FactionID), pf, loy, &expl)
	return expl, nil
}

// factionByIDOrUnaffiliated 按 ID 查派系；nil / 无阵营桶统一返回 nil (按无归属解释)。
func factionByIDOrUnaffiliated(silo *model.Silo, factionID *uint) *model.Faction {
	if factionID == nil {
		return nil
	}
	f := factionByID(silo, *factionID)
	if f == nil || f.Signature == unaffiliatedSignature {
		return nil
	}
	return f
}

// fillAffiliationDistances 计算到本派系与次近政治派系的质心距离并填充解释。
// faction 为 nil 表示无归属 (未政治化 / 无阵营桶)。
func fillAffiliationDistances(silo *model.Silo, faction *model.Faction, pf, loy float64, expl *AffiliationExplanation) {
	if faction == nil {
		expl.Reason = "unaffiliated: 未达到政治化门槛或人数不足"
		expl.OwnDistance = math.Inf(1)
		expl.NearestDist = math.Inf(1)
		return
	}
	expl.FactionID = faction.ID
	expl.FactionName = faction.Name
	ownPF, ownLoy, ok := factionIdeologyCentroid(silo, faction.ID)
	if !ok {
		expl.Reason = "本派系无成员质心 (异常态)"
		return
	}
	expl.OwnDistance = math.Sqrt((pf-ownPF)*(pf-ownPF) + (loy-ownLoy)*(loy-ownLoy))
	for i := range silo.Factions {
		f := &silo.Factions[i]
		if f.ID == faction.ID || f.Signature == unaffiliatedSignature {
			continue
		}
		cPF, cLoy, ok := factionIdeologyCentroid(silo, f.ID)
		if !ok {
			continue
		}
		d := math.Sqrt((pf-cPF)*(pf-cPF) + (loy-cLoy)*(loy-cLoy))
		if expl.NearestID == 0 || d < expl.NearestDist {
			expl.NearestID = f.ID
			expl.NearestName = f.Name
			expl.NearestDist = d
		}
	}
	if expl.NearestID == 0 {
		expl.Reason = fmt.Sprintf("唯一政治派系：距离本派系质心 %.3f，无其他政治派系可比", expl.OwnDistance)
		return
	}
	expl.Margin = expl.NearestDist - expl.OwnDistance
	if expl.Margin >= 0 {
		expl.Reason = fmt.Sprintf("本派系更近：距离 %.3f vs 次近 %s %.3f (余量 %.3f)",
			expl.OwnDistance, expl.NearestName, expl.NearestDist, expl.Margin)
	} else {
		expl.HeldByInertia = true
		expl.Reason = fmt.Sprintf("惯性保留：次近 %s 更近 (%.3f vs %.3f)，因归属惯性留在原派系",
			expl.NearestName, expl.NearestDist, expl.OwnDistance)
	}
}

func factionByID(silo *model.Silo, id uint) *model.Faction {
	for i := range silo.Factions {
		if silo.Factions[i].ID == id {
			return &silo.Factions[i]
		}
	}
	return nil
}

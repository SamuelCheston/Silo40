package engine

// 可解释归属 (24 篇 C7 语义)：成员资格由发起/加入事件确立，持久不变；
// 到本派系质心与次近派系质心的特征距离只是亲缘场参考 (加入概率/退出
// 压力的输入)，不再决定归属。纯只读旁路：不修改状态、不消耗 RNG。

import (
	"fmt"
	"math"

	"silo40/internal/model"
)

// AffiliationExplanation 一个对象的归属解释。
type AffiliationExplanation struct {
	SubjectKind string  `json:"subject_kind"` // cohort / resident
	SubjectID   uint    `json:"subject_id"`
	SubjectName string  `json:"subject_name"`
	FactionID   uint    `json:"faction_id"`
	FactionName string  `json:"faction_name"`
	OwnDistance float64 `json:"own_distance"` // 到本派系质心的特征距离
	NearestID   uint    `json:"nearest_id"`   // 次近政治派系 ID (0 = 无其他派系)
	NearestName string  `json:"nearest_name"`
	NearestDist float64 `json:"nearest_distance"`
	Margin      float64 `json:"margin"` // NearestDist - OwnDistance (亲缘参考，非归属判据)
	Reason      string  `json:"reason"`
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
		expl.Reason = "无阵营：尚未加入任何派系 (成员资格由发起/加入事件产生)"
		expl.OwnDistance = math.Inf(1)
		expl.NearestDist = math.Inf(1)
		return
	}
	expl.FactionID = faction.ID
	expl.FactionName = faction.Name
	ownPF, ownLoy, ok := factionIdeologyCentroid(silo, faction.ID)
	if !ok {
		expl.Reason = fmt.Sprintf("事件驱动成员资格：归属由发起/加入事件确立；「%s」尚无成员质心 (零成员筹备期)", faction.Name)
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
	expl.Margin = expl.NearestDist - expl.OwnDistance
	if expl.NearestID == 0 {
		expl.Reason = fmt.Sprintf("事件驱动成员资格：归属持久；到本派系质心 %.3f (亲缘参考)，无其他政治派系可比", expl.OwnDistance)
		return
	}
	expl.Reason = fmt.Sprintf("事件驱动成员资格：归属持久，不因距离翻转；到本派系质心 %.3f vs 次近 %s %.3f (亲缘参考，余量 %.3f)",
		expl.OwnDistance, expl.NearestName, expl.NearestDist, expl.Margin)
}

func factionByID(silo *model.Silo, id uint) *model.Faction {
	for i := range silo.Factions {
		if silo.Factions[i].ID == id {
			return &silo.Factions[i]
		}
	}
	return nil
}

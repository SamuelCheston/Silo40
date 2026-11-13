package engine

// P1-S2 距离聚类：派系分组从"签名分桶"改为"特征向量距离聚类"。
//
// 签名分桶的两类语义缺陷 (P1 设计稿 §二)：
//   - 双胞胎派系：出生档位不同、思潮已收敛的 cohort 因签名不同永远并立；
//   - 签名坍缩：签名只含出生档位+drive，分辨率过粗，意识形态相距甚远的
//     cohort 被并入同一派系，且派系身份每 tick 按规模重排、无归属惯性，
//     产生归属翻转/振荡。
//
// 距离聚类在特征空间 (当前亲外思潮 / 当前忠诚 / 阶层) 上按加权欧氏距离
// 归并：思潮收敛的 cohort 自然同派系 (双胞胎结构上不可能)，意识形态不同
// 的 cohort 被半径分开 (坍缩结构上不可能)。归属惯性 (留在上一 tick 派系
// 的半径放大) 压制临界往返。全程确定性：cohort 按切片序处理、平局先惯性
// 后先建簇、不消耗 RNG。

import (
	"math"

	"silo40/internal/model"
)

// cohortFeature cohort 的聚类特征向量 (权重已乘入各维)。
func cohortFeature(silo *model.Silo, cohort *model.PopulationCohort, prof *model.Profession) []float64 {
	classCoord := 0.0
	if prof != nil && prof.ClassType == "ELITE" {
		classCoord = 1.0
	}
	return []float64{
		cohort.Ideologies[model.IdeologyProForeign] * FactionConfig.FeatureWeightProForeign,
		cohort.Ideologies[model.IdeologyLoyalty] * FactionConfig.FeatureWeightLoyalty,
		classCoord * FactionConfig.FeatureWeightClass,
	}
}

// featureDistance 加权欧氏距离 (权重已含在向量各维中)。
func featureDistance(a, b []float64) float64 {
	sum := 0.0
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return math.Sqrt(sum)
}

// distCluster 正在构建的距离簇 (最终成为一个派系)。P1-S3 起簇只负责分组与
// lineage 标记；公开状态/威望等身份状态由 RebuildImplicitFactions 直接从上一
// tick 的派系记录继承。
type distCluster struct {
	members      []*model.PopulationCohort
	profIDs      map[uint]bool
	centroid     []float64 // 人数加权质心 (增量更新)
	totalPop     int
	lineage      uint // 延续的上一 tick 派系 ID (0 = 新生派系)
	seedCohortID uint // 首个成员 cohort ID (确定性平局裁决)
}

func newDistCluster(cohort *model.PopulationCohort, feat []float64) *distCluster {
	return &distCluster{
		members:      []*model.PopulationCohort{cohort},
		profIDs:      map[uint]bool{cohort.ProfessionID: true},
		centroid:     append([]float64(nil), feat...),
		totalPop:     cohort.Count,
		seedCohortID: cohort.ID,
	}
}

// add 把 cohort 并入簇并增量更新质心。
func (c *distCluster) add(cohort *model.PopulationCohort, feat []float64) {
	newTotal := c.totalPop + cohort.Count
	if newTotal > 0 {
		for i := range c.centroid {
			c.centroid[i] = (c.centroid[i]*float64(c.totalPop) + feat[i]*float64(cohort.Count)) / float64(newTotal)
		}
	}
	c.totalPop = newTotal
	c.members = append(c.members, cohort)
	c.profIDs[cohort.ProfessionID] = true
}

// hostileTo 派系中任一部门与该职业双向关系恶劣即拒绝并入 (与签名分桶时代
// 的部门敌意约束一致)。
func (c *distCluster) hostileTo(silo *model.Silo, prof *model.Profession) bool {
	if prof == nil {
		return false
	}
	for id := range c.profIDs {
		if id == prof.ID {
			continue
		}
		other := getProfessionByID(silo, id)
		if other == nil {
			continue
		}
		if prof.Relations[other.Name] < FactionConfig.BadRelationThreshold ||
			other.Relations[prof.Name] < FactionConfig.BadRelationThreshold {
			return true
		}
	}
	return false
}

// factionAnchor 上一 tick 派系的质心锚点 (归属惯性用)，在清空 cohort.FactionID
// 之前从仍完整的旧派系/旧归属关系计算。
type factionAnchor struct {
	centroid []float64 // nil = 无成员锚点 (不可用于惯性)
}

// buildFactionAnchors 以当前特征计算旧派系质心 (思潮会漂移，锚点必须用
// 当前值而非旧快照)。
func buildFactionAnchors(silo *model.Silo) map[uint]*factionAnchor {
	anchors := map[uint]*factionAnchor{}
	for i := range silo.Factions {
		f := &silo.Factions[i]
		if f.Signature == unaffiliatedSignature {
			continue
		}
		anchors[f.ID] = &factionAnchor{}
	}
	type acc struct {
		sum   []float64
		total int
	}
	accs := map[uint]*acc{}
	for i := range silo.Cohorts {
		cohort := &silo.Cohorts[i]
		if cohort.FactionID == nil || cohort.Count <= 0 {
			continue
		}
		if _, ok := anchors[*cohort.FactionID]; !ok {
			continue
		}
		prof := getProfessionByID(silo, cohort.ProfessionID)
		feat := cohortFeature(silo, cohort, prof)
		ac := accs[*cohort.FactionID]
		if ac == nil {
			ac = &acc{sum: make([]float64, len(feat))}
			accs[*cohort.FactionID] = ac
		}
		for k := range feat {
			ac.sum[k] += feat[k] * float64(cohort.Count)
		}
		ac.total += cohort.Count
	}
	for id, a := range anchors {
		if ac, ok := accs[id]; ok && ac.total > 0 {
			a.centroid = make([]float64, len(ac.sum))
			for k := range ac.sum {
				a.centroid[k] = ac.sum[k] / float64(ac.total)
			}
		}
	}
	return anchors
}

// clusterPoliticalCohorts 把政治 cohort 按距离归并为簇 (按切片序处理，确定性)。
// 归属规则：
//  1. 已有簇中取距离 <= JoinRadius 的最近者 (平局取先建簇)；若最近者是
//     cohort 上一 tick 所属派系的延续簇，半径放大 (惯性)。
//  2. 上一 tick 派系尚未成簇且锚点距离 <= JoinRadius*(1+bonus) → 以该锚点
//     开簇 (延续旧派系身份与状态)。
//  3. 否则新开簇。
func clusterPoliticalCohorts(silo *model.Silo, political []*model.PopulationCohort,
	anchors map[uint]*factionAnchor, prevFactionOf map[uint]uint) []*distCluster {
	var clusters []*distCluster
	lineageCluster := map[uint]*distCluster{}

	for _, cohort := range political {
		prof := getProfessionByID(silo, cohort.ProfessionID)
		feat := cohortFeature(silo, cohort, prof)
		lineage := prevFactionOf[cohort.ID]

		bestCluster := -1
		bestDist := math.Inf(1)
		bestInertia := false
		for idx, c := range clusters {
			if c.hostileTo(silo, prof) {
				continue
			}
			d := featureDistance(feat, c.centroid)
			radius := FactionConfig.ClusterJoinRadius
			inertia := lineage != 0 && c.lineage == lineage
			if inertia {
				radius *= 1 + FactionConfig.ClusterInertiaBonus
			}
			if d <= radius && (d < bestDist || (d == bestDist && inertia && !bestInertia)) {
				bestCluster, bestDist, bestInertia = idx, d, inertia
			}
		}

		// 惯性锚点开簇 (仅当 lineage 尚未物化为簇)。
		if lineage != 0 {
			if _, materialized := lineageCluster[lineage]; !materialized {
				if a, ok := anchors[lineage]; ok && a.centroid != nil {
					d := featureDistance(feat, a.centroid)
					radius := FactionConfig.ClusterJoinRadius * (1 + FactionConfig.ClusterInertiaBonus)
					if d <= radius && d <= bestDist {
						c := newDistCluster(cohort, feat)
						c.lineage = lineage
						clusters = append(clusters, c)
						lineageCluster[lineage] = c
						continue
					}
				}
			}
		}

		if bestCluster >= 0 {
			clusters[bestCluster].add(cohort, feat)
			continue
		}
		clusters = append(clusters, newDistCluster(cohort, feat))
	}
	return clusters
}

// clusterDerivedProfile 由簇质心派生意识形态档位标签 (供命名/组织力/签名用)。
// 质心亲外 >=0.7 → High，>=0.3 → Medium，否则无档位标签 (与旧桶边界一致)。
func clusterDerivedProfile(c *distCluster) []string {
	pf := 0.0
	if FactionConfig.FeatureWeightProForeign > 0 && len(c.centroid) > 0 {
		pf = c.centroid[0] / FactionConfig.FeatureWeightProForeign
	}
	switch {
	case pf >= 0.7:
		return []string{model.IdeologyProForeign + ":High"}
	case pf >= 0.3:
		return []string{model.IdeologyProForeign + ":Medium"}
	default:
		return nil
	}
}

// factionGroupSize 组成员 cohort 的总人口 (排序用)。
func factionGroupSize(members []*model.PopulationCohort) int {
	total := 0
	for _, m := range members {
		total += m.Count
	}
	return total
}

// clusterDriveTag 簇的 drive 标签：取人数最多成员 cohort 的 drive (平局取
// cohort ID 更小者，确定性)。
func clusterDriveTag(c *distCluster) string {
	drive := "drive:care"
	var best *model.PopulationCohort
	for _, m := range c.members {
		if best == nil || m.Count > best.Count || (m.Count == best.Count && m.ID < best.ID) {
			best = m
		}
	}
	if best != nil {
		for _, tag := range best.Tags {
			if len(tag) > 6 && tag[:6] == "drive:" {
				drive = tag
				break
			}
		}
	}
	return drive
}

package engine

// 亲缘场 (24 篇 C7)：特征向量与加权距离保留，只用于计算加入概率与
// 退出压力 (C9) 及归属解释的亲缘参考。距离聚类不再产生派系——派系
// 只能由 FOUND_FACTION 事件创立，成员资格持久。

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


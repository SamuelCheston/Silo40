package engine

// FactionTuning centralizes the tunable constants of the implicit faction
// clustering (RebuildImplicitFactions) and the post-grouping loyalty
// dynamics. The defaults preserve the previously hardcoded behavior exactly;
// the struct gives the values one named home instead of scattered literals.
// It carries no persistence: save format is unaffected.
type FactionTuning struct {
	// ---- grouping ----

	// MinFactionSize: groups with less total population are folded into the
	// unaffiliated bucket.
	MinFactionSize int
	// BadRelationThreshold: departments whose bilateral relation drops below
	// this refuse to share a faction (either direction suffices).
	BadRelationThreshold float64
	// NonFactionName: display name of the unaffiliated bucket.
	NonFactionName string

	// ---- political participation gate ----

	// PoliticalEntryThreshold: an unaffiliated cohort politicizes when its
	// pro_foreign reaches this value (or a potential leader emerges).
	PoliticalEntryThreshold float64
	// PoliticalExitThreshold: a political cohort de-politicizes only when its
	// pro_foreign falls below this value (and it has no potential leader).
	// Kept below PoliticalEntryThreshold as hysteresis so cohorts hovering
	// near the gate do not flip every tick. Equal to the entry threshold
	// disables hysteresis.
	PoliticalExitThreshold float64

	// AmbitionHighThreshold: ambition at or above this marks a resident as
	// "ambitious" (drive:ambition / status:ambitious) and is the ambition
	// half of the potential-leader gate.
	AmbitionHighThreshold float64
	// LeaderInfluenceThreshold: influence half of the potential-leader gate.
	LeaderInfluenceThreshold float64

	// ---- drive tag selection ----

	// DriveStabilityPanicThreshold: department panic at or above this drives
	// drive:stability.
	DriveStabilityPanicThreshold float64
	// DriveStabilityCohesionThreshold: silo cohesion below this drives
	// drive:stability.
	DriveStabilityCohesionThreshold float64

	// ---- post-grouping loyalty dynamics (per-month steps) ----

	// LoyaltyDynamicsMonthlyStep: cohort loyalty convergence step applied by
	// the post-grouping dynamics once per game month (time-scaled for other
	// step sizes; zero-duration steps apply nothing).
	LoyaltyDynamicsMonthlyStep float64
	// ResidentLoyaltySyncMonthlyStep: resident->cohort loyalty sync step with
	// the same per-month semantics as LoyaltyDynamicsMonthlyStep.
	ResidentLoyaltySyncMonthlyStep float64

	// ---- faction cohesion formula ----

	// CohesionRelationWeight: weight of the average internal department
	// relation.
	CohesionRelationWeight float64
	// CohesionLeaderMaxBonus: leader prestige bonus at prestige 100.
	CohesionLeaderMaxBonus float64
	// CohesionSizePenaltyScale: scale of the log10(pop/100+1) size penalty.
	CohesionSizePenaltyScale float64

	// ---- representative cohort score weights ----

	RepScorePrestigeWeight float64 // weight of PoliticalPrestige/20
	RepScoreLoyaltyWeight  float64 // weight of cohort loyalty
	RepScoreSizeWeight     float64 // weight of ln(count+1)

	// ---- loyalty target biases (ApplyFactionLoyaltyDynamics) ----

	DissentBiasHigh    float64 // pro_foreign:High contribution to dissent bias
	DissentBiasMedium  float64 // pro_foreign:Medium contribution
	OrderBiasStatusQuo float64 // ideology:status_quo contribution to order bias
	OrderBiasMedium    float64 // pro_foreign:Medium contribution

	FormationBiasHigh   float64 // organization strength of formation:high factions
	FormationBiasMedium float64 // formation:medium factions
	FormationBiasLeader float64 // leader-formed (no formation tier) factions
	// FormationNonPublicBase / FormationPrestigeDivisor: non-public factions
	// keep only base + prestige/divisor of their organization strength.
	FormationNonPublicBase     float64
	FormationPrestigeDivisor   float64
	UnrestDissentBase          float64 // target -= formation*unrest*(base + extra*dissent)
	UnrestDissentExtra         float64
	OrderLegitimacyGain        float64 // target += formation*legitimacy*gain*order

	// ---- visibility model ----

	VisibilityRelationVisible float64 // max relation to a member dept for Visible
	VisibilityRelationAware   float64 // max relation to a member dept for Aware

	// ---- distance clustering (P1-S2) ----

	// FeatureWeightProForeign / FeatureWeightLoyalty / FeatureWeightClass:
	// cohort 特征向量的权重 (当前亲外思潮 / 当前忠诚 / 阶层坐标 ELITE=1,
	// COMMONER=0)。聚类距离 = 加权欧氏距离。权重为 0 即该维度不参与聚类。
	FeatureWeightProForeign float64
	FeatureWeightLoyalty    float64
	FeatureWeightClass      float64
	// ClusterJoinRadius: cohort 与派系质心的距离不超过该半径才并入。
	ClusterJoinRadius float64
	// ClusterInertiaBonus: 归属惯性——留在上一 tick 所属派系时，半径放大为
	// JoinRadius*(1+bonus)。压制临界往返 (A→B→A) 振荡。
	ClusterInertiaBonus float64

	// ---- representative ambition generation ----

	AmbitionBase           float64 // base ambition of generated representatives
	AmbitionPrestigeWeight float64 // weight of min(prestige,30)/30
	AmbitionIntensityWeight float64 // weight of ideological intensity
	AmbitionNoise          float64 // uniform noise amplitude
}

// FactionConfig is the active tuning for faction clustering and the
// post-grouping loyalty dynamics. Defaults reproduce the previous hardcoded
// values one-to-one.
var FactionConfig = FactionTuning{
	MinFactionSize:       200,
	BadRelationThreshold: 0.05,
	NonFactionName:       "Unaffiliated",

	PoliticalEntryThreshold: 0.7,
	PoliticalExitThreshold:  0.6, // hysteresis band: exit only below 0.6

	AmbitionHighThreshold:    0.62,
	LeaderInfluenceThreshold: 0.25,

	DriveStabilityPanicThreshold:    0.5,
	DriveStabilityCohesionThreshold: 0.45,

	LoyaltyDynamicsMonthlyStep:     0.50,
	ResidentLoyaltySyncMonthlyStep: 0.60,

	CohesionRelationWeight:   0.6,
	CohesionLeaderMaxBonus:   0.4,
	CohesionSizePenaltyScale: 0.05,

	RepScorePrestigeWeight: 0.45,
	RepScoreLoyaltyWeight:  0.20,
	RepScoreSizeWeight:     0.10,

	DissentBiasHigh:    0.6,
	DissentBiasMedium:  0.3,
	OrderBiasStatusQuo: 0.7,
	OrderBiasMedium:    0.1,

	FormationBiasHigh:        1.0,
	FormationBiasMedium:      0.7,
	FormationBiasLeader:      0.6,
	FormationNonPublicBase:   0.4,
	FormationPrestigeDivisor: 50.0,
	UnrestDissentBase:        0.12,
	UnrestDissentExtra:       0.12,
	OrderLegitimacyGain:      0.08,

	VisibilityRelationVisible: 0.4,
	VisibilityRelationAware:   0.15,

	FeatureWeightProForeign: 1.0,
	FeatureWeightLoyalty:    0.5,
	FeatureWeightClass:      0.4,
	ClusterJoinRadius:       0.30,
	ClusterInertiaBonus:     0.5,

	AmbitionBase:            0.22,
	AmbitionPrestigeWeight:  0.40,
	AmbitionIntensityWeight: 0.30,
	AmbitionNoise:           0.08,
}

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

	// AmbitionHighThreshold: ambition at or above this marks a resident as
	// "ambitious" (drive:ambition / status:ambitious)。纯 tagging 参数——
	// 潜在领袖硬门槛已废除 (24 篇 C7)，野心只进加权决策权重。
	AmbitionHighThreshold float64

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

	RepScorePrestigeWeight float64 // weight of derived prestige/20
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
	FormationNonPublicBase   float64
	FormationPrestigeDivisor float64
	UnrestDissentBase        float64 // target -= formation*unrest*(base + extra*dissent)
	UnrestDissentExtra       float64
	OrderLegitimacyGain      float64 // target += formation*legitimacy*gain*order

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

	// ---- representative ambition generation ----

	AmbitionBase            float64 // base ambition of generated representatives
	AmbitionPowerWeight     float64 // weight of min(PowerLevel,10)/10
	AmbitionIntensityWeight float64 // weight of ideological intensity
	AmbitionNoise           float64 // uniform noise amplitude

	// ---- FOUND_FACTION 发起资格 (24 篇 §1/§4, C6) ----

	FoundPrestigeThreshold  float64 // 发起所需派生威望 (50×(A+I))
	FoundIntensityThreshold float64 // 发起所需意识形态强度 |pf−0.5|
	FoundAPCost             float64 // 发起 AP 成本 (与 ACTION_COSTS 一致)

	// ---- 派系生命周期 (24 篇 §1.5, C7) ----

	FactionGraceMonths       int     // 筹备期月数：届满不足 MinFactionSize 即解散
	SuccessionPrestigeWeight float64 // 接任评分威望权重 (ambition 占 1−w)

	// ---- 三通道感知 (24 篇 §1.2/§4, C8) ----

	AwarenessRelationWeight float64 // 关系通道权重 (观察者→发起者所属部门人脉)
	AwarenessIdeologyWeight float64 // 思潮通道权重 (pf/loyalty 平面加权距离)
	AwarenessTierWeight     float64 // 阶层通道权重 (权力级差 + 同阶层加成)
	AwarenessSameClassBonus float64 // 同 ClassType 时阶层通道加成

	// ---- 月度口碑传播 (24 篇 §1.3/§4, C8) ----

	GossipReliabilityDecay float64 // 每跳可靠度折减 (接收强度 = 源 B × 折减)
	GossipBeliefThreshold  float64 // 源置信度达到此值才传播
	GossipRelationMin      float64 // 沿关系网络传播的最低关系值
	GossipSpreadChance     float64 // 每 (源, 目标) 对的传播掷骰概率 TODO(balance)

	// ---- 信息动作搭载 (24 篇 §1.3/§4, C8) ----

	// RumorPiggybackChance: 信息动作成功时搭载派系传闻的概率。
	// NPC/居民掷骰；玩家确定性搭载 (玩家动作不消耗 RNG，§6)。
	RumorPiggybackChance float64

	// ---- 加入/退出 (24 篇 §1.4/§4, C9) ----

	JoinAPCost           float64 // JOIN_FACTION 动作 AP 成本 (与 ACTION_COSTS 一致)
	JoinAffinityExponent float64 // 加入概率 = 相近度^k (意识形态主导)
	JoinRelationVeto     float64 // 关系一票否决地板
	CrossTierPowerGap    int     // PowerLevel 差 ≥ 此值算跨层级
	CrossTierRelationMin float64 // 跨层级所需关系
	CrossClassRelationMin float64 // 跨阶级所需关系
	AwarenessJoinFloor   float64 // 可考虑加入的信念地板 (= QualRumor 0.30，C8 感知/C9 加入共用)
	AffinityRadius       float64 // 亲缘半径：加权特征距离 ≤ 此值算亲缘圈内
	ExitInertiaFactor    float64 // 退出惯性系数：退出概率 ×(1−此值)，低于加入概率

	// ---- 合并 (24 篇 §1.6/§4, C10) ----

	MergeRelationMin float64 // 合并所需双方领袖个人关系 (LeaderRelations 子矩阵)
	MergeAPCost      float64 // MERGE_FACTION 动作 AP 成本 (与 ACTION_COSTS 一致)
}

// FactionConfig is the active tuning for faction clustering and the
// post-grouping loyalty dynamics. Defaults reproduce the previous hardcoded
// values one-to-one.
var FactionConfig = FactionTuning{
	MinFactionSize:       200,
	BadRelationThreshold: 0.05,
	NonFactionName:       "Unaffiliated",

	AmbitionHighThreshold: 0.62,

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

	AmbitionBase:            0.22,
	AmbitionPowerWeight:     0.40,
	AmbitionIntensityWeight: 0.30,
	AmbitionNoise:           0.08,

	FoundPrestigeThreshold:  50,
	FoundIntensityThreshold: 0.25,
	FoundAPCost:             20,

	FactionGraceMonths:       18,
	SuccessionPrestigeWeight: 0.5,

	AwarenessRelationWeight: 0.40,
	AwarenessIdeologyWeight: 0.35,
	AwarenessTierWeight:     0.25,
	AwarenessSameClassBonus: 0.15,

	GossipReliabilityDecay: 0.6,
	GossipBeliefThreshold:  0.50,
	GossipRelationMin:      0.30,
	GossipSpreadChance:     0.25,

	RumorPiggybackChance: 0.15,

	JoinAPCost:            10,
	JoinAffinityExponent:  2.0,
	JoinRelationVeto:      0.10,
	CrossTierPowerGap:     3,
	CrossTierRelationMin:  0.40,
	CrossClassRelationMin: 0.60,
	AwarenessJoinFloor:    0.30,
	AffinityRadius:        0.30,
	ExitInertiaFactor:     0.5,

	MergeRelationMin: 0.60,
	MergeAPCost:      15,
}

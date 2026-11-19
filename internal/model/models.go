package model

import (
	"sort"
	"time"

	"gorm.io/gorm"
)

const (
	IdeologyProForeign = "pro_foreign"
	IdeologyLoyalty    = "loyalty"
)

// CurrentSaveVersion 存档格式版本。任何会破坏旧存档可读性的模型/语义变更
// 都必须提升该版本，让 Resume 显式拒绝旧档，而不是靠 AutoMigrate 静默改表。
//
// 版本历史：
//   - v1: 单堡存档 (Silo 为根聚合)。
//   - v2: 引入 World 层 (世界模型骨架)；v1 存档经 UpgradeV1ToWorldSave
//     一次性显式升级 (备份 + 事务 + 回读校验)，空邻堡配置行为逐位不变。
//   - v3: 双地堡原型 (P0.5)：World 携带跨堡队列 (CrossQueue/CrossSeq) 与世界
//     flag (Flags)；Agent 携带 SiloID 以支持每堡一个代理操作者。v2 存档经
//     UpgradeV2ToTwoSiloSave 一次性显式升级；单堡 (空邻堡) 配置行为逐位不变。
//   - v4: 派系持久身份 (P1-S5)：Faction 增 Stance/Memory/Demands；派系 ID
//     为持久身份 (lineage 继承、新派系单调新 ID)。v3 存档经
//     UpgradeV3ToFactionIdentitySave 一次性显式升级 (备份 + 事务 + 回读校验)，
//     旧档字段由签名派生回填；距离聚类的行为变化已在 S2 激活步度量。
//   - v5: 可试玩多堡世界 (Live Silo 1、跨堡 operation 与历史引导评分)。
//     v4 存档经 UpgradeV4ToPlayableWorldSave 一次性显式升级；仅初始化新增
//     JSON 字段，不向旧档注入新堡或改写派系，保持既有模拟轨迹。
//   - v6: 外派特工身份模型。驻地顶替主 silo，40 号不再是模拟实体。
//     v5 身份语义不兼容，必须显式拒绝并开新局，不提供伪迁移。
//   - v7: 数据驱动行动门控与多实体加权结局。World 持久化阶段和已解锁行动，
//     VictoryStatus 持久化权重仲裁记录；v6 不提供伪迁移，必须显式拒绝。
//   - v8: Mayor 个人正统性、Police 服从、幕后威势 W 与 Judicial 三方附着。
//     v7 经备份+单事务升级；Mayor 从旧 Legitimacy 初始化，W 使用标注的保守初值。
//   - v9: 结局事件链机制底座。逃出群体 (EscapedGroup)、覆灭前夕思潮快照
//     (PreFallSnapshot)、公开约章事实键 (PublicFactKeys)、跨堡出站投递日志
//     (World.OutboundSignals) 与结局仲裁审计字段。新增字段零值即旧档语义，
//     v8 存档经备份+单事务+回读校验升级，不回填任何推测状态。
//   - v10: 准备度账本与顶层立场 (16 分册 Layer 0/1)。Silo 携带逃跑准备账本
//     (EscapePrep)、外溢准备账本 (SpilloverPrep)、顶层立场 (TopStance) 与
//     逃出群体审计字段 (Departed/Fragile)。新字段零值即旧档语义 (无准备、
//     立场零值=阻挠)，账本在出发结算落地前不被任何结局读取；v9 存档经
//     备份+单事务+回读校验升级，不回填任何推测状态。
//   - v11: 堡外维持 (16 分册 Layer 2)。EscapeGroup 携带 OutsideState：
//     补给余量/防护服完整性/路线安全/追踪四档/群体组合与覆没原因。
//     零值 = 未激活 = 旧档行为逐位不变 (月度结算只作用于 Active 群体)；
//     v10 存档经备份+单事务+回读校验升级，不回填任何推测状态。
//   - v12: 线信息架构。World 携带当下主线 (MainLine) 与改道日志
//     (LineShifts)——一局只有一条主线，改道是玩家可见的事实。
//     新字段零值即旧档语义 (主线未定、无改道记录)，由当前事实确定性
//     重推；v11 存档经备份+单事务+回读校验升级，不回填任何推测状态。
//   - v13: 市长与叛乱方谈判/妥协层 (21 设计卷)。Silo 携带谈判状态
//     (Negotiation)：信道、回合、市长路径与性情、桌上提案、妥协事实、
//     妥协后二选一与决策日志。新字段零值即旧档语义 (无谈判信道)，
//     未开信道的运行不消耗任何新 RNG；v12 存档经备份+单事务+回读校验
//     升级，不回填任何推测状态。
//   - v14: 时间渗透 (18-time-permeation)。部门信念条目 (FactBelief) 携带
//     渗透计数 (PermeationCounter) 与成熟标记 (Matured)：B 连续不低于分享
//     地板满渗透月数后事实成熟，《共证》资格门改读成熟计数。零值 = 未渗透
//     = 旧档"信念门未达成"的状态一致，不回填任何推测进度；v13 存档经
//     备份+单事务+回读校验升级 (v12 旧档先经 v13 谈判层升级接力)。
//   - v15: 黑箱学习层 + 盲区经营 (22 黑箱学习系列、23 盲区经营系列)。World
//     携带权衡层世界侧台账 (Silo1Stewardship) 与功能开关 (Silo1BlackBoxEnabled)，
//     Silo 携带每堡黑箱状态 (BlackBox：真值与信念分离)、盲区经营累积
//     (BlindSpot) 与后果事件登记簿 (SafeguardIncidents)。新字段零值 = 未启用
//     = 旧档逐位不变 (开关关闭 → 黑箱回合跳过、零 RNG 消耗；盲区零覆盖 →
//     观测注入恒等映射)，不回填任何推测状态 (先验初始化在运行时首次启用)；
//     旧档 Silo.SafeguardRisk 数值自此语义为「1 号指派基线 + 内容增量」
//     (数值不动)；v14 存档经备份+单事务+回读校验升级 (v12 旧档先经谈判层→
//     渗透两跳接力)。
//   - v16: 政治威望全层派生 (24-faction-event-algorithm-plan)。Agent/Profession/
//     Resident/PopulationCohort 删除独立 PoliticalPrestige 字段，威望改由
//     50×(Ambition+Influence) 派生；Agent/Profession 补齐 Ambition/Influence。
//     破坏性语义变更，无升级步：v15 及更早存档拒载 (ADR-0017 硬切断)。
//     与 v15 黑箱层合并时顺延编号：黑箱升级链终点 v15 保留不动。
//
// 19-opening-variability：Silo 另携带驻地原型标识 (Archetype)，建局时
// 写入；零值 = 无原型。旧档按 ADR-0017 硬切断拒载，不设原型升级步。
const CurrentSaveVersion = 16

// SaveMeta 存档级元数据，与具体游戏记录解耦的版本标记。
type SaveMeta struct {
	Key            string    `gorm:"primarykey" json:"key"`
	SaveVersion    int       `json:"save_version"`
	LegacyUpgraded bool      `json:"legacy_upgraded"` // 是否由无标记旧档一次性升级而来（升级留痕）
	UpgradedAt     time.Time `json:"upgraded_at"`     // 升级发生时间（仅 LegacyUpgraded 时有意义）
}

// User 用户基础模型
type User struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	Username  string         `gorm:"uniqueIndex;size:64" json:"username"`
	Password  string         `gorm:"size:255" json:"-"`
	SiloID    uint           `json:"silo_id"`
	Agent     Agent          `gorm:"foreignKey:UserID" json:"agent"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Agent 特工模型 (玩家角色)
type Agent struct {
	ID              uint         `gorm:"primarykey" json:"id"`
	UserID          uint         `gorm:"index" json:"user_id"`
	SiloID          uint         `gorm:"index" json:"silo_id"`                   // Current simulated silo; the player belongs to the assigned station.
	StationSiloID   uint         `gorm:"index" json:"station_silo_id,omitempty"` // Player's assigned station; zero marks an incompatible pre-v6 save.
	Name            string       `gorm:"size:64" json:"name"`
	Profession      string       `gorm:"size:50" json:"profession"`               // 职业 (如: Mayor, IT, Mechanical)
	Traits          []string     `gorm:"type:text;serializer:json" json:"traits"` // 特质 (如: "地堡土著", "一号地堡密使")
	Ambition        float64      `gorm:"default:0.0" json:"ambition"`             // 野心 (开局定死，仅事件可改)
	Influence       float64      `gorm:"default:0.0" json:"influence"`            // 个人影响力
	PoliticalPoints float64      `gorm:"default:0.0" json:"political_points"`     // 政治点数
	ActionPoints    float64      `gorm:"default:0.0" json:"action_points"`        // 行动点数 (AP)
	PropagandaLevel float64      `gorm:"default:0.0" json:"propaganda_level"`     // 宣传力度
	SuspicionLevel  float64      `gorm:"default:0.0" json:"suspicion_level"`      // 怀疑度指数
	Connections     []Connection `gorm:"foreignKey:AgentID" json:"connections"`
	KnownFragments  []string     `gorm:"type:text;serializer:json" json:"known_fragments"` // 特工个人掌握的信息碎片
	// Beliefs 分层信念层 (10-it-control-credibility §3.3/§3.5)：每个事实的连续
	// 确信 B、解释版本分布 π 与去重观察记录。旧档没有该字段时为空，读取时按
	// KnownFragments 保守补种 (engine.SeedLegacyBeliefs)，不升存档版本。
	Beliefs   []FactBelief   `gorm:"type:text;serializer:json" json:"beliefs,omitempty"`
	Relics    []Relic        `gorm:"foreignKey:AgentID" json:"relics,omitempty"` // 特工私吞的遗物
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// FactBelief 单个观察者对单个事实的分层信念 (10-it-control-credibility §3.3)。
// B 是事实成立的确信度；Versions 是事实成立时各解释版本的条件概率 π (和为 1)；
// Observations 按 observation_id 去重，同一观察不重放收益；ObsCount 是有效观察数 n。
//
// 时间渗透 (18-time-permeation §3.1)：PermeationCounter/Matured 是部门信念层
// 之上的只读派生状态——B 连续不低于分享地板满 InformationConfig.PermeationMonths
// 个月后 Matured 置真 (粘性，不回退)。仅部门层写入 (月度结算钩点)；特工层
// 与具名人物层不渗透 (§3.2)，这两层的条目恒为零值。零值 = 未渗透 = 旧档语义。
type FactBelief struct {
	FactID            string             `json:"fact_id"`
	B                 float64            `json:"b"`
	Versions          map[string]float64 `json:"versions,omitempty"`
	Observations      []string           `json:"observations,omitempty"`
	ObsCount          int                `json:"obs_count"`
	PermeationCounter int                `json:"permeation_counter,omitempty"`
	Matured           bool               `json:"matured,omitempty"`
}

// PublicEvidenceRecord 一份已公开证据的登记 (§3.7)：provenance 为来源等级
// (私人猜疑 0 / 匿名传闻 0.3 / 公开证词或未经核验记录 0.6 / 已核验档案或
// 市面可查原始记录 1)，coverage 为实际公开覆盖率 [0,1]。
type PublicEvidenceRecord struct {
	FactID     string  `json:"fact_id"`
	Provenance float64 `json:"provenance"`
	Coverage   float64 `json:"coverage"`
	Year       int     `json:"year"`
	Month      int     `json:"month"`
}

// ProtocolClause Order 的一条版本化条款 (10-it-control-credibility §6.1)。
// 真实副本由加载器持有 (不可伪造)；believed 副本持久化在存档里。
type ProtocolClause struct {
	ID                   string   `json:"id"`
	Version              int      `json:"version"`
	Category             string   `json:"category"` // prohibition/required_response/outcome_target/reporting/gap
	Description          string   `json:"description,omitempty"`
	ForbiddenAction      string   `json:"forbidden_action,omitempty"`
	RequiredAction       string   `json:"required_action,omitempty"`
	TriggerMetric        string   `json:"trigger_metric,omitempty"`
	TriggerAtLeast       float64  `json:"trigger_at_least,omitempty"`
	DeadlineMonths       int      `json:"deadline_months,omitempty"`
	TargetMetric         string   `json:"target_metric,omitempty"`
	TargetMin            *float64 `json:"target_min,omitempty"`
	TargetMax            *float64 `json:"target_max,omitempty"`
	ReviewIntervalMonths int      `json:"review_interval_months,omitempty"`
	Severity             string   `json:"severity"` // minor/major/existential/critical
	InterventionPoints   float64  `json:"intervention_points"`
	ReportingRequirement string   `json:"reporting_requirement,omitempty"`
	KnownGaps            string   `json:"known_gaps,omitempty"`
}

// ProtocolDeviation 一条已记录的偏离。每个条款每类只记一次 (§6.2)。
type ProtocolDeviation struct {
	ClauseID     string `json:"clause_id"`
	Kind         string `json:"kind"` // violation/lapse/unmet
	Severity     string `json:"severity"`
	Year         int    `json:"year"`
	Month        int    `json:"month"`
	Open         bool   `json:"open"`
	MissedCount  int    `json:"missed_count"` // 逾期月数 / 连续失败 review 数
	Acknowledged bool   `json:"acknowledged"` // 关键联盟是否知悉 (决定 W 惩罚)
}

// PendingRequirement 一个被激活、等待限期完成的必需动作。
type PendingRequirement struct {
	ClauseID       string `json:"clause_id"`
	DueTimestamp   int    `json:"due_timestamp"`
	ActivatedYear  int    `json:"activated_year"`
	ActivatedMonth int    `json:"activated_month"`
}

// ItAuthorityState 本地 IT 权威的二元联盟与隐蔽维护运行时状态 (§5)。
// 联盟只有 Judicial 法警与 IT 内部两块；R_op 是内部操作资源与可隐蔽支配
// 法警时间的统一抽象。
type ItAuthorityState struct {
	Initialized bool    `json:"initialized"`
	ROp         float64 `json:"r_op"` // [0,100]，每月 +8 收入，动作消耗按弱势成本曲线
	// 每组的忠诚拆为常规与恐惧两个分量：l_g = clamp01(L+F)。恐惧短期便宜、
	// 长期脆弱 (月回归 8%，常规 3%)。
	BailiffLoyalty    float64  `json:"bailiff_loyalty"`
	BailiffFear       float64  `json:"bailiff_fear"`
	ITInternalLoyalty float64  `json:"it_internal_loyalty"`
	ITInternalFear    float64  `json:"it_internal_fear"`
	DispatchCount     int      `json:"dispatch_count"`         // 法警调度次数 (暴露事实编号用)
	LastActions       []string `json:"last_actions,omitempty"` // 审计：最近动作 (升序追加，截断保留)
}

// ProtocolState 堡级 Protocol 运行时状态 (持久化)。
type ProtocolState struct {
	ConfigVersion     int                  `json:"config_version"`
	BelievedVersion   int                  `json:"believed_version"`
	BelievedClauses   []ProtocolClause     `json:"believed_clauses,omitempty"`
	Deviations        []ProtocolDeviation  `json:"deviations,omitempty"`
	Pending           []PendingRequirement `json:"pending,omitempty"`
	ReviewAnchors     map[string]int       `json:"review_anchors,omitempty"` // clauseID -> 上次 review 时间戳
	J                 float64              `json:"j"`
	WarningIssued     bool                 `json:"warning_issued"`
	InterventionStage string               `json:"intervention_stage,omitempty"` // warning/correction/override/assessment
	StageMonths       int                  `json:"stage_months"`
	FailedReviews     int                  `json:"failed_reviews"` // 关键目标连续未恢复的 review 数
	G                 float64              `json:"g"`              // 主任对上层控制方的认知 (§6.4)
	// DirectorBeliefs 主任对上层控制方相关事实 (Order_*) 的信念；G 由其聚合。
	// 三类生成器 (§6.5) 只经观察推进它，不直接改 G/W/E/Q。
	DirectorBeliefs []FactBelief `json:"director_beliefs,omitempty"`
	ChannelOpen     bool         `json:"channel_open"` // 控制通道仅在介入链开启
}

// Connection 各部门人脉值
type Connection struct {
	ID           uint    `gorm:"primarykey" json:"id"`
	AgentID      uint    `gorm:"index" json:"agent_id"`
	ProfessionID uint    `gorm:"index" json:"profession_id"`
	Value        float64 `gorm:"default:0.0" json:"value"` // 人脉值 (0-1: 0都不认识, 1全是熟人)
}

// WorldPacing 节奏定制 (开局高级选项)。所有旋钮只缩放"单位游戏时间内的
// 速率"，绝不触碰日历/事件时点/冷却月数/行动耗时/阈值/事件效果 delta——
// 剧情事件锚定绝对日历。默认全 1.0 = 历史行为逐位不变。
// 防抵消规则：每条账目只有一个属主旋钮 (生产只调 ResourceRate，消耗不调；
// 恐慌衰减与恐慌驱动的思潮漂移同属 IdeologyRate；叛乱 +/- 同属 PoliticalRate)。
type WorldPacing struct {
	ResourceRate   float64 `json:"resource_rate"`   // 生产产率倍率 (0.5–2.0)
	IdeologyRate   float64 `json:"ideology_rate"`   // 思潮漂移/恐慌衰减/忠诚收敛倍率 (0–3.0)
	PoliticalRate  float64 `json:"political_rate"`  // 叛乱累积与消退、派系动力学步长倍率 (0.5–2.0)
	ActionRate     float64 `json:"action_rate"`     // AP 恢复倍率 (0.5–3.0)
	DemographyRate float64 `json:"demography_rate"` // 死亡率倍率 (0.5–2.0)
}

// DefaultWorldPacing 全部旋钮 1.0 (与历史行为逐位一致)。
func DefaultWorldPacing() WorldPacing {
	return WorldPacing{
		ResourceRate:   1.0,
		IdeologyRate:   1.0,
		PoliticalRate:  1.0,
		ActionRate:     1.0,
		DemographyRate: 1.0,
	}
}

// Normalize 把缺省 (0 值) 旋钮补为默认 1.0，并把越界值夹回设计区间。
// 旧存档/旧请求缺字段时等价于 DefaultWorldPacing，行为不变。
func (p WorldPacing) Normalize() WorldPacing {
	norm := func(v, min, max float64) float64 {
		if v == 0 {
			v = 1.0
		}
		if v < min {
			v = min
		}
		if v > max {
			v = max
		}
		return v
	}
	return WorldPacing{
		ResourceRate:   norm(p.ResourceRate, 0.5, 2.0),
		IdeologyRate:   norm(p.IdeologyRate, 0.0, 3.0),
		PoliticalRate:  norm(p.PoliticalRate, 0.5, 2.0),
		ActionRate:     norm(p.ActionRate, 0.5, 3.0),
		DemographyRate: norm(p.DemographyRate, 0.5, 2.0),
	}
}

// Silo1State 一号地堡脚本化状态机 (多堡研究 §5 方案 A 的占位结构)。
// 一号地堡不跑完整模拟：它观测各堡信号、按 Order 规则动作。当前阶段仅
// 存储状态，无玩法消费方。
type Silo1State struct {
	Alive               bool             `json:"alive"`
	OrderPhase          string           `json:"order_phase"`
	Attention           map[uint]float64 `json:"attention"`
	LastSignalTimestamp int              `json:"last_signal_timestamp,omitempty"`
	LastReport          string           `json:"last_report,omitempty"`
	Exposure            float64          `json:"exposure,omitempty"`
	CounterintelStage   int              `json:"counterintel_stage,omitempty"`
	NextOperationID     int              `json:"next_operation_id,omitempty"`
	Operations          []Silo1Operation `json:"operations"`
}

// Silo1Operation is a delayed player operation against the live peer silo.
type Silo1Operation struct {
	ID             int     `json:"id"`
	TargetSiloID   uint    `json:"target_silo_id,omitempty"`
	ActionID       string  `json:"action_id"`
	Label          string  `json:"label"`
	Status         string  `json:"status"`
	StartYear      int     `json:"start_year"`
	StartMonth     int     `json:"start_month"`
	DueYear        int     `json:"due_year"`
	DueMonth       int     `json:"due_month"`
	Effect         string  `json:"effect"`
	TraceSuspicion float64 `json:"trace_suspicion"`
	Resolution     string  `json:"resolution,omitempty"`
}

// OperationTrack tracks operations against the player's assigned station.
type OperationTrack struct {
	LastSignalTimestamp int              `json:"last_signal_timestamp,omitempty"`
	LastReport          string           `json:"last_report,omitempty"`
	Exposure            float64          `json:"exposure,omitempty"`
	CounterintelStage   int              `json:"counterintel_stage,omitempty"`
	NextOperationID     int              `json:"next_operation_id,omitempty"`
	Operations          []Silo1Operation `json:"operations"`
}

// ============ 1 号权衡层 (22 设计卷、22 黑箱学习系列、ADR-0019) ============
//
// 语义锚点：
//   - 指标一 (Experiment)：实验履约与保留多样性——只影响"最终可保留地堡的
//     多样性"，不判负、不处刑、不进玩家结局资格。
//   - 指标二 (Lifespan)：剩余人员平均寿命——必然消耗的预算 (时间自然衰减 +
//     行动档位扣减)，耗尽导致能力退化而非判负。
//   - 指标三 (每堡 BlackBox)：safeguard 风险度由 1 号在黑箱回合指派；隐藏量
//     self_preservation (Θ) 与 20 箱后验信念 (b) 真值/信念分离存放
//     (22 黑箱学习系列 01 卷 §8 权威)。
//
// 零值语义 = 未启用：World.Silo1BlackBoxEnabled 为 false 时全部步骤跳过，
// 零 RNG 消耗，单堡与旧档行为逐位不变。

// 监控档位 (指标二的行动成本分档)。
const (
	Silo1PostureHigh       = "HIGH"        // 高度监控：成本最高
	Silo1PostureNormal     = "NORMAL"      // 正常监控
	Silo1PostureReportOnly = "REPORT_ONLY" // 只听取 IT 主任汇报：成本极低
)

// 黑箱观测信号类型 (22 黑箱学习系列 01 卷 §5.2)。堡内按类型升序产生与抽干。
const (
	BlackBoxSignalReceiptDelay = "receipt_delay" // Order 指令回执/执行回报时延，δ=1
	BlackBoxSignalExecutionGap = "execution_gap" // 复核结果与本地呈报差异，δ=2
	BlackBoxSignalLineTone     = "line_tone"     // IT 主任例行汇报态度，δ=2
)

// ExperimentMissRecord 一次"该清除而未清除"的履约缺口记录。
type ExperimentMissRecord struct {
	SiloID uint `json:"silo_id"`
	Year   int  `json:"year"`
	Month  int  `json:"month"`
}

// ExperimentStewardship 指标一：实验履约与保留多样性。
type ExperimentStewardship struct {
	// FulfillmentDeficit 履约缺口：该按要求清除的叛乱地堡没被清除的累积恶化
	// (随时间衰减)。
	FulfillmentDeficit float64 `json:"fulfillment_deficit"`
	// DiversityLoss 多样性损耗：每清除一个地堡的累积恶化 (随时间衰减，且越
	// 接近实验结束清除带来的损耗越轻)。
	DiversityLoss float64 `json:"diversity_loss"`
	// Scheduled 清除义务登记：通牒签发时登记 (siloID -> 登记月份)。
	Scheduled map[uint]int `json:"scheduled,omitempty"`
	// Cleared 清除成立事实 (siloID -> 覆灭月份)。
	Cleared map[uint]int `json:"cleared,omitempty"`
	// Missed 已结算过缺口的登记 (防重复计费)。
	Missed map[uint]bool `json:"missed,omitempty"`
	// Misses 缺口审计记录。
	Misses []ExperimentMissRecord `json:"misses,omitempty"`
	// Ledger 杂项台账 (防重复计费的已结算标记等)，键由实现定义。
	Ledger map[string]int `json:"ledger,omitempty"`
}

// LifespanStewardship 指标二：剩余人员平均寿命 (1 号人员轮流休眠)。
type LifespanStewardship struct {
	// AvgLifespan 平均寿命 (年)。时间本身必然使其下降；行动按档位额外扣减。
	AvgLifespan float64 `json:"avg_lifespan"`
	// Initialized 区分"零值旧档"与"已启用"。
	Initialized bool `json:"initialized,omitempty"`
	// Postures 每堡监控档位 (siloID -> Silo1Posture*)。
	Postures map[uint]string `json:"postures,omitempty"`
	// Ledger 杂项台账 (一次性行动成本的已计费标记等)，键由实现定义。
	Ledger map[string]int `json:"ledger,omitempty"`
}

// Silo1StewardshipState 1 号权衡层世界侧状态 (指标一/二)。指标三的每堡状态
// 在 Silo.BlackBox (真值与信念分离，22 黑箱学习系列 01 卷 §8)。
type Silo1StewardshipState struct {
	// Initialized 区分"零值旧档"与"已启用"。
	Initialized bool                  `json:"initialized,omitempty"`
	Experiment  ExperimentStewardship `json:"experiment"`
	Lifespan    LifespanStewardship   `json:"lifespan"`
}

// BlackBoxPendingObservation 在途延迟观测信号 (22 黑箱学习系列 01 卷 §8)。
// 世界侧 4.65 产生、1 号侧黑箱回合按到达月抽干；排序键 (到达月, 信号类型, Seq)。
type BlackBoxPendingObservation struct {
	ArriveYear     int     `json:"arrive_year"`
	ArriveMonth    int     `json:"arrive_month"`
	ReflectedYear  int     `json:"reflected_year"` // 反映月 (归因记录，审计用)
	ReflectedMonth int     `json:"reflected_month"`
	SignalType     string  `json:"signal_type"` // BlackBoxSignal*
	Value          float64 `json:"value"`       // y ∈ [0,1]
	Seq            int     `json:"seq"`         // 堡内单调序号 (排序第三键)
}

// BlackBoxLogEntry 一次黑箱指派决策的审计留痕 (22 黑箱学习系列 04 卷 §2)。
// 审计渠道：不进玩家界面。
type BlackBoxLogEntry struct {
	Year     int     `json:"year"`
	Month    int     `json:"month"`
	ThetaHat float64 `json:"theta_hat"` // 决策时刻后验均值 (1 号自己的估计)
	Lo       float64 `json:"lo"`        // 90% 质量最短区间下界
	Hi       float64 `json:"hi"`        // 90% 质量最短区间上界
	PDanger  float64 `json:"p_danger"`  // 危险区概率
	ChosenR  float64 `json:"chosen_r"`
	Utility  float64 `json:"utility"`
	Reason   string  `json:"reason,omitempty"` // 理由码 exploit/explore/event (04 卷 §2.3)
	Factors  string  `json:"factors"`          // 因子读数摘要 (可复盘依据)
}

// Silo1BlackBoxState 每堡黑箱状态 (22 黑箱学习系列 01 卷 §8)。
// 真值 (Theta) 与信念 (Belief) 分字段存放：世界侧结算读 Theta，1 号决策侧
// 只读 Belief 与观测——任何让决策读到 Theta 真值或 E 曲线参数的路径都是违规。
type Silo1BlackBoxState struct {
	// Initialized 零值 false → 功能未启用 = 旧档语义。
	Initialized bool `json:"initialized,omitempty"`
	// Theta 世界侧真实隐藏量 Θ (self_preservation)；0 = 完全按 Order 办事。
	Theta float64 `json:"theta"`
	// Belief N=20 箱后验 (长度 N)；nil → 首次启用时按先验初始化 (运行时，
	// 不经升级器)。
	Belief []float64 `json:"belief,omitempty"`
	// LastAssignedRisk 上月 2b 指派、本月生效的 r；零值 → 首次启用月以当前
	// SafeguardRisk 为生效 r (连续性)。
	LastAssignedRisk float64 `json:"last_assigned_risk"`
	// CurrentExecution 本月 E 生效槽 (4.55 写入，供 4.6 消费与审计回放)。
	CurrentExecution float64 `json:"current_execution"`
	// PendingObservations 在途延迟信号；零值 = 无在途。
	PendingObservations []BlackBoxPendingObservation `json:"pending_observations,omitempty"`
	// ObsSeq 堡内观测单调序号计数器。
	ObsSeq int `json:"obs_seq,omitempty"`
	// ProbeCooldown 探索冷却计数 (01 卷 §6.4 接口预留)；零值 = 可用。
	ProbeCooldown int `json:"probe_cooldown,omitempty"`
	// ObediencePulse 通牒后表面服从脉冲剩余月 (22-02 §3.5 ΔE_show、缺口 G10)：
	// 通牒签发时置 1–2 月，4.55 结算时对 E 叠加临时项；零值 = 无脉冲。
	ObediencePulse int `json:"obedience_pulse,omitempty"`
	// Log 决策审计日志，环形上限 24 (每堡独立)。
	Log []BlackBoxLogEntry `json:"log,omitempty"`
}

// 盲区经营六通道 (23 系列 04 卷 §1.1)。固定顺序即存档数组序与累加序，
// 禁止 map 遍历序 (确定性纪律)。
const (
	BlindSpotChannelEquipment = "equipment" // 监控设备盲区
	BlindSpotChannelSchedule  = "schedule"  // 排班/迎检安排
	BlindSpotChannelReports   = "reports"   // 报表口径
	BlindSpotChannelLedger    = "ledger"    // 台账/回执
	BlindSpotChannelCensus    = "census"    // 人口底册
	BlindSpotChannelTiming    = "timing"    // 上报时点
)

// 拆穿阶段 (23 系列 04 卷 §3.4)。
const (
	BlindSpotStageNone      = ""
	BlindSpotStageDoubt     = "doubt"     // 起疑
	BlindSpotStageVerify    = "verify"    // 查证
	BlindSpotStageConfirmed = "confirmed" // 坐实
)

// BlindSpotTraceEntry 手段使用在世界侧留下的可追溯痕迹 (事实层；1 号侧永不
// 直读，只能经稽查抽样与统计形状间接捕捉)。
type BlindSpotTraceEntry struct {
	Year      int     `json:"year"`
	Month     int     `json:"month"`
	Channel   string  `json:"channel"`   // BlindSpotChannel*
	Kind      string  `json:"kind"`      // 痕迹类型 (04 卷 §1.4 表)
	Magnitude float64 `json:"magnitude"` // 痕迹量级 ∝ 手段力度
	Seq       int     `json:"seq"`       // 堡内单调序号 (排序键)
}

// BlindSpotEvidenceEntry 拆穿判据窗口内的证据事件 (环形上限 24)。
type BlindSpotEvidenceEntry struct {
	Year      int     `json:"year"`
	Month     int     `json:"month"`
	Source    string  `json:"source"`    // cross_mismatch / proto_contrast / burst_residual / trace_shape
	Magnitude float64 `json:"magnitude"` // 证据量级 ∈ [0,1]
	Seq       int     `json:"seq"`
}

// BlindSpotCultivationState 每堡盲区经营累积状态 (23 系列 04 卷 §1.5)。
// 零值 = 未经营 = 注入恒等映射 = 旧档逐位不变。
type BlindSpotCultivationState struct {
	// Initialized 零值 false → 功能未启用 = 旧档语义。
	Initialized bool `json:"initialized,omitempty"`
	// Coverage 逐通道盲区厚度 Cov_c ∈ [0,1]，固定通道序 (六通道)。
	Coverage [6]float64 `json:"coverage"`
	// Bias 逐通道被经营偏置 ∈ [0,1] (方向固定朝"平静")，固定通道序。
	Bias [6]float64 `json:"bias"`
	// Suspicion 怀疑计数器 ∈ [0,1] (04 卷 §3.2)。
	Suspicion float64 `json:"suspicion"`
	// Stage 拆穿阶段 (BlindSpotStage*)；StageMonths 当前阶段已持续月数。
	Stage       string `json:"stage,omitempty"`
	StageMonths int    `json:"stage_months,omitempty"`
	// Traces 痕迹账本 (保留期 36 月，升序)。
	Traces []BlindSpotTraceEntry `json:"traces,omitempty"`
	// TraceSeq 堡内痕迹单调序号计数器。
	TraceSeq int `json:"trace_seq,omitempty"`
	// Evidence 近期证据事件 (环形上限 24)。
	Evidence []BlindSpotEvidenceEntry `json:"evidence,omitempty"`
	// ChannelResetMonth 逐通道最近一次重置月 (04 卷 §4.5)。
	ChannelResetMonth [6]int `json:"channel_reset_month"`
	// StrictUntil 「更严监控」标记的截止月 (04 卷 §4.2)。
	StrictUntil int `json:"strict_until,omitempty"`
	// ActiveDisguise 上月选中手段类 (惯性因子输入，03 卷 §2.9)。
	ActiveDisguise string `json:"active_disguise,omitempty"`
	// Log 盲区经营决策审计日志 (世界侧，环形上限 24；不进玩家界面)。
	Log []BlindSpotDecisionLogEntry `json:"log,omitempty"`
}

// BlindSpotDecisionLogEntry 一次盲区经营决策的审计留痕 (23 系列 03 卷)。
type BlindSpotDecisionLogEntry struct {
	Year      int    `json:"year"`
	Month     int    `json:"month"`
	Acted     bool   `json:"acted"`            // 第一级抽取结果
	Choice    string `json:"choice,omitempty"` // 手段类 ID (不动手为 idle)
	Channel   string `json:"channel,omitempty"`
	Intensity int    `json:"intensity,omitempty"` // 力度档
	Weights   string `json:"weights"`             // 归一化权重表
	Factors   string `json:"factors"`             // 六因子读数摘要 (可归因)
}

// World 一局游戏的顶层聚合 (多堡研究 §2)。P0.5 起可挂两个完整模拟堡：
// 世界时钟为权威 (各堡时钟是其镜像)、跨堡事件队列按 (到期月, Seq) 排序、
// Silo 1 为脚本化状态机。单堡 (空邻堡) 配置下所有多堡机制关闭，行为与
// v2 逐位一致。
type World struct {
	ID           uint            `gorm:"primarykey" json:"id"`
	Seed         int64           `json:"seed"` // 主种子；玩家堡 RNGSeed == Seed (旧档升级兼容)
	CurrentYear  int             `json:"current_year"`
	CurrentMonth int             `json:"current_month"`
	PlayerSiloID uint            `json:"player_silo_id"`
	Pacing       WorldPacing     `gorm:"type:text;serializer:json" json:"pacing"`
	Silo1        Silo1State      `gorm:"type:text;serializer:json" json:"silo1"`
	StationOps   OperationTrack  `gorm:"type:text;serializer:json" json:"station_operation_state"`
	Topology     map[uint][]uint `gorm:"type:text;serializer:json" json:"topology"` // siloID -> 相邻 siloID (空 = 无邻堡)
	// CrossQueue 跨堡事件队列 (多堡研究 §4.3)。派发严格按 (到期月, Seq) 排序；
	// Seq 由 CrossSeq 单调计数器生成。SourceSilo = 0 表示 Silo 1 / 世界发起。
	CrossQueue []CrossEvent `gorm:"type:text;serializer:json" json:"cross_queue"`
	CrossSeq   int          `json:"cross_seq"`
	// Flags 世界事实 (与堡 flag 分开的白名单表，见 ValidWorldFlag)。
	Flags           map[string]bool `gorm:"type:text;serializer:json" json:"flags"`
	ProgressStage   string          `json:"progress_stage"`
	UnlockedActions []string        `gorm:"type:text;serializer:json" json:"unlocked_actions"`
	HistoryImpacts  []HistoryImpact `gorm:"type:text;serializer:json" json:"history_impacts"`
	// OutboundSignals 跨堡出站投递日志 (G7)：派发阶段按 (到期月, Seq) 顺序
	// 追加，因此整个日志本身即确定性有序。外溢结算只读它，不改写投递。
	OutboundSignals []OutboundSignal `gorm:"type:text;serializer:json" json:"outbound_signals,omitempty"`
	// LastEndingAudit 最近一次终局仲裁记录 (含"继续模拟"的空池裁决)，
	// 供结局体系 12 §9 的解释性审计；不影响任何模拟状态。
	LastEndingAudit *EndingDecision `gorm:"type:text;serializer:json" json:"last_ending_audit,omitempty"`
	// MainLine 当下主线 (线信息架构)：玩家实际在走的那条结局线。由里程碑
	// 事实确定性推出 (最深者胜，平层取最近推进)，不参与任何结算；空 = 未定。
	MainLine string `json:"main_line,omitempty"`
	// LineShifts 改道日志：主线每次变化记一条 (含首次确立，From 为空)。
	// 改道是玩家可见的事实——从《共证》改道《公议》不该是"某个数字悄悄不涨"。
	LineShifts []LineShift `gorm:"type:text;serializer:json" json:"line_shifts,omitempty"`
	// Silo1Stewardship 1 号权衡层 (22 设计卷)：指标一/二。零值 = 未启用。
	Silo1Stewardship Silo1StewardshipState `gorm:"type:text;serializer:json" json:"silo1_stewardship"`
	// Silo1BlackBoxEnabled 权衡层功能开关 (建档请求 Silo1BlackBox 持久化)：
	// false (默认/旧档) → 黑箱回合、4.55/4.65、指标一/二全部跳过，零 RNG 消耗，
	// 与既有 parity 基线逐位一致。
	Silo1BlackBoxEnabled bool      `gorm:"default:false" json:"silo1_black_box_enabled"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// LineShift 一次主线变化：From 为空表示首次确立主线。
type LineShift struct {
	Year  int    `json:"year"`
	Month int    `json:"month"`
	From  string `json:"from,omitempty"`
	To    string `json:"to"`
}

// OutboundSignal records one delivered cross-silo signal for spillover
// settlement (G7). SourceSilo = 0 marks Silo 1 / world origin.
//
// Layer 3 (16 §3.5 外溢投递侧) 扩展字段全部零值即旧档语义：Spillover=false
// 的记录是通用跨堡信号，计数口径与 G7 逐位一致；Spillover=true 的记录是
// 外溢包裹投递，口径收紧为"到达且独立接收"才进触达计数 (Spillover=false
// 的旧记录不受影响)。
type OutboundSignal struct {
	Seq        int    `json:"seq"`
	SourceSilo uint   `json:"source_silo"`
	TargetSilo uint   `json:"target_silo"`
	EventID    string `json:"event_id"`
	Year       int    `json:"year"`
	Month      int    `json:"month"`
	// Spillover 该记录是外溢包裹投递 (区别于通用跨堡信号)。
	Spillover bool `json:"spillover,omitempty"`
	// Received 目标堡按自身模拟状态独立完成核验 (接收结算)。到达而未独立
	// 接收的包裹不进触达计数，只留到达记录。
	Received bool `json:"received,omitempty"`
	// Generation 链代次：1 = 驻地发出的第一代；>1 = 接收方再复制的下一代。
	Generation int `json:"generation,omitempty"`
	// Quality 到达时的包裹质量 (信道容量与路上损耗结算后的值)。
	Quality float64 `json:"quality,omitempty"`
	// RootSilo 链的起源堡 (玩家驻地)：再复制代的 SourceSilo 是接收堡，
	// 计数按 RootSilo 归属整条链。0 = 非外溢记录。
	RootSilo uint `json:"root_silo,omitempty"`
}

// CrossEvent 跨堡事件：某个堡 (或 Silo 1 / 世界) 向目标域投递的延迟消息。
// Payload 只允许标量/字符串 (可序列化、可复现)。
type CrossEvent struct {
	Seq         int               `json:"seq"`
	SourceSilo  uint              `json:"source_silo"` // 0 = Silo 1 / 世界
	TargetScope string            `json:"target_scope"`
	EventID     string            `json:"event_id"`
	DueYear     int               `json:"due_year"`
	DueMonth    int               `json:"due_month"`
	Payload     map[string]string `json:"payload,omitempty"`
}

// 世界 flag 白名单：内容效果的 world_flag_set 与触发器 world_flag_true/false
// 只允许读写这些键 (封闭集原则，未知 flag = 加载失败)。
const (
	WorldFlagSilo1Destroyed = "silo1_destroyed" // 世界事实：一号地堡是否覆灭 (!Silo1.Alive 的别名)
	WorldFlagNetworkJammed  = "network_jammed"  // COALITION 线预留：终止信号被干扰
)

// ValidWorldFlag 世界 flag 白名单校验。
func ValidWorldFlag(flag string) bool {
	switch flag {
	case WorldFlagSilo1Destroyed, WorldFlagNetworkJammed:
		return true
	}
	return false
}

// MayorOffice is the named office holder's personal authority. Silo.Legitimacy
// remains only a mirrored compatibility readout for old clients.
type MayorOffice struct {
	Initialized      bool    `json:"initialized"`
	HolderResidentID uint    `json:"holder_resident_id,omitempty"`
	HolderName       string  `json:"holder_name"`
	Legitimacy       float64 `json:"legitimacy"`
}

// JudicialAttachment stores the Judicial allegiance and its hysteresis state.
// G12 ruling: attachment is the IT/pact split only (IT + Pact == 1); the Mayor
// field persists as a bounded influence modifier (pressure / replacement /
// per-case relaxation) and never feeds Regime. Regime ∈ IT/SWING/PACT; legacy
// saves may still carry "MAYOR", which the authority engine normalizes to
// SWING on the next advance. Pact fields are structural qualifications only;
// no current content grants them.
type JudicialAttachment struct {
	IT                     float64 `json:"a_it"`
	Mayor                  float64 `json:"a_mayor"` // influence modifier, not an attachment share (G12)
	Pact                   float64 `json:"a_pact"`
	TargetIT               float64 `json:"target_it"`
	TargetMayor            float64 `json:"target_mayor"`
	TargetPact             float64 `json:"target_pact"`
	CommandIT              float64 `json:"command_it"`
	CommandMayor           float64 `json:"command_mayor"`
	CommandPact            float64 `json:"command_pact"`
	Regime                 string  `json:"regime"`
	MayorLowMonths         int     `json:"mayor_low_months"`
	MayorHighMonths        int     `json:"mayor_high_months"`
	PactLowMonths          int     `json:"pact_low_months"`
	PactQualified          bool    `json:"pact_qualified"`
	PublicEvidenceQ        float64 `json:"public_evidence_q"`
	IndependentPublicFacts int     `json:"independent_public_facts"`
	// PublicFactKeys 已登记的独立公共事实键。子项 3 起由公开证据登记的合格事实
	// 月度覆写 (engine.ComputeInformationEnvironment)；authority_public_fact 为
	// 遗留通路。与 IndependentPublicFacts 长度保持一致。
	PublicFactKeys []string `json:"public_fact_keys,omitempty"`
	PactLocked     bool     `json:"pact_locked"`
}

// DecisiveShock 结构性事件的一次性离散负冲击 (裁决：冲击通道)。不复用 E、
// 不走日常累积；可击穿日常平衡点下限。记录永久保留作审计，消费游标单独推进。
type DecisiveShock struct {
	Value  float64 `json:"value"`
	Source string  `json:"source"`
	Year   int     `json:"year"`
	Month  int     `json:"month"`
}

// ShadowAuthority is the independent IT authority state. OrderDeviation is a
// reserved input interface and is deliberately absent from the v8 W formula.
type ShadowAuthority struct {
	Initialized          bool    `json:"initialized"`
	InitializationSource string  `json:"initialization_source"`
	W                    float64 `json:"w"`
	K                    float64 `json:"k"`
	E                    float64 `json:"e"`
	D                    float64 `json:"d"`
	P                    float64 `json:"p"`
	X                    float64 `json:"x"`
	OrderDeviation       float64 `json:"order_deviation"`
	// Shocks 待消费的 W 冲击队列 (§3.2 的 Σδe)：Protocol 偏离等写入，下一次
	// 月度权威更新消费一次 (单月合计 cap ±12)。
	Shocks []float64 `json:"shocks,omitempty"`
	// DecisiveShocks 结构性事件的离散冲击 (裁决的冲击通道)：不受 ±12 上限约束，
	// 可击穿日常平衡点下限；DecisiveApplied 是消费游标。
	DecisiveShocks  []DecisiveShock    `json:"decisive_shocks,omitempty"`
	DecisiveApplied int                `json:"decisive_applied,omitempty"`
	PoliceObedience float64            `json:"police_obedience"`
	Judicial        JudicialAttachment `json:"judicial"`
}

// Silo 地堡核心模型
type Silo struct {
	ID      uint     `gorm:"primarykey" json:"id"`
	WorldID uint     `gorm:"index;default:1" json:"world_id"` // 所属世界 (世界模型 v2 起)
	Name    string   `gorm:"size:100" json:"name"`
	Traits  []string `gorm:"type:text;serializer:json" json:"traits"` // 地堡特质 (abundant/leak/psychoactive_meds)
	// Archetype 驻地原型 (19-opening-variability §3)：建档时写入的既成事实
	// (号数确定性查表派生或请求覆写)，月度结算据此读档间旋钮。零值 = 无
	// 原型 = 旧档/空配置行为逐位不变。
	Archetype     string  `gorm:"size:20" json:"archetype,omitempty"`
	SafeguardRisk float64 `gorm:"default:0.0" json:"safeguard_risk"` // IT 专属风险系数
	// BlackBox 每堡黑箱状态 (22 黑箱学习系列 01 卷 §8)：真值 Theta 与信念 Belief
	// 分字段存放。零值 = 功能未启用 = 旧档语义。
	BlackBox Silo1BlackBoxState `gorm:"type:text;serializer:json" json:"black_box"`
	// BlindSpot 每堡盲区经营累积状态 (23 系列 04 卷)。零值 = 未经营 = 恒等映射。
	BlindSpot BlindSpotCultivationState `gorm:"type:text;serializer:json" json:"blind_spot"`
	// SafeguardIncidents 引擎侧后果事件登记簿 (ADR-0019)：原直接累加风险度的各
	// 累加点改为在此登记 (kind -> 计数)，经观测管线进入 1 号信念。引擎包写入
	// (协议偏离/IT 过度干预/不作为/蒙蔽崩塌)，观测管线消费。
	SafeguardIncidents map[string]int          `gorm:"type:text;serializer:json" json:"safeguard_incidents,omitempty"`
	TotalPopulation    int                     `gorm:"default:10000" json:"total_population"`
	Legitimacy         float64                 `gorm:"default:1.0" json:"legitimacy"`                      // Compatibility mirror of Mayor.Legitimacy.
	Mayor              MayorOffice             `gorm:"type:text;serializer:json" json:"mayor"`             // 现任 Mayor 个人正统性
	Authority          ShadowAuthority         `gorm:"type:text;serializer:json" json:"authority"`         // IT 幕后威势与 Judicial 附着
	EscapedGroup       EscapeGroup             `gorm:"type:text;serializer:json" json:"escaped_group"`     // 已离开且可能存续的堡外群体 (G5)
	TopStance          TopStanceState          `gorm:"type:text;serializer:json" json:"top_stance"`        // 顶层立场 (16 §3.2)：内部状态，仲裁不读
	Negotiation        NegotiationState        `gorm:"type:text;serializer:json" json:"negotiation"`       // 市长与叛乱方谈判/妥协层 (21 设计卷)：只改变谁在阻碍谁，不发放胜利
	EscapePrep         EscapePreparation       `gorm:"type:text;serializer:json" json:"escape_prep"`       // 逃跑准备账本 (16 §2)：事实账本，仲裁不读
	SpilloverPrep      SpilloverPreparation    `gorm:"type:text;serializer:json" json:"spillover_prep"`    // 外溢准备账本 (16 §2.3)：信道建立侧
	EscapeResponse     EscapeResponseState     `gorm:"type:text;serializer:json" json:"escape_response"`   // 筹备反制层 (20 修订/ADR-0014)：组织怀疑度与 IT 带权响应
	Countermeasure     CountermeasureState     `gorm:"type:text;serializer:json" json:"countermeasure"`    // 20 分册第 5 批：T1-3/T2-4/T3-1b/T3-2 反制账本
	PreFallSnapshot    PreFallIdeologySnapshot `gorm:"type:text;serializer:json" json:"pre_fall_snapshot"` // 覆灭前夕人口加权思潮快照 (G4)
	// PublicEvidence 已公开证据登记 (§3.7)：Q 与独立公开事实资格只读这里，
	// 私下确信 (玩家 B) 永远不直接进入曝光或公开证据。
	PublicEvidence []PublicEvidenceRecord `gorm:"type:text;serializer:json" json:"public_evidence,omitempty"`
	// VerifiedEvidence 已完成核验、具备已核验档案级 provenance 的事实 ID
	// (子项 6：核验是发布的前置)。按 ID 升序去重。
	VerifiedEvidence []string `gorm:"type:text;serializer:json" json:"verified_evidence,omitempty"`
	// Protocol Order 条款运行时状态 (子项 4)：believed 副本、偏离记录、介入链
	// 与主任认知 G。真实条款由加载器持有，不持久化、不可伪造。
	Protocol ProtocolState `gorm:"type:text;serializer:json" json:"protocol"`
	// ItAuthority 本地 IT 权威的二元联盟与隐蔽维护状态 (子项 5)。
	ItAuthority        ItAuthorityState   `gorm:"type:text;serializer:json" json:"it_authority"`
	Cohesion           float64            `gorm:"default:1.0" json:"cohesion"`                               // 凝聚力值
	Rebellion          float64            `gorm:"default:0.0" json:"rebellion"`                              // 地堡叛乱值
	DeptTension        float64            `gorm:"default:0.0" json:"dept_tension"`                           // 部门间紧张程度 (0-1)
	ClassFragmentation float64            `gorm:"default:0.0" json:"class_fragmentation"`                    // 精英与平民割裂程度 (0-1)
	HistoryBurden      float64            `gorm:"default:0.0" json:"history_burden"`                         // 历史包袱值 (罪恶/荣誉)
	EventTrigger       float64            `gorm:"default:0.0" json:"event_trigger"`                          // 外部事件触发值
	CurrentYear        int                `gorm:"default:122" json:"current_year"`                           // 当前年份
	CurrentMonth       int                `gorm:"default:1" json:"current_month"`                            // 当前月份 (1-12)
	Countdown          float64            `gorm:"default:500.0" json:"countdown"`                            // 500年倒计时 (时间胜利)
	Silo1Destroyed     bool               `gorm:"default:false" json:"silo1_destroyed"`                      // 1号地堡是否已覆灭
	RNGSeed            int64              `gorm:"default:0" json:"-"`                                        // 本局 RNG 种子 (读档时重建随机源，保证同种子复现)
	VictoryStatus      *VictoryStatus     `gorm:"type:text;serializer:json" json:"victory_status,omitempty"` // 胜利判定结果
	Resources          []Resource         `gorm:"foreignKey:SiloID" json:"resources"`
	Professions        []Profession       `gorm:"foreignKey:SiloID" json:"professions"`
	Floors             []Floor            `gorm:"foreignKey:SiloID" json:"floors"`
	Cohorts            []PopulationCohort `gorm:"foreignKey:SiloID" json:"cohorts,omitempty"`
	Residents          []Resident         `gorm:"foreignKey:SiloID" json:"residents,omitempty"`
	Factions           []Faction          `gorm:"foreignKey:SiloID" json:"factions,omitempty"`
	Relics             []Relic            `gorm:"foreignKey:SiloID" json:"relics,omitempty"` // 地堡中存在的所有遗物
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
	DeletedAt          gorm.DeletedAt     `gorm:"index" json:"-"`
}

// MayorLegitimacy is the sole legitimacy read used by simulation settlement.
// Uninitialized in-memory legacy fixtures may still read the compatibility field;
// persisted v8 saves and all new games always initialize Mayor explicitly.
func (s *Silo) MayorLegitimacy() float64 {
	if s == nil {
		return 0
	}
	if !s.Mayor.Initialized {
		return s.Legitimacy
	}
	return s.Mayor.Legitimacy
}

// SetMayorLegitimacy updates personal authority and mirrors the old readout.
func (s *Silo) SetMayorLegitimacy(value float64) {
	if s == nil {
		return
	}
	if value < 0 {
		value = 0
	} else if value > 1 {
		value = 1
	}
	s.Mayor.Initialized = true
	s.Mayor.Legitimacy = value
	s.Legitimacy = value
}

// 顶层立场三档 (16 §3.2 裁决 C/D、ADR-0008)：市长与 IT 对逃离的立场。
// 零值 ("") 与 block 同义，与旧档语义一致 (升级后零值 = 阻挠)。
// 立场是内部状态：不是玩家按钮、不是显式判定闸，仲裁不读它；它只塑形
// 出发结算的输入 (防护服/物资/监控/通道)，结算的唯一读取对象是逃出人数。
const (
	TopStanceBlock     = "block"     // 阻挠：封闸、执法追缉、审计反制 → 人数趋 0
	TopStanceNonBlock  = "nonblock"  // 不阻挠：不阻止逃离；暗中准备可行，无顶层资源与掩护
	TopStanceCooperate = "cooperate" // 配合：开防护服库、放物资、压 1 号监控 → 带走更多
)

// TopStanceState 顶层立场 (16 §3.2)：由带权重的随机决策产生，随机源走地堡
// 种子 (引擎单一 RNG 流)；权重读既有状态 (MayorLegitimacy、叛乱、W、Silo 1
// 状态、保障可信度)，但不被状态锁死——不存在强制阈值。磋商是持续过程：
// 立场可反复重估 (随状态漂移)，包括回摆到阻挠 (玉石俱焚)；最后一刻妥协也算。
// 读现任市长那一刻的立场 (裁决 E)，不读职位归属。
type TopStanceState struct {
	Initialized   bool    `json:"initialized"`
	Mayor         string  `json:"mayor"`         // ∈ ""(=block)/block/nonblock/cooperate
	IT            string  `json:"it"`            // 同上
	Guarantee     float64 `json:"guarantee"`     // 叛乱方保障可信度 (0-1)：赦免谈判、职能保留等承诺
	Reevaluations int     `json:"reevaluations"` // 审计：带权重随机重估累计次数
}

// MayorStance / ITStance 读那一刻的立场，零值归一为阻挠。
func (s TopStanceState) MayorStance() string { return normalizeTopStance(s.Mayor) }
func (s TopStanceState) ITStance() string    { return normalizeTopStance(s.IT) }

func normalizeTopStance(v string) string {
	switch v {
	case TopStanceNonBlock, TopStanceCooperate:
		return v
	default:
		return TopStanceBlock
	}
}

// 市长面对叛乱的三条路 (21 设计卷 §3.1)：带权重的随机决策档位，不是阈值
// 触发——不存在"叛乱 ≥ X → 必须妥协"。
const (
	NegotiationPathSuppress = "suppress" // 镇压：封楼、抓人、断信道
	NegotiationPathTalk     = "talk"     // 谈判：保持通气、拖延、不决议
	NegotiationPathConcede  = "concede"  // 妥协：开出/接受妥协提案
)

// 叛乱方姿态 (18 §3.3)。
const (
	NegotiationPosturePress = "press" // 施压：维持压力
	NegotiationPostureProbe = "probe" // 试探：观察市长倾向
	NegotiationPostureOffer = "offer" // 递条款：推动玩家递出提案
)

// 市长性情 (18 §2)：开信道时经引擎单一 RNG 流一次性抽取，持久保存，
// 作为后续路径决策的权重输入 (性格/特质归因)。
const (
	MayorDispositionHardliner  = "hardliner"  // 强硬派：镇压加权、妥协减权
	MayorDispositionPragmatist = "pragmatist" // 务实派：谈判加权
	MayorDispositionSurvivor   = "survivor"   // 求生派：命令源枯竭时妥协加权
)

// IT 对谈判信道的反应 (18 §3.4)：第三方独立裁量。
const (
	NegotiationITStandAside = "stand_aside" // 不干涉
	NegotiationITUndermine  = "undermine"   // 破坏：削保障可信度、可能曝光信道
	NegotiationITWatch      = "watch"       // 观望
)

// 妥协成立后叛乱方的二选一 (18 §3.7)。零值 (未选) 时结算不读加成，
// 与 R-D 抉择未选择用中间系数同口径。
const (
	NegotiationTakeover = "takeover"  // 接管市长职位：顶层改由叛乱方主导
	NegotiationHandsOff = "hands_off" // 不接管：带组织能力推进逃跑/外溢
)

// 信道崩塌原因 (合法失败结果)。
const (
	NegotiationClosedCrackdown  = "crackdown"   // 被镇压打断
	NegotiationClosedITExposure = "it_exposure" // 被 IT 曝光
)

// NegotiationLogEntry 谈判决策日志的一条：可复盘依据 (18 §2)。
// Weights 为归一化后的档位权重，Factors 为决策时刻的因子读数摘要。
type NegotiationLogEntry struct {
	Year     int                `json:"year"`
	Month    int                `json:"month"`
	Decider  string             `json:"decider"` // mayor | rebels | it
	Decision string             `json:"decision"`
	Weights  map[string]float64 `json:"weights,omitempty"`
	Factors  string             `json:"factors,omitempty"`
	Outcome  string             `json:"outcome,omitempty"`
}

// NegotiationState 市长与叛乱方的谈判/妥协层 (21 设计卷、ADR-0013)。
//
// 设计纪律：
//   - 本层只改变"谁在阻碍谁"：产出写入顶层立场的权重输入与组织事实，
//     不直接发放胜利；最终结算仍只看逃出去多少人。
//   - 三方 (市长/叛乱方/IT) 的取舍都是带权重的随机决策，权重可归因，
//     每个决策点经 Log 留下可复盘依据。
//   - 谈判是持续过程：回合可反复，拖延是合法结果；最后一刻的妥协也成立，
//     但须叛乱方接受 (接受是带权决策，不是自动成立)。
//   - 零值 = 无谈判 = 旧档语义；未开信道的运行不消耗任何新 RNG。
type NegotiationState struct {
	Initialized           bool                  `json:"initialized"`
	ChannelOpen           bool                  `json:"channel_open"`
	OpenedYear            int                   `json:"opened_year,omitempty"`
	OpenedMonth           int                   `json:"opened_month,omitempty"`
	MayorDisposition      string                `json:"mayor_disposition,omitempty"` // 开信道时一次性抽取
	Rounds                int                   `json:"rounds"`                      // 已交换回合数
	MayorPath             string                `json:"mayor_path,omitempty"`        // 市长最近一次路径决策
	MayorLean             float64               `json:"mayor_lean"`                  // 玩家可感知的市长倾向 (0-1)
	RebelPosture          string                `json:"rebel_posture,omitempty"`     // 叛乱方最近姿态
	StalledRounds         int                   `json:"stalled_rounds"`              // 连续谈判无进展回合 (谈判权重软衰减输入)
	SuppressRounds        int                   `json:"suppress_rounds"`             // 累计镇压回合 (叛乱方拒绝权重输入)
	OfferOnTable          bool                  `json:"offer_on_table"`
	OfferedBy             string                `json:"offered_by,omitempty"` // mayor | rebels
	OfferTerms            string                `json:"offer_terms,omitempty"`
	CompromiseEstablished bool                  `json:"compromise_established"`
	CompromiseYear        int                   `json:"compromise_year,omitempty"`
	CompromiseMonth       int                   `json:"compromise_month,omitempty"`
	Takeover              string                `json:"takeover,omitempty"` // ""(未选)/takeover/hands_off
	Closed                bool                  `json:"closed"`
	CloseReason           string                `json:"close_reason,omitempty"`
	Log                   []NegotiationLogEntry `json:"log,omitempty"` // 环形保留最近 24 条
}

// NegotiationLogCap 决策日志环形上限。
const NegotiationLogCap = 24

// AppendLogEntry 追加一条决策日志并夹到环形上限 (超限时丢弃最旧条目)。
func (n *NegotiationState) AppendLogEntry(entry NegotiationLogEntry) {
	if n == nil {
		return
	}
	n.Log = append(n.Log, entry)
	if len(n.Log) > NegotiationLogCap {
		n.Log = n.Log[len(n.Log)-NegotiationLogCap:]
	}
}

// NegotiationActive 报告月度回合是否应当进行：信道畅通、未崩塌、
// 妥协尚未成立 (妥协成立后谈判阶段结束，进入选择阶段)。
func (n NegotiationState) NegotiationActive() bool {
	return n.Initialized && n.ChannelOpen && !n.Closed && !n.CompromiseEstablished
}

// IT 对逃跑筹备的响应档位 (20 分册修订、ADR-0014)：组织怀疑度到门槛后，
// IT 必须做选择——带权重的随机裁量，不是阈值 if-else (ADR-0007 同构)。
const (
	EscapeResponseSuppress = "suppress" // 镇压：驱散组织、抬戒备恐慌；不触碰囤积账本
	EscapeResponseConceal  = "conceal"  // 蒙蔽：不镇压，自己把 1 号蒙住 (占 IT 资源，暴露风险随月上浮)
	EscapeResponseIgnore   = "ignore"   // 不作为：自由裁量，但承担后果 (safeguard 风险上浮)
)

// EscapeResponseLogEntry IT 响应决策日志的一条：权重可归因、事后可复盘
// (同谈判层日志口径，21 设计卷)。
type EscapeResponseLogEntry struct {
	Year     int                `json:"year"`
	Month    int                `json:"month"`
	Decision string             `json:"decision"`
	Weights  map[string]float64 `json:"weights,omitempty"` // 归一化后的档位权重
	Factors  string             `json:"factors,omitempty"` // 决策时刻的因子读数摘要
	Outcome  string             `json:"outcome,omitempty"` // 后续结果 (蒙蔽崩塌/窗口开启等)
}

// EscapeResponseLogCap 决策日志环形上限 (同谈判层)。
const EscapeResponseLogCap = 24

// EscapeResponseState 逃跑筹备的反制层状态 (20 分册修订、ADR-0014)。
//
// 设计纪律：
//   - 组织怀疑度只由组织行为 (动员范围、通信频次与规模) 驱动；囤积
//     (修服/划拨/囤备) 永不抬高怀疑度——准备本身不惹眼，惹眼的是组织行为。
//   - 反制只抬成本与风险：任何响应都不扣减囤积账本、不直接判负、
//     不困死玩家；玩家永远还有路走。
//   - 零值 = 无响应层活动 = 旧档语义 (未做任何组织动作的运行逐位不变)。
type EscapeResponseState struct {
	Initialized       bool                     `json:"initialized"`
	OrgSuspicion      float64                  `json:"org_suspicion"` // 组织怀疑度 (0-1)
	LastDecisionYear  int                      `json:"last_decision_year,omitempty"`
	LastDecisionMonth int                      `json:"last_decision_month,omitempty"`
	CoverActive       bool                     `json:"cover_active"` // IT 正在把动向蒙住，不让 1 号察觉
	CoverMonths       int                      `json:"cover_months"`
	Decisions         []EscapeResponseLogEntry `json:"decisions,omitempty"` // 环形保留最近 24 条
}

// CountermeasureState 20 分册第 5 批反制的持久账本：T1-3 关系冷却、T2-4
// 谈判可信度侵蚀、T3-1b 身份线日常绑定、T3-2 联系人监视。零值 = 无任何
// 反制生效 = 旧档语义；不含 RNG 状态，不参与月度采样。
type CountermeasureState struct {
	// RelationCooldown 部门名 → 剩余关系冷却月数 (T1-3)。只在触发月写入，
	// 同源在冷却期内不刷新、不叠加。
	RelationCooldown map[string]int `json:"relation_cooldown,omitempty"`
	// ContactWatch 部门名 → 联系人不可用剩余月数 (T3-2)。键的"存在"即该部门
	// 的联系人已被调离/问话一次 (同源一次性去重)；值归零后键保留。
	ContactWatch map[string]int `json:"contact_watch,omitempty"`
	// LocationLockMonths 身份复核期绑定：地点类动作剩余灰置月数 (T3-1b)。
	LocationLockMonths int `json:"location_lock_months,omitempty"`
	// ProfessionLockMonths 身份复查期绑定：职务动作剩余灰置月数 (T3-1b)。
	ProfessionLockMonths int `json:"profession_lock_months,omitempty"`
	// GuaranteeEroded T2-4 已结算：prep_exposure 越线只侵蚀一次保障可信度。
	GuaranteeEroded bool `json:"guarantee_eroded,omitempty"`
	// IdentityQueryBound / ArchiveReviewBound 身份线节点只绑定一次。
	IdentityQueryBound bool `json:"identity_query_bound,omitempty"`
	ArchiveReviewBound bool `json:"archive_review_bound,omitempty"`
}

// AppendDecision 追加一条决策日志并夹到环形上限 (超限时丢弃最旧条目)。
func (r *EscapeResponseState) AppendDecision(entry EscapeResponseLogEntry) {
	if r == nil {
		return
	}
	r.Decisions = append(r.Decisions, entry)
	if len(r.Decisions) > EscapeResponseLogCap {
		r.Decisions = r.Decisions[len(r.Decisions)-EscapeResponseLogCap:]
	}
}

// 接应档位 (16 §2.1)：none = 无接应事实；surveyed = 路线已勘察；
// confirmed = 先遣往返确认；stocked = 接应点已囤物资。
const (
	ReceptionNone      = "none"
	ReceptionSurveyed  = "surveyed"
	ReceptionConfirmed = "confirmed"
	ReceptionStocked   = "stocked"
)

// 预案功能位 (16 §2.2，剧情台阶 R-2 清单)：预案完备度 = 功能位覆盖率。
// 每个功能位由对应的准备动作/事件写入，权重 TODO(balance) (当前等权)。
const (
	PlanRoleMechanical   = "mechanical"   // 出口与设备可重复使用
	PlanRoleMines        = "mines"        // 通行限制与结构影响
	PlanRoleSupply       = "supply"       // 多批次物资换算
	PlanRoleMedical      = "medical"      // 暴露风险与照护
	PlanRoleAgricultural = "agricultural" // 可携带食物与生产起点
	PlanRoleOrder        = "order"        // Mayor/Police 公开秩序 (可争取非必须)
)

// PlanRoleCount 功能位总数 (派生完备度的分母)。
const PlanRoleCount = 6

// EscapePreparation 逃跑准备账本 (16 §2/§6.1)。事实账本，不是成功率系数：
// 每条记录必须有明确生产者 (玩家动作/内容事件)，禁止月度被动积累与
// AP 直接兑换。可数实物获取即付成本 (划拨即扣减、暴露即累积)。
// 空账本 = 无准备 = 旧行为逐位不变。
type EscapePreparation struct {
	Initialized bool `json:"initialized"`
	// 可数实物
	SuitsFunctional int     `json:"suits_functional"` // 可用防护服套数 (预置缓存可用部分 + 本地修复成功部分)
	SuppliesPacked  float64 `json:"supplies_packed"`  // 已划拨封存的便携补给 (人·月单位)
	MaterialsPacked float64 `json:"materials_packed"` // 已划拨的装备材料 (含生产起点载体)
	// 程度事实
	PlanCompleteness float64  `json:"plan_completeness"`         // 预案完备度 = 功能位覆盖率 (0-1)，由 PlanRoles 派生
	PlanRoles        []string `json:"plan_roles,omitempty"`      // 已完成实质对接的功能位 (升序去重)
	DrillLevel       float64  `json:"drill_level"`               // 演练水平 (0-1，按次累积)
	CareReady        float64  `json:"care_ready"`                // 照护能力 (0-1)
	ProductionStart  bool     `json:"production_start"`          // 生产起点 ("明天") 已备
	ReceptionStage   string   `json:"reception_stage,omitempty"` // 接应档位：none/surveyed/confirmed/stocked
	// 暴露记账
	PrepExposure float64 `json:"prep_exposure"` // 筹备活动暴露累计 (修服/划拨/演练可见性)
}

// RecomputePlanCompleteness 由已对接功能位重算预案完备度 (等权基线，
// 权重 TODO(balance))。PlanRoles 升序去重。
func (p *EscapePreparation) RecomputePlanCompleteness() {
	if p == nil {
		return
	}
	seen := map[string]bool{}
	roles := make([]string, 0, len(p.PlanRoles))
	for _, role := range p.PlanRoles {
		if role == "" || seen[role] {
			continue
		}
		seen[role] = true
		roles = append(roles, role)
	}
	sort.Strings(roles)
	p.PlanRoles = roles
	p.PlanCompleteness = float64(len(roles)) / PlanRoleCount // TODO(balance): 功能位权重
}

// 外溢受限信道三种 (16 §2.3，与《共证》I-6 台阶一致)。
const (
	SpilloverChannelITLine          = "it_line"          // IT 专线类：可靠度最高、暴露成本最高
	SpilloverChannelCrossSignal     = "cross_signal"     // 跨堡信号类：受邻接约束
	SpilloverChannelPhysicalHandoff = "physical_handoff" // 实体交接类：与逃跑耦合
)

// SpilloverChannel 一条已建立的受限信道 (建立侧)。投递与独立接收结算
// 属 Layer 3 (外溢模型主体)，本期不实现。
type SpilloverChannel struct {
	Kind         string  `json:"kind"`
	TargetSilo   uint    `json:"target_silo"`
	Reliability  float64 `json:"reliability"`   // TODO(balance)
	ExposureCost float64 `json:"exposure_cost"` // TODO(balance)
	Year         int     `json:"year"`
	Month        int     `json:"month"`
	// Uses 已使用次数 (Layer 3 投递侧)：T2-2 信道审计读它——使用次数到达
	// 档位门槛后可靠度降档，跌破地板则信道中断。零值 = 未使用。
	Uses int `json:"uses,omitempty"`
	// Broken 信道被审计中断 (不可再投递)；建立记录保留供审计追溯。
	Broken bool `json:"broken,omitempty"`
}

// SpilloverPreparation 外溢准备账本 (16 §2.3/§6.1)。外溢是逃跑失败后的
// 退路，不是并行的第二套准备 (用户裁决，16 §3.5)。
type SpilloverPreparation struct {
	Channels       []SpilloverChannel `json:"channels,omitempty"`
	PackageQuality float64            `json:"package_quality"` // 五种能力完备度 (0-1)
	// Deliveries 投递尝试账本 (Layer 3 发送侧)：每次投递动作记一条，含
	// 是否到达 (可靠度结算)。到达后的独立接收记录在 World.OutboundSignals。
	Deliveries []SpilloverDelivery `json:"deliveries,omitempty"`
}

// SpilloverDelivery 一次外溢投递尝试 (发送侧事实，玩家可审计)。
type SpilloverDelivery struct {
	Kind        string  `json:"kind"`
	TargetSilo  uint    `json:"target_silo"`
	Year        int     `json:"year"`
	Month       int     `json:"month"`
	ArriveYear  int     `json:"arrive_year"`
	ArriveMonth int     `json:"arrive_month"`
	Quality     float64 `json:"quality"`   // 发出时的包裹质量 (信道容量与路上损耗结算后)
	Delivered   bool    `json:"delivered"` // 可靠度结算：是否送达 (失败 = 路上丢失)
}

// 追踪四档 (16 §3.3 追踪 T 的 Layer 2 形态)：从"没人注意"到"正在追捕"。
// 档位由筹备暴露、出发清点与 1 号注意力决定；追踪不是必然失败，而是改变
// 后续的选择空间——被追着跑的人带不走东西，也走不快。
const (
	TrackingNone     = "none"     // 没人注意
	TrackingAware    = "aware"    // 已被察觉 (清点发现缺口)
	TrackingTrailing = "trailing" // 咬住痕迹 (带不走重物，弃置补给)
	TrackingPursuit  = "pursuit"  // 正在追捕 (走不快、不能觅食、逐月损耗)
)

// 群体覆没原因 (堡外维持的三种死法＋追踪歼灭)：结算写事实，叙事读事实。
const (
	DemiseFreeze  = "freeze"  // 冻死：防护服完整性挡不住当季严寒
	DemiseExhaust = "exhaust" // 耗尽：封存补给吃完且觅食不足
	DemiseTrap    = "trap"    // 被地形困住：路线安全坍塌，走不出去也觅不了食
	DemisePursuit = "pursuit" // 追捕歼灭：追踪档位的逐月损耗耗尽人口
)

// OutsideState 堡外维持状态 (16 §6.4，Layer 2)：出发结算初始化，其后每月
// 结算一次。全部字段是事实，不是系数：补给余量、防护服完整性、路线安全、
// 追踪档位与群体组合各自记录、各自出结果。零值 = 未激活 = 旧档行为不变。
type OutsideState struct {
	Active bool `json:"active"` // 堡外月度结算是否激活 (群体覆灭后置假)
	// 维持事实 (三个侧面各自结算)
	SuppliesLeft  float64 `json:"supplies_left"`  // 剩余封存补给 (人·月)
	SuitIntegrity float64 `json:"suit_integrity"` // 防护服完整性 (0-1)，随天气磨损
	RouteSafety   float64 `json:"route_safety"`   // 路线安全 (0-1)，随天气与地形退化
	TrappedMonths int     `json:"trapped_months"` // 连续被地形困住的月数
	// 追踪事实
	Tracking     string  `json:"tracking"`      // 追踪档位 (四档)
	TrackingBase float64 `json:"tracking_base"` // 出发时定档基数 (筹备暴露 + 出发清点)
	// 群体组合事实 (质量)：群体不是一个人
	Elders   int `json:"elders"`   // 老人
	Wounded  int `json:"wounded"`  // 伤者
	Artisans int `json:"artisans"` // 工匠
	Leaders  int `json:"leaders"`  // 领头人
	// 结算
	MonthsOut   int    `json:"months_out"`             // 堡外已存续月数
	DemiseCause string `json:"demise_cause,omitempty"` // 覆没原因 (仅覆灭时写)
	// 累计损耗审计 (覆没归因读主导项，并列按固定序)
	AttritionFreeze  int `json:"attrition_freeze,omitempty"`
	AttritionStarve  int `json:"attrition_starve,omitempty"`
	AttritionTrap    int `json:"attrition_trap,omitempty"`
	AttritionPursuit int `json:"attrition_pursuit,omitempty"`
}

// EscapeGroup 逃出群体模型 (G5)：已经离开驻地、在堡外可能存续的人口。
// 存续的逃出群体是《越岭》的资格事实，也是《寂堡》《遗忘》《断层》的例外：
// 原驻地毁灭/重置/崩坏时不再追溯判死该群体。
type EscapeGroup struct {
	Initialized   bool         `json:"initialized"`
	Population    int          `json:"population"`
	Departed      int          `json:"departed,omitempty"` // 出发人数 (审计：路上损耗 = Departed - Population)
	DepartedYear  int          `json:"departed_year,omitempty"`
	DepartedMonth int          `json:"departed_month,omitempty"`
	Surviving     bool         `json:"surviving"`
	Fragile       bool         `json:"fragile,omitempty"` // 脆弱存续标记 (维存缺口或追踪压力，供堡外月度检查)
	Outside       OutsideState `json:"outside"`           // 堡外维持状态 (Layer 2)
}

// SurvivingEscapeGroup 报告是否存在真实可读的存续逃出群体 (结局体系 12 §6：
// 硬失败护栏先读逃出群体)。人口必须为正，未初始化或已覆灭的群体不算。
func (g EscapeGroup) SurvivingEscapeGroup() bool {
	return g.Initialized && g.Surviving && g.Population > 0
}

// PreFallIdeologySnapshot 覆灭前夕思潮快照 (G4／《无人裁决》思潮闸)。
// 在 Silo 1 覆灭事实置位的瞬间、覆灭余波制造新恐慌之前截取；读取的是
// Cohort 的人口加权思潮，不是部门派生视图。
type PreFallIdeologySnapshot struct {
	Taken      bool               `json:"taken"`
	Year       int                `json:"year,omitempty"`
	Month      int                `json:"month,omitempty"`
	Population int                `json:"population"`
	Ideologies map[string]float64 `json:"ideologies,omitempty"` // 人口加权 (0-1)
	// DominantAxis 权重最高的思潮轴；并列时按轴名字典序取小者 (确定性)。
	DominantAxis string `json:"dominant_axis,omitempty"`
	// OpenQualified 亲外思潮为主导或复合主导一极即合格；纯种地堡思潮
	// (忠诚主导) 一律不合格 (结局体系 12 §3.3)。
	OpenQualified bool `json:"open_qualified"`
}

// EndingEntityContribution records one entity's weighted support for each outcome.
type EndingEntityContribution struct {
	EntityID string             `json:"entity_id"`
	Kind     string             `json:"kind"`
	Name     string             `json:"name"`
	Scores   map[string]float64 `json:"scores"`
}

// EndingDecision explains the deterministic multi-entity outcome arbitration.
type EndingDecision struct {
	ConfigVersion int                        `json:"config_version"`
	Selected      string                     `json:"selected"`
	MinimumScore  float64                    `json:"minimum_score"`
	Scores        map[string]float64         `json:"scores"`
	Contributions []EndingEntityContribution `json:"contributions"`
	// v9 审计字段 (结局体系 12 §9)：解释哪些候选进入评分、哪些因何资格失败、
	// 哪个硬失败护栏或同分规则生效。Selected 为空串表示"继续模拟"的空池裁决。
	Reason   string            `json:"reason,omitempty"`
	Pool     []string          `json:"pool,omitempty"`
	Excluded map[string]string `json:"excluded,omitempty"`
	// Explanations 候选结局的四维审计解释 (10 §9：动机/介入/证据/附着)。
	Explanations map[string]string `json:"explanations,omitempty"`
	// Recap 本局关键抉择回读：按时序投影 3–6 条因果相关条目 (何时/哪条线/
	// 哪一支/后果)。文案取自既有事件定义与完成键事实；缺料时为空，呈现端
	// 退化为既有结算行。纯增量字段，旧档反序列化为 nil，不升存档版本。
	Recap []EndingRecapEntry `json:"recap,omitempty"`
}

// EndingRecapEntry 结局回读的一条关键抉择。Line 为空表示不隶属五条可玩线
// (如失败征兆、跨线反制)；Title/Description 复用内容事件的既有文案。
type EndingRecapEntry struct {
	Year        int    `json:"year"`
	Month       int    `json:"month"`
	Line        string `json:"line,omitempty"`
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Consequence string `json:"consequence,omitempty"`
}

// VictoryStatus 胜利状态
type VictoryStatus struct {
	IsWon       bool            `json:"is_won"`
	Type        string          `json:"type"` // NONE/INFORMATION/TIME/REBELLION/RULE_OF_LAW/PERSONAL_RULE/EXCLUSIONIST/RESET/FRACTURE/DEATH/AGENT_COMPROMISED
	Description string          `json:"description"`
	Score       *GameScore      `json:"score,omitempty"`
	Decision    *EndingDecision `json:"decision,omitempty"`
}

// GameScore 评分结算
type GameScore struct {
	Total            int                    `json:"total"`
	SurvivalPoints   int                    `json:"survival_points"`
	DiversityPoints  int                    `json:"diversity_points"`
	HeritagePoints   int                    `json:"heritage_points"`
	IdeologyPoints   int                    `json:"ideology_points"`
	Multiplier       float64                `json:"multiplier"`
	HistoryInfluence *HistoryInfluenceScore `json:"history_influence,omitempty"`
}

// HistoryImpact records measured deltas caused by one resolved player action.
type HistoryImpact struct {
	ID              int     `json:"id"`
	OperationID     int     `json:"operation_id"`
	TargetSiloID    uint    `json:"target_silo_id"`
	TargetName      string  `json:"target_name"`
	ActionID        string  `json:"action_id"`
	Label           string  `json:"label"`
	Year            int     `json:"year"`
	Month           int     `json:"month"`
	LegitimacyDelta float64 `json:"legitimacy_delta"`
	CohesionDelta   float64 `json:"cohesion_delta"`
	RebellionDelta  float64 `json:"rebellion_delta"`
	ProForeignDelta float64 `json:"pro_foreign_delta"`
	SuppliesDelta   float64 `json:"supplies_delta"`
	Points          int     `json:"points"`
	Summary         string  `json:"summary"`
}

// HistorySiloScore explains the player's measured influence on one silo.
type HistorySiloScore struct {
	SiloID         uint            `json:"silo_id"`
	Name           string          `json:"name"`
	Points         int             `json:"points"`
	OperationCount int             `json:"operation_count"`
	Summary        string          `json:"summary"`
	Impacts        []HistoryImpact `json:"impacts,omitempty"`
}

// HistoryInfluenceScore is added once to the original ending score.
type HistoryInfluenceScore struct {
	Total   int                `json:"total"`
	Grade   string             `json:"grade"`
	Summary string             `json:"summary"`
	Silos   []HistorySiloScore `json:"silos"`
}

// Resource 资源模型
type Resource struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	SiloID     uint      `gorm:"index" json:"silo_id"`
	Type       string    `gorm:"size:20" json:"type"` // Energy, Materials, Supplies
	Amount     float64   `gorm:"default:0" json:"amount"`
	NetBalance float64   `gorm:"default:0" json:"net_balance"` // 资源结余
	UpdatedAt  time.Time `json:"updated_at"`
}

// Profession 职业/部门模型
type Profession struct {
	ID           uint               `gorm:"primarykey" json:"id"`
	SiloID       uint               `gorm:"index" json:"silo_id"`
	Name         string             `gorm:"size:50" json:"name"`
	ClassType    string             `gorm:"size:20" json:"class_type"` // ELITE / COMMONER
	Population   int                `gorm:"default:0" json:"population"`
	Ideologies   map[string]float64 `gorm:"type:text;serializer:json" json:"ideologies"` // 各项思潮值 (0-1)
	PanicValue   float64            `gorm:"default:0.0" json:"panic_value"`              // 恐慌值
	Productivity float64            `gorm:"default:1.0" json:"productivity"`             // 生产力值
	// Alertness 部门戒备 (10-it-control-credibility §3.4 的 alert_d)：失败探查与
	// 信息管控抬高，压低对该部门的碎片获取概率。旧档缺省 0。
	Alertness      float64            `gorm:"default:0.0" json:"alertness"`
	PowerLevel     int                `gorm:"default:1" json:"power_level"`
	Zone           string             `gorm:"size:20" json:"zone"`
	KnownFragments []string           `gorm:"type:text;serializer:json" json:"known_fragments"`     // 掌握的其他部门信息碎片来源
	Relations      map[string]float64 `gorm:"type:text;serializer:json" json:"relations,omitempty"` // NPC部门之间的人脉/关系网
	// Beliefs 部门层信念 (§3.3 的部门×阶层层，本轮以部门为聚合单元)。
	Beliefs []FactBelief `gorm:"type:text;serializer:json" json:"beliefs,omitempty"`
	// ---- 统一 Actor 经济体系 (与 Agent 同构) ----
	ActionPoints    float64   `gorm:"default:0" json:"action_points"`                    // 行动点数 (AP)
	SuspicionLevel  float64   `gorm:"default:0" json:"suspicion_level"`                  // 怀疑度
	Ambition        float64   `gorm:"default:0" json:"ambition"`                         // 部门野心 (开局定死，仅事件可改)
	Influence       float64   `gorm:"default:0" json:"influence"`                        // 部门影响力
	PropagandaLevel float64   `gorm:"default:0" json:"propaganda_level"`                 // 宣传力度
	Traits          []string  `gorm:"type:text;serializer:json" json:"traits,omitempty"` // 特质
	Relics          []Relic   `gorm:"foreignKey:ProfessionID" json:"relics,omitempty"`   // 部门保管的遗物 (如司法部)
	UpdatedAt       time.Time `json:"updated_at"`
}

// Floor 楼层模型 (Silo 核心空间)
type Floor struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	SiloID     uint      `gorm:"index" json:"silo_id"`
	Level      int       `gorm:"index" json:"level"` // 1-144
	Function   string    `gorm:"size:50" json:"function"`
	Zone       string    `gorm:"size:20" json:"zone"`
	Stability  float64   `gorm:"default:1.0" json:"stability"` // 楼层稳定性
	Population int       `gorm:"default:0" json:"population"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// PopulationCohort 聚合人口单元：以职业为首要维度，按意识形态组合划分。
type PopulationCohort struct {
	ID               uint               `gorm:"primarykey" json:"id"`
	SiloID           uint               `gorm:"index" json:"silo_id"`
	ProfessionID     uint               `gorm:"index" json:"profession_id"`
	FactionID        *uint              `gorm:"index" json:"faction_id,omitempty"`
	Name             string             `gorm:"size:100" json:"name"`
	Count            int                `gorm:"default:0" json:"count"`
	IdeologyProfile  []string           `gorm:"type:text;serializer:json" json:"ideology_profile"` // 核心意识形态组合 (1-2个)
	HomeZone         string             `gorm:"size:20" json:"home_zone"`
	Loyalty          float64            `gorm:"default:0.5" json:"loyalty"`
	Influence        float64            `gorm:"default:0.0" json:"influence"`
	ActionPoints     float64            `gorm:"default:0.0" json:"action_points"`
	PanicSensitivity float64            `gorm:"default:1.0" json:"panic_sensitivity"`
	Ideologies       map[string]float64 `gorm:"type:text;serializer:json" json:"ideologies"`
	KnownFragments   []string           `gorm:"type:text;serializer:json" json:"known_fragments,omitempty"`
	Tags             []string           `gorm:"type:text;serializer:json" json:"tags,omitempty"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

// Resident 显式居民/单位模型：每个 NPC unit 都有自己的 tags 与状态
type Resident struct {
	ID              uint               `gorm:"primarykey" json:"id"`
	SiloID          uint               `gorm:"index" json:"silo_id"`
	Name            string             `gorm:"size:80" json:"name"`
	CohortID        *uint              `gorm:"index" json:"cohort_id,omitempty"`
	ProfessionID    uint               `gorm:"index" json:"profession_id"`
	Profession      string             `gorm:"size:50" json:"profession"`
	FactionID       *uint              `gorm:"index" json:"faction_id,omitempty"`
	HomeFloor       int                `json:"home_floor"`
	Tags            []string           `gorm:"type:text;serializer:json" json:"tags"`
	Loyalty         float64            `gorm:"default:0.5" json:"loyalty"`
	Ambition        float64            `gorm:"default:0.0" json:"ambition"`
	Ideologies      map[string]float64 `gorm:"type:text;serializer:json" json:"ideologies"`
	Influence       float64            `gorm:"default:0.0" json:"influence"`
	ActionPoints    float64            `gorm:"default:0.0" json:"action_points"`
	SuspicionLevel  float64            `gorm:"default:0.0" json:"suspicion_level"`
	PropagandaLevel float64            `gorm:"default:0.0" json:"propaganda_level"`
	KnownFragments  []string           `gorm:"type:text;serializer:json" json:"known_fragments"`
	Relations       map[string]float64 `gorm:"type:text;serializer:json" json:"relations,omitempty"`
	// Beliefs 具名人物层信念 (§3.3)：公开传播与关键人物确信进入 E 聚合。
	Beliefs          []FactBelief `gorm:"type:text;serializer:json" json:"beliefs,omitempty"`
	IsRepresentative bool         `gorm:"default:false" json:"is_representative"`
	// Alive 不带 gorm default：default:true 会让零值 (false=死亡) 在 INSERT 时
	// 被跳过而被数据库默认值复活。代码侧创建居民时一律显式置 Alive。
	Alive     bool      `json:"alive"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Relic 遗物模型
type Relic struct {
	ID            uint               `gorm:"primarykey" json:"id"`
	SiloID        uint               `gorm:"index" json:"silo_id"`
	Name          string             `gorm:"size:100" json:"name"`
	Description   string             `gorm:"type:text" json:"description"`
	SourceDept    string             `gorm:"size:50" json:"source_dept"`
	DiscoveryYear int                `json:"discovery_year"`
	Effects       map[string]float64 `gorm:"type:text;serializer:json" json:"effects"`
	// LocationTag 开局实例化时为该遗物选定的藏匿位置，引用清单位置词表
	// (events/relics.json locations[].id)。未被发现/没收前一直停留在此地。
	LocationTag string `gorm:"size:64" json:"location_tag,omitempty"`

	// 归属关系
	AgentID      *uint `gorm:"index" json:"agent_id,omitempty"`      // 如果被特工私吞
	ProfessionID *uint `gorm:"index" json:"profession_id,omitempty"` // 如果在部门（如司法部）保管

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Faction 派系模型：独立于 Profession，由 resident tags 隐式聚类生成
type Faction struct {
	ID                       uint           `gorm:"primarykey" json:"id"`
	SiloID                   uint           `gorm:"index" json:"silo_id"`
	Name                     string         `gorm:"size:100" json:"name"`
	Signature                string         `gorm:"size:120" json:"signature"`
	Tags                     []string       `gorm:"type:text;serializer:json" json:"tags"`
	TagStats                 map[string]int `gorm:"type:text;serializer:json" json:"tag_stats,omitempty"` // 阵营内部标签分布统计
	MemberCount              int            `gorm:"default:0" json:"member_count"`
	RepresentativeResidentID uint           `gorm:"default:0" json:"representative_resident_id"`
	RepresentativeCohortID   *uint          `gorm:"index" json:"representative_cohort_id,omitempty"`
	RepresentativeName       string         `gorm:"size:80" json:"representative_name"`
	Influence                float64        `gorm:"default:0.0" json:"influence"`
	Cohesion                 float64        `gorm:"default:0.0" json:"cohesion"`
	Prestige                 float64        `gorm:"default:0.0" json:"prestige"`    // 阵营威望
	IsPublic                 bool           `gorm:"default:false" json:"is_public"` // 是否已向全地堡公开存在
	// P1-S3 persistent identity: Stance is fixed at founding and inherited
	// with the ID; Memory carries the founding record (event-system readable);
	// Demands re-derive every tick from the dominant member cohort's drive.
	Stance    string    `gorm:"size:120" json:"stance,omitempty"`
	Memory    []string  `gorm:"type:text;serializer:json" json:"memory,omitempty"`
	Demands   []string  `gorm:"type:text;serializer:json" json:"demands,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// StoryEventLog 剧情事件历史记录 (用于持久化与 debug 查询)
type StoryEventLog struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	SiloID      uint      `gorm:"index" json:"silo_id"`
	Timestamp   int       `gorm:"index" json:"timestamp"` // 以 Y100-M1-D1 为 0，按 30 天/月折算
	Year        int       `gorm:"index" json:"year"`
	Month       int       `gorm:"index" json:"month"`
	Source      string    `gorm:"size:64" json:"source"` // PASS_TIME / ACTION:<type>
	EventID     string    `gorm:"size:100" json:"event_id"`
	Category    string    `gorm:"size:32" json:"category"` // history / special / crisis / player_action
	Title       string    `gorm:"size:120" json:"title"`
	Description string    `gorm:"type:text" json:"description"`
	Type        string    `gorm:"size:32" json:"type"` // SOCIAL / TECHNICAL / EXTERNAL
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ContentEventDefinition 文件驱动的事件定义（来自 events/ 下的各个分组目录）
type ContentEventDefinition struct {
	ID             uint   `gorm:"primarykey" json:"id"`
	Key            string `gorm:"uniqueIndex;size:180" json:"key"`
	SourceGroup    string `gorm:"index;size:32" json:"source_group"` // histories / special / crisis / player_actions / legacy events
	SourceFile     string `gorm:"size:255" json:"source_file"`
	SourceFormat   string `gorm:"size:24" json:"source_format"`
	EventID        string `gorm:"index;size:100" json:"event_id"`
	Title          string `gorm:"size:120" json:"title"`
	Description    string `gorm:"type:text" json:"description"`
	Type           string `gorm:"size:32" json:"type"`
	FireMode       string `gorm:"size:32" json:"fire_mode"` // ONCE / REPEATABLE
	CooldownMonths int    `gorm:"default:0" json:"cooldown_months"`
	// DeadlineYear / DeadlineMonth mirror the engine definition's force-raise
	// fallback (硬伤 #3): an untriggered event past its deadline fires once.
	DeadlineYear     int       `gorm:"default:0" json:"deadline_year"`
	DeadlineMonth    int       `gorm:"default:0" json:"deadline_month"`
	Enabled          bool      `gorm:"default:true" json:"enabled"`
	TriggerSpec      string    `gorm:"type:text" json:"trigger_spec"`
	EffectsSpec      string    `gorm:"type:text" json:"effects_spec"`
	PlayerActionSpec string    `gorm:"type:text" json:"player_action_spec"`
	ScriptSource     string    `gorm:"type:text" json:"script_source"`
	ScriptCanTrigger bool      `gorm:"default:false" json:"script_can_trigger"`
	ScriptApply      bool      `gorm:"default:false" json:"script_apply"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ContentEventState 单局游戏内文件驱动事件的触发状态
type ContentEventState struct {
	ID                     uint      `gorm:"primarykey" json:"id"`
	SiloID                 uint      `gorm:"index" json:"silo_id"`
	DefinitionID           uint      `gorm:"index" json:"definition_id"`
	EventKey               string    `gorm:"index;size:180" json:"event_key"`
	Triggered              bool      `gorm:"default:false" json:"triggered"`
	TriggerCount           int       `gorm:"default:0" json:"trigger_count"`
	LastTriggeredTimestamp int       `gorm:"default:0" json:"last_triggered_timestamp"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// ============ 游戏交互 DTO (前后端通信) ============

// AgentActionType 动作类型
type AgentActionType string

const (
	ActionGatherInfo AgentActionType = "GATHER_INFO"
	// ActionSilentProbe 无声探查 (19-opening-variability §4.2)：隐秘行事特质
	// 解锁的探查变体——效果同 GATHER_INFO，但检测分类强制 UNSAMPLED (不进
	// 抽样、无怀疑增量)，AP 成本为普通探查的 2 倍。
	ActionSilentProbe       AgentActionType = "SILENT_PROBE"
	ActionShareInfo         AgentActionType = "SHARE_INFO"
	ActionBuildConnection   AgentActionType = "BUILD_CONNECTION"
	ActionInciteRebellion   AgentActionType = "INCITE_REBELLION"
	ActionConductPropaganda AgentActionType = "CONDUCT_PROPAGANDA"
	ActionProfession        AgentActionType = "PROFESSION_ACTION"
	ActionPlayerEvent       AgentActionType = "PLAYER_EVENT"
	ActionPublicizeFaction  AgentActionType = "PUBLICIZE_FACTION"
	ActionSilo1Operation    AgentActionType = "SILO1_OPERATION"
	ActionStationOperation  AgentActionType = "STATION_OPERATION"
)

// AgentAction 特工/部门动作请求
type AgentAction struct {
	Type             AgentActionType `json:"type"`
	ActionID         string          `json:"action_id,omitempty"`
	TargetDept       string          `json:"target_dept,omitempty"`
	FragmentIds      []string        `json:"fragment_ids,omitempty"`
	ProfessionAction string          `json:"profession_action,omitempty"`
	ResourceTarget   string          `json:"resource_target,omitempty"`
	Cost             float64         `json:"cost"`
}

// ActionResult 动作执行结果
type ActionResult struct {
	Executed bool   `json:"executed"`
	Message  string `json:"message"`
}

// StoryEvent 剧情随机事件 (响应用)
type StoryEvent struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Type        string `json:"type"` // SOCIAL / TECHNICAL / EXTERNAL
	// FireMode 事件的点火方式 (ONCE / REPEATABLE)：邻堡谣言过滤用——
	// 一次性转折值得传出去，可重复的日常不值得。
	FireMode string `json:"fire_mode,omitempty"`
	// 来源堡归属 (月度反馈用)：多堡同月触发相同内容时，玩家需要分清是哪座堡的事。
	// 仅下发快照使用，不参与持久化 (持久化的 StoryEventLog 自带 SiloID)。
	SiloID   uint   `json:"silo_id,omitempty"`
	SiloName string `json:"silo_name,omitempty"`
}

// CreateGameRequest 新建游戏请求
type CreateGameRequest struct {
	SiloName     string              `json:"silo_name"`
	StartYear    int                 `json:"start_year"`
	TraitIds     []string            `json:"trait_ids"`
	AgentName    string              `json:"agent_name"`
	Profession   string              `json:"profession"`
	Seed         int64               `json:"seed,omitempty"`          // 可选：显式指定本局 RNG 种子 (0 = 按 SILO40_SEED/随机决议)
	Archetype    string              `json:"archetype,omitempty"`     // 可选：驻地原型覆写 (19 §3.3，同 Seed/Pacing 口径)；缺省 = 按号数确定性查表派生
	Pacing       *WorldPacing        `json:"pacing,omitempty"`        // 可选：节奏定制 (缺省 = 全 1.0，行为与历史逐位一致)
	Neighbor     *NeighborSiloConfig `json:"neighbor,omitempty"`      // Optional Silo 18; nil preserves the legacy path.
	IncludeSilo1 bool                `json:"include_silo1,omitempty"` // Opt in to a fully simulated peer Silo 1.
	// Silo1BlackBox 权衡层功能开关 (22 黑箱学习系列)：开启后 1 号黑箱回合、
	// 4.55/4.65 步骤与指标一/二启用；关闭 (默认) 时行为与既有基线逐位一致。
	Silo1BlackBox bool `json:"silo1_black_box,omitempty"`
}

// NeighborSiloConfig 双地堡模式的启用开关。第二堡固定为 18 号地堡并跑
// 原著事件链 (作者裁决)：完整模拟 + 确定性 AI 群演 + 正典链条。字段仅为
// API 兼容保留，身份/前史/具名人物全部为正典常量，不受配置影响。
type NeighborSiloConfig struct {
	Name       string `json:"name"`       // 已废弃：固定 "18 号地堡"
	AgentName  string `json:"agent_name"` // 已废弃：固定 "18 号 AI 操作员"
	Profession string `json:"profession"` // 已废弃：固定 "IT"
}

// WorldSiloRef 世界视图中对地堡的最小引用。
type WorldSiloRef struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	IsPlayer  bool   `json:"is_player"`
	IsStation bool   `json:"is_station"`
}

// RumorView 迷雾下玩家对外部地堡的模糊认知 (多堡研究 §4.4)：只给量化档位，
// 不给精确值；玩家堡自身的真实值照常下发。
type RumorView struct {
	SiloID         uint   `json:"silo_id"`
	Name           string `json:"name"`
	Status         string `json:"status"`                    // normal / unrest / silent / destroyed
	LastConfirmed  int    `json:"last_confirmed"`            // 最后确认时间戳 (游戏历)
	SignalSource   string `json:"signal_source"`             // rumor / it_line
	RebellionBand  string `json:"rebellion_band,omitempty"`  // IT 信道附加：low / mid / high
	PopulationBand string `json:"population_band,omitempty"` // IT 信道附加：low / mid / high
}

// WorldView 下发前端的世界级快照 (增量字段，不破坏单堡消费方)。
type WorldView struct {
	ID                     uint             `json:"id"`
	Seed                   int64            `json:"seed"`
	CurrentYear            int              `json:"current_year"`
	CurrentMonth           int              `json:"current_month"`
	PlayerSiloID           uint             `json:"player_silo_id"`
	ProgressStage          string           `json:"progress_stage"`
	Pacing                 WorldPacing      `json:"pacing"`
	Silo1Alive             bool             `json:"silo1_alive"`
	Silo1Simulated         bool             `json:"silo1_simulated"`
	Silo1Departments       []string         `json:"silo1_departments,omitempty"`
	Silo1LastReport        string           `json:"silo1_last_report,omitempty"`
	Silo1Exposure          float64          `json:"silo1_exposure"`
	Silo1CounterintelStage int              `json:"silo1_counterintel_stage"`
	Silo1Operations        []Silo1Operation `json:"silo1_operations,omitempty"`
	StationSiloID          uint             `json:"station_silo_id,omitempty"`
	// 驻地不再整堡重复序列化：StationSiloID 即引用，驻地完整快照随 GameState.Silo
	// 单份下发 (二者为同一堡)，避免每个回合快照重复携带 floors/residents 等死重量。
	StationLastReport        string           `json:"station_last_report,omitempty"`
	StationExposure          float64          `json:"station_exposure"`
	StationCounterintelStage int              `json:"station_counterintel_stage"`
	StationOperations        []Silo1Operation `json:"station_operations,omitempty"`
	Silos                    []WorldSiloRef   `json:"silos"`
	Neighbors                []RumorView      `json:"neighbors,omitempty"` // 非玩家堡的迷雾视图 (P0.5)
	// 1 号权衡层对玩家是黑箱 (22 黑箱学习系列 04 卷)：Θ、信念与决策日志一律不进
	// 玩家界面，玩家经 Order 渠道感知 r 档位变化 (故事事件)。这里只下发"功能是
	// 否启用"供呈现端决定要不要渲染既有的 1 号动向骨架。
	StewardshipEnabled bool `json:"stewardship_enabled,omitempty"`
}

// GameState 后端权威游戏状态快照 (返回给前端)
type GameState struct {
	Silo                Silo                   `json:"silo"`
	Agent               Agent                  `json:"agent"`
	AgentStats          AgentStats             `json:"agent_stats"`
	GameOver            bool                   `json:"game_over"`
	EndingNarrative     string                 `json:"ending_narrative,omitempty"`
	VictoryStatus       *VictoryStatus         `json:"victory_status,omitempty"`
	ProfessionActions   []ProfessionActionMeta `json:"profession_actions,omitempty"`
	AvailableActions    []PlayerActionMeta     `json:"available_actions,omitempty"`
	World               *WorldView             `json:"world,omitempty"`
	InformationProgress *InformationProgress   `json:"information_progress,omitempty"`
	LinesProgress       *LinesProgress         `json:"lines_progress,omitempty"`
	WitnessTree         *WitnessTree           `json:"witness_tree,omitempty"`
}

// InformationChainNode 是《共证》链上单个节点的点亮状态。链的推进只存在于
// 内容事件状态里，事件历史会被新事件挤出，玩家此前没有任何途径看到"复制
// 走到第几节、活链是否成立"——本结构把链节点状态提升为一等进度。
type InformationChainNode struct {
	Key   string `json:"key"`
	Title string `json:"title"`
	Lit   bool   `json:"lit"`
	Count int    `json:"count,omitempty"`
}

// DepartmentFactCoverage 单个部门在结局门槛下的事实覆盖：信念层里有多少份
// 不同事实的浓度达到分享地板。
type DepartmentFactCoverage struct {
	Name       string `json:"name"`
	AboveFloor int    `json:"above_floor"`
	// Matured 已完成渗透 (成熟) 的事实数——结局资格门读取的量
	// (18-time-permeation §3.1)。AboveFloor ≥ 3 但 Matured 不足时，
	// 界面必须解释"还差多少个月成熟"，不能让玩家以为门槛已达成。
	Matured int `json:"matured"`
	// Permeating 正在渗透 (已过地板、尚未成熟) 的事实剩余月数，
	// 按剩余月数升序；跌破地板的事实计数清零，不在此列。
	Permeating []PermeationProgress `json:"permeating,omitempty"`
}

// PermeationProgress 一条部门信念的渗透进度 (18-time-permeation §7.2 反馈面)。
type PermeationProgress struct {
	FactID     string `json:"fact_id"`
	MonthsLeft int    `json:"months_left"`
}

// InformationProgress 汇总《共证》线进度。信念层原始值已在
// Silo.Professions[].Beliefs，登记簿在 Silo.PublicEvidence，Q 在
// Authority.Judicial.PublicEvidenceQ；本结构补上玩家看不到的链节点状态，
// 并随附门槛常量，让前端能解释"还差什么"。
type InformationProgress struct {
	Nodes         []InformationChainNode   `json:"nodes"`
	Departments   []DepartmentFactCoverage `json:"departments"`
	GateFactCount int                      `json:"gate_fact_count"`
	BeliefFloor   float64                  `json:"belief_floor"`
	// PermeationMonths 门槛读取的渗透月数 (18-time-permeation §4)：
	// 事实 B 连续过地板满该月数才成熟。前端用它解释"还差多少个月"。
	PermeationMonths int `json:"permeation_months"`
	// 终止链读数 (开局自解释)：任务判据三要求驻地不再受 Silo 1 单方
	// 终止链支配。先兆出现与反制成立都是内容状态里既有的事实，这里只读
	// 下发，让目标卡能回答"终止链现在怎样"。
	ResetOmenSeen           bool `json:"reset_omen_seen"`
	ResetCountermeasureDone bool `json:"reset_countermeasure_done"`
}

// 线信息架构：一局只有一条主线，其余是"还开着的路"。五条可玩结局线各由
// 里程碑事实键定义；主线由里程碑深度确定性推出，改道持久化为日志。
// 跨线共享的事实 (信念覆盖、登记簿、W、警察服从……) 不在线卡里重复，
// 由呈现端从既有快照单点读取。
const (
	LineInformation  = "information"   // 《共证》
	LineRebellion    = "rebellion"     // 《越岭》
	LineRuleOfLaw    = "rule_of_law"   // 《公议》
	LinePersonalRule = "personal_rule" // 《人治》
	LineTime         = "time"          // 《无人裁决》
)

// LineMilestone 单条线上的一个里程碑事实键的点亮状态。
type LineMilestone struct {
	Key   string `json:"key"`
	Title string `json:"title"`
	Lit   bool   `json:"lit"`
	Count int    `json:"count,omitempty"`
}

// LineProgress 单条结局线的玩家可见进度。
//
// 状态语义：
//   - Depth>0：线上已有里程碑事实，玩家在这条路上留下了脚印。
//   - Qualified：线的硬资格当前成立 (与仲裁读同一组事实)。
//   - Regressed：里程碑已亮但当前资格不成立——《人治》回摆即此态：
//     路没有消失，是秩序摆回去了。
//   - Closed：线被事实关闭 (如《公议》开窗期已过而未锁定)。
type LineProgress struct {
	Line         string          `json:"line"`
	Title        string          `json:"title"`
	Depth        int             `json:"depth"`
	LastAdvanced int             `json:"last_advanced,omitempty"` // 最近一次里程碑点亮的世界时刻 (天)
	Qualified    bool            `json:"qualified"`
	QualNote     string          `json:"qual_note,omitempty"` // 资格未成立时的具体原因 (与仲裁同一口径)
	Regressed    bool            `json:"regressed,omitempty"`
	Closed       bool            `json:"closed,omitempty"`
	CloseNote    string          `json:"close_note,omitempty"`
	Milestones   []LineMilestone `json:"milestones"`
}

// OmenFact 一个已发生的失败线征兆。征兆只报告已经发生的事实，不点名
// 结局、不预测概率——提示的分寸止于"事实已被看见"。
type OmenFact struct {
	Key   string `json:"key"`
	Title string `json:"title"`
	Count int    `json:"count,omitempty"`
	Year  int    `json:"year,omitempty"`
	Month int    `json:"month,omitempty"`
}

// LinesProgress 五条结局线的统一进度快照 (只读下发，不参与结算)。
type LinesProgress struct {
	MainLine string         `json:"main_line,omitempty"` // 空 = 主线未定 (尚无里程碑)
	Lines    []LineProgress `json:"lines"`
	Shifts   []LineShift    `json:"shifts,omitempty"`
	Omens    []OmenFact     `json:"omens,omitempty"`
}

// 见证者成就树 (ADR-0004)：国策树即成就系统。点亮节点的是可核验的成就——
// 驻地里真实发生、别人也能确认的事实，不是玩家按钮或意愿。整棵树是结局
// 资格事实与事件链完成键的只读派生快照：不裁决结局、不发放奖励、不引入
// 任何新的持久化状态 (锁 = 资格否定的显形，同局永久；新开一局事实重置，
// 树随之重置)。
type WitnessTree struct {
	Branches []WitnessBranch `json:"branches"`
	// Shadow 树的影子：判据四——不是节点，是所有节点的永久前提。
	Shadow string `json:"shadow,omitempty"`
}

// WitnessBranch 树上一条分支 (主干／三条胜利分支／收敛层)。
type WitnessBranch struct {
	Branch string        `json:"branch"` // trunk / info / rebellion / time / convergence
	Title  string        `json:"title"`
	Note   string        `json:"note,omitempty"` // 分支级事实说明 (如整支被事实锁定)
	Nodes  []WitnessNode `json:"nodes"`
}

// WitnessNode 树上的一个节点。状态语义：
//   - Lit：成就已经发生 (事实点亮，不是按钮点亮)。
//   - Locked：被事实锁定——同局内永久不可解锁，节点保留在树上可见。
//   - Reserved：预留节点，对应机制未实装，不假装可点亮。
type WitnessNode struct {
	ID       string          `json:"id"`
	Title    string          `json:"title"`
	Lit      bool            `json:"lit"`
	Locked   bool            `json:"locked,omitempty"`
	Reserved bool            `json:"reserved,omitempty"`
	Count    int             `json:"count,omitempty"`
	LockNote string          `json:"lock_note,omitempty"` // 锁的事实来源 (为什么关上了)
	Missing  []string        `json:"missing,omitempty"`   // 还差什么 (行动指南，只报事实)
	Guide    string          `json:"guide,omitempty"`     // 行动指南 (文档口径)
	Note     string          `json:"note,omitempty"`      // 点亮说明 (哪个子事实点的亮)
	Choices  []WitnessChoice `json:"choices,omitempty"`   // 互斥点子项 (二选一/三选一)
}

// WitnessChoice 互斥点里的一个选项：点亮其一，其余同局永久锁定。
type WitnessChoice struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Lit    bool   `json:"lit"`
	Locked bool   `json:"locked,omitempty"`
}

// AgentStats 后端下发的 Agent 派生指标，用于前端展示，避免 UI 复算引擎公式
type AgentStats struct {
	AvgConnection        float64 `json:"avg_connection"`
	Ambition             float64 `json:"ambition"`
	Influence            float64 `json:"influence"`
	PoliticalPrestige    float64 `json:"political_prestige"`
	APBaseRecovery       float64 `json:"ap_base_recovery"`
	APPrestigeBonus      float64 `json:"ap_prestige_bonus"`
	APTotalRecovery      float64 `json:"ap_total_recovery"`
	APMax                float64 `json:"ap_max"`
	PropagandaLevel      float64 `json:"propaganda_level"`
	PropagandaMultiplier float64 `json:"propaganda_multiplier"`
	RebellionBaseEffect  float64 `json:"rebellion_base_effect"`
	IsFactionLeader      bool    `json:"is_faction_leader"`
}

// TickResult 时间推进结果
type TickResult struct {
	Silo             Silo               `json:"silo"`
	Agent            Agent              `json:"agent"`
	AgentStats       AgentStats         `json:"agent_stats"`
	AvailableActions []PlayerActionMeta `json:"available_actions,omitempty"`
	World            *WorldView         `json:"world,omitempty"`
	Logs             []string           `json:"logs"`
	Stories          []StoryEvent       `json:"stories"`
	GameOver         bool               `json:"game_over"`
	EndingNarrative  string             `json:"ending_narrative,omitempty"`
}

// ActionOutcome 动作执行结果 (含最新状态)
type ActionOutcome struct {
	Silo             Silo               `json:"silo"`
	Agent            Agent              `json:"agent"`
	AgentStats       AgentStats         `json:"agent_stats"`
	AvailableActions []PlayerActionMeta `json:"available_actions,omitempty"`
	World            *WorldView         `json:"world,omitempty"`
	Result           ActionResult       `json:"result"`
	Logs             []string           `json:"logs"`
	Stories          []StoryEvent       `json:"stories"`
	GameOver         bool               `json:"game_over"`
	EndingNarrative  string             `json:"ending_narrative,omitempty"`
}

// EventQueueState 当前事件队列与完整游戏快照。
type EventQueueState struct {
	Silo                Silo                 `json:"silo"`
	Agent               Agent                `json:"agent"`
	AgentStats          AgentStats           `json:"agent_stats"`
	AvailableActions    []PlayerActionMeta   `json:"available_actions,omitempty"`
	World               *WorldView           `json:"world,omitempty"`
	GameOver            bool                 `json:"game_over"`
	EndingNarrative     string               `json:"ending_narrative,omitempty"`
	PendingEvents       int                  `json:"pending_events"`
	PendingOperation    string               `json:"pending_operation,omitempty"`
	InformationProgress *InformationProgress `json:"information_progress,omitempty"`
	LinesProgress       *LinesProgress       `json:"lines_progress,omitempty"`
	WitnessTree         *WitnessTree         `json:"witness_tree,omitempty"`
}

// EventStepResult 单步处理一个事件后的完整结果。
type EventStepResult struct {
	Silo               Silo               `json:"silo"`
	Agent              Agent              `json:"agent"`
	AgentStats         AgentStats         `json:"agent_stats"`
	AvailableActions   []PlayerActionMeta `json:"available_actions,omitempty"`
	World              *WorldView         `json:"world,omitempty"`
	GameOver           bool               `json:"game_over"`
	EndingNarrative    string             `json:"ending_narrative,omitempty"`
	PendingEvents      int                `json:"pending_events"`
	PendingOperation   string             `json:"pending_operation,omitempty"`
	ProcessedEventID   string             `json:"processed_event_id,omitempty"`
	ProcessedEventType string             `json:"processed_event_type,omitempty"`
	OperationComplete  bool               `json:"operation_complete"`
	Logs               []string           `json:"logs,omitempty"`
	Stories            []StoryEvent       `json:"stories,omitempty"`
	ActionResult       *ActionResult      `json:"action_result,omitempty"`

	InformationProgress *InformationProgress `json:"information_progress,omitempty"`
	LinesProgress       *LinesProgress       `json:"lines_progress,omitempty"`
	WitnessTree         *WitnessTree         `json:"witness_tree,omitempty"`
}

// EventHistoryResult 事件历史查询结果
type EventHistoryResult struct {
	Events []StoryEventLog `json:"events"`
}

// ProfessionActionMeta 职业专属行动元数据 (前端渲染按钮用，不含执行逻辑)
type ProfessionActionMeta struct {
	ID               string  `json:"id"`
	Profession       string  `json:"profession"`
	Label            string  `json:"label"`
	Description      string  `json:"description"`
	APCost           int     `json:"ap_cost"`
	TargetType       string  `json:"target_type"` // NONE / DEPT / RESOURCE
	SuspicionPenalty float64 `json:"suspicion_penalty"`
}

// ActionVisibility is the server-authoritative progression state of an action.
type ActionVisibility string

const (
	ActionLocked    ActionVisibility = "locked"
	ActionUnlocked  ActionVisibility = "unlocked"
	ActionAvailable ActionVisibility = "available"
)

// PlayerActionMeta 前端渲染玩家动作列表的统一元数据。
type PlayerActionMeta struct {
	ID                  string           `json:"id"`
	Source              string           `json:"source"`
	EventID             string           `json:"event_id,omitempty"`
	Label               string           `json:"label"`
	Description         string           `json:"description"`
	Group               string           `json:"group"`
	ActionType          AgentActionType  `json:"action_type"`
	TargetType          string           `json:"target_type"`
	APCost              int              `json:"ap_cost"`
	DurationMonths      int              `json:"duration_months"`
	Enabled             bool             `json:"enabled"`
	Visibility          ActionVisibility `json:"visibility"`
	DisabledReason      string           `json:"disabled_reason,omitempty"`
	UnavailableBehavior string           `json:"unavailable_behavior,omitempty"`
}

// ALL_FRAGMENTS 全部信息碎片
var ALL_FRAGMENTS = []string{
	"Mayor_1", "Mayor_2", "Mayor_3", "Mayor_4", "Mayor_5",
	"Judicial_1", "Judicial_2", "Judicial_3", "Judicial_4", "Judicial_5",
	"IT_1", "IT_2", "IT_3", "IT_4", "IT_5",
	"Police_1", "Police_2",
	"Medical_1", "Medical_2",
	"Mechanical_1", "Mechanical_2",
	"Supply_1", "Supply_2",
	"Mines_1", "Mines_2",
	"Agricultural_1",
}

// ACTION_COSTS 通用动作 AP 成本。
// 流通三动词 (收集/分享/建立关系) 的成本按 17 号经济文档 §3 的推导下调：
// AP 月收入约 3 点时，分享每 ~1 月一次、收集/建联穿插其间，流通才可能
// 跑赢恒定淡忘。戏剧性动作 (煽动/宣传/公开化) 保持高成本不变。
// TODO(balance): 全部为 sim 起点。
var ACTION_COSTS = map[AgentActionType]float64{
	ActionGatherInfo: 2, // TODO(balance): 10 → 2
	// 无声探查基准 = 普通探查 ×2 (免怀疑的定价，19 §4.2，TODO(balance))；
	// 服务端实际成本在 resolveActionCost 里按 2×(探查成本含规范冲突上浮) 结算。
	ActionSilentProbe:       4,
	ActionShareInfo:         3, // TODO(balance): 20 → 3
	ActionBuildConnection:   4, // TODO(balance): 15 → 4
	ActionInciteRebellion:   30,
	ActionConductPropaganda: 20,
	ActionPublicizeFaction:  25,
	ActionProfession:        0, // 实际成本由职业行动注册表决定
	ActionPlayerEvent:       0, // 实际成本由 content event 元数据决定
	ActionSilo1Operation:    0, // Actual cost comes from the operation definition.
	ActionStationOperation:  0, // Actual cost comes from the operation definition.
}

// ACTION_DURATIONS 通用动作耗时 (月)
var ACTION_DURATIONS = map[AgentActionType]int{
	ActionGatherInfo:        0,
	ActionSilentProbe:       0,
	ActionShareInfo:         0,
	ActionBuildConnection:   1,
	ActionInciteRebellion:   2,
	ActionConductPropaganda: 1,
	ActionPublicizeFaction:  0, // 立即生效
	ActionProfession:        1,
	ActionPlayerEvent:       0,
	ActionSilo1Operation:    0,
	ActionStationOperation:  0,
}

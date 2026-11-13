package model

import (
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
const CurrentSaveVersion = 8

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
	ID                uint           `gorm:"primarykey" json:"id"`
	UserID            uint           `gorm:"index" json:"user_id"`
	SiloID            uint           `gorm:"index" json:"silo_id"`                   // Current simulated silo; the player belongs to the assigned station.
	StationSiloID     uint           `gorm:"index" json:"station_silo_id,omitempty"` // Player's assigned station; zero marks an incompatible pre-v6 save.
	Name              string         `gorm:"size:64" json:"name"`
	Profession        string         `gorm:"size:50" json:"profession"`               // 职业 (如: Mayor, IT, Mechanical)
	Traits            []string       `gorm:"type:text;serializer:json" json:"traits"` // 特质 (如: "地堡土著", "一号地堡密使")
	PoliticalPrestige float64        `gorm:"default:0.0" json:"political_prestige"`   // 政治威望
	PoliticalPoints   float64        `gorm:"default:0.0" json:"political_points"`     // 政治点数
	ActionPoints      float64        `gorm:"default:0.0" json:"action_points"`        // 行动点数 (AP)
	PropagandaLevel   float64        `gorm:"default:0.0" json:"propaganda_level"`     // 宣传力度
	SuspicionLevel    float64        `gorm:"default:0.0" json:"suspicion_level"`      // 怀疑度指数
	Connections       []Connection   `gorm:"foreignKey:AgentID" json:"connections"`
	KnownFragments    []string       `gorm:"type:text;serializer:json" json:"known_fragments"` // 特工个人掌握的信息碎片
	Relics            []Relic        `gorm:"foreignKey:AgentID" json:"relics,omitempty"`       // 特工私吞的遗物
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
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
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
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

// JudicialAttachment stores the three-way allegiance and its hysteresis state.
// Pact fields are structural qualifications only; no current content grants them.
type JudicialAttachment struct {
	IT                     float64 `json:"a_it"`
	Mayor                  float64 `json:"a_mayor"`
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
	PactLocked             bool    `json:"pact_locked"`
}

// ShadowAuthority is the independent IT authority state. OrderDeviation is a
// reserved input interface and is deliberately absent from the v8 W formula.
type ShadowAuthority struct {
	Initialized          bool               `json:"initialized"`
	InitializationSource string             `json:"initialization_source"`
	W                    float64            `json:"w"`
	K                    float64            `json:"k"`
	E                    float64            `json:"e"`
	D                    float64            `json:"d"`
	P                    float64            `json:"p"`
	X                    float64            `json:"x"`
	OrderDeviation       float64            `json:"order_deviation"`
	PoliceObedience      float64            `json:"police_obedience"`
	Judicial             JudicialAttachment `json:"judicial"`
}

// Silo 地堡核心模型
type Silo struct {
	ID                 uint               `gorm:"primarykey" json:"id"`
	WorldID            uint               `gorm:"index;default:1" json:"world_id"` // 所属世界 (世界模型 v2 起)
	Name               string             `gorm:"size:100" json:"name"`
	Traits             []string           `gorm:"type:text;serializer:json" json:"traits"` // 地堡特质 (abundant/leak/psychoactive_meds)
	SafeguardRisk      float64            `gorm:"default:0.0" json:"safeguard_risk"`       // IT 专属风险系数
	TotalPopulation    int                `gorm:"default:10000" json:"total_population"`
	Legitimacy         float64            `gorm:"default:1.0" json:"legitimacy"`                             // Compatibility mirror of Mayor.Legitimacy.
	Mayor              MayorOffice        `gorm:"type:text;serializer:json" json:"mayor"`                    // 现任 Mayor 个人正统性
	Authority          ShadowAuthority    `gorm:"type:text;serializer:json" json:"authority"`                // IT 幕后威势与 Judicial 附着
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
}

// VictoryStatus 胜利状态
type VictoryStatus struct {
	IsWon       bool            `json:"is_won"`
	Type        string          `json:"type"` // NONE/INFORMATION/TIME/REBELLION/EXCLUSIONIST/DEATH/AGENT_COMPROMISED
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
	ID             uint               `gorm:"primarykey" json:"id"`
	SiloID         uint               `gorm:"index" json:"silo_id"`
	Name           string             `gorm:"size:50" json:"name"`
	ClassType      string             `gorm:"size:20" json:"class_type"` // ELITE / COMMONER
	Population     int                `gorm:"default:0" json:"population"`
	Ideologies     map[string]float64 `gorm:"type:text;serializer:json" json:"ideologies"` // 各项思潮值 (0-1)
	PanicValue     float64            `gorm:"default:0.0" json:"panic_value"`              // 恐慌值
	Productivity   float64            `gorm:"default:1.0" json:"productivity"`             // 生产力值
	PowerLevel     int                `gorm:"default:1" json:"power_level"`
	Zone           string             `gorm:"size:20" json:"zone"`
	KnownFragments []string           `gorm:"type:text;serializer:json" json:"known_fragments"`     // 掌握的其他部门信息碎片来源
	Relations      map[string]float64 `gorm:"type:text;serializer:json" json:"relations,omitempty"` // NPC部门之间的人脉/关系网
	// ---- 统一 Actor 经济体系 (与 Agent 同构) ----
	ActionPoints      float64   `gorm:"default:0" json:"action_points"`                    // 行动点数 (AP)
	SuspicionLevel    float64   `gorm:"default:0" json:"suspicion_level"`                  // 怀疑度
	PoliticalPrestige float64   `gorm:"default:0" json:"political_prestige"`               // 政治威望
	PropagandaLevel   float64   `gorm:"default:0" json:"propaganda_level"`                 // 宣传力度
	Traits            []string  `gorm:"type:text;serializer:json" json:"traits,omitempty"` // 特质
	Relics            []Relic   `gorm:"foreignKey:ProfessionID" json:"relics,omitempty"`   // 部门保管的遗物 (如司法部)
	UpdatedAt         time.Time `json:"updated_at"`
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
	ID                uint               `gorm:"primarykey" json:"id"`
	SiloID            uint               `gorm:"index" json:"silo_id"`
	ProfessionID      uint               `gorm:"index" json:"profession_id"`
	FactionID         *uint              `gorm:"index" json:"faction_id,omitempty"`
	Name              string             `gorm:"size:100" json:"name"`
	Count             int                `gorm:"default:0" json:"count"`
	IdeologyProfile   []string           `gorm:"type:text;serializer:json" json:"ideology_profile"` // 核心意识形态组合 (1-2个)
	HomeZone          string             `gorm:"size:20" json:"home_zone"`
	Loyalty           float64            `gorm:"default:0.5" json:"loyalty"`
	Influence         float64            `gorm:"default:0.0" json:"influence"`
	ActionPoints      float64            `gorm:"default:0.0" json:"action_points"`
	PoliticalPrestige float64            `gorm:"default:0.0" json:"political_prestige"`
	PanicSensitivity  float64            `gorm:"default:1.0" json:"panic_sensitivity"`
	Ideologies        map[string]float64 `gorm:"type:text;serializer:json" json:"ideologies"`
	KnownFragments    []string           `gorm:"type:text;serializer:json" json:"known_fragments,omitempty"`
	Tags              []string           `gorm:"type:text;serializer:json" json:"tags,omitempty"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

// Resident 显式居民/单位模型：每个 NPC unit 都有自己的 tags 与状态
type Resident struct {
	ID                uint               `gorm:"primarykey" json:"id"`
	SiloID            uint               `gorm:"index" json:"silo_id"`
	Name              string             `gorm:"size:80" json:"name"`
	CohortID          *uint              `gorm:"index" json:"cohort_id,omitempty"`
	ProfessionID      uint               `gorm:"index" json:"profession_id"`
	Profession        string             `gorm:"size:50" json:"profession"`
	FactionID         *uint              `gorm:"index" json:"faction_id,omitempty"`
	HomeFloor         int                `json:"home_floor"`
	Tags              []string           `gorm:"type:text;serializer:json" json:"tags"`
	Loyalty           float64            `gorm:"default:0.5" json:"loyalty"`
	Ambition          float64            `gorm:"default:0.0" json:"ambition"`
	Ideologies        map[string]float64 `gorm:"type:text;serializer:json" json:"ideologies"`
	Influence         float64            `gorm:"default:0.0" json:"influence"`
	ActionPoints      float64            `gorm:"default:0.0" json:"action_points"`
	SuspicionLevel    float64            `gorm:"default:0.0" json:"suspicion_level"`
	PoliticalPrestige float64            `gorm:"default:0.0" json:"political_prestige"`
	PropagandaLevel   float64            `gorm:"default:0.0" json:"propaganda_level"`
	KnownFragments    []string           `gorm:"type:text;serializer:json" json:"known_fragments"`
	Relations         map[string]float64 `gorm:"type:text;serializer:json" json:"relations,omitempty"`
	IsRepresentative  bool               `gorm:"default:false" json:"is_representative"`
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
	ID               uint      `gorm:"primarykey" json:"id"`
	Key              string    `gorm:"uniqueIndex;size:180" json:"key"`
	SourceGroup      string    `gorm:"index;size:32" json:"source_group"` // histories / special / crisis / player_actions / legacy events
	SourceFile       string    `gorm:"size:255" json:"source_file"`
	SourceFormat     string    `gorm:"size:24" json:"source_format"`
	EventID          string    `gorm:"index;size:100" json:"event_id"`
	Title            string    `gorm:"size:120" json:"title"`
	Description      string    `gorm:"type:text" json:"description"`
	Type             string    `gorm:"size:32" json:"type"`
	FireMode         string    `gorm:"size:32" json:"fire_mode"` // ONCE / REPEATABLE
	CooldownMonths   int       `gorm:"default:0" json:"cooldown_months"`
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
	ActionGatherInfo        AgentActionType = "GATHER_INFO"
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
}

// CreateGameRequest 新建游戏请求
type CreateGameRequest struct {
	SiloName     string              `json:"silo_name"`
	StartYear    int                 `json:"start_year"`
	TraitIds     []string            `json:"trait_ids"`
	AgentName    string              `json:"agent_name"`
	Profession   string              `json:"profession"`
	Seed         int64               `json:"seed,omitempty"`          // 可选：显式指定本局 RNG 种子 (0 = 按 SILO40_SEED/随机决议)
	Pacing       *WorldPacing        `json:"pacing,omitempty"`        // 可选：节奏定制 (缺省 = 全 1.0，行为与历史逐位一致)
	Neighbor     *NeighborSiloConfig `json:"neighbor,omitempty"`      // Optional Silo 18; nil preserves the legacy path.
	IncludeSilo1 bool                `json:"include_silo1,omitempty"` // Opt in to a fully simulated peer Silo 1.
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
	ID                       uint             `json:"id"`
	Seed                     int64            `json:"seed"`
	CurrentYear              int              `json:"current_year"`
	CurrentMonth             int              `json:"current_month"`
	PlayerSiloID             uint             `json:"player_silo_id"`
	ProgressStage            string           `json:"progress_stage"`
	Pacing                   WorldPacing      `json:"pacing"`
	Silo1Alive               bool             `json:"silo1_alive"`
	Silo1Simulated           bool             `json:"silo1_simulated"`
	Silo1Departments         []string         `json:"silo1_departments,omitempty"`
	Silo1LastReport          string           `json:"silo1_last_report,omitempty"`
	Silo1Exposure            float64          `json:"silo1_exposure"`
	Silo1CounterintelStage   int              `json:"silo1_counterintel_stage"`
	Silo1Operations          []Silo1Operation `json:"silo1_operations,omitempty"`
	StationSiloID            uint             `json:"station_silo_id,omitempty"`
	StationSilo              *Silo            `json:"station_silo,omitempty"`
	StationLastReport        string           `json:"station_last_report,omitempty"`
	StationExposure          float64          `json:"station_exposure"`
	StationCounterintelStage int              `json:"station_counterintel_stage"`
	StationOperations        []Silo1Operation `json:"station_operations,omitempty"`
	Silos                    []WorldSiloRef   `json:"silos"`
	Neighbors                []RumorView      `json:"neighbors,omitempty"` // 非玩家堡的迷雾视图 (P0.5)
}

// GameState 后端权威游戏状态快照 (返回给前端)
type GameState struct {
	Silo              Silo                   `json:"silo"`
	Agent             Agent                  `json:"agent"`
	AgentStats        AgentStats             `json:"agent_stats"`
	GameOver          bool                   `json:"game_over"`
	EndingNarrative   string                 `json:"ending_narrative,omitempty"`
	VictoryStatus     *VictoryStatus         `json:"victory_status,omitempty"`
	ProfessionActions []ProfessionActionMeta `json:"profession_actions,omitempty"`
	AvailableActions  []PlayerActionMeta     `json:"available_actions,omitempty"`
	World             *WorldView             `json:"world,omitempty"`
}

// AgentStats 后端下发的 Agent 派生指标，用于前端展示，避免 UI 复算引擎公式
type AgentStats struct {
	AvgConnection        float64 `json:"avg_connection"`
	PrestigeBase         float64 `json:"prestige_base"`
	ProfessionFactor     float64 `json:"profession_factor"`
	TraitFactor          float64 `json:"trait_factor"`
	StructuralPrestige   float64 `json:"structural_prestige"`
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
	Silo             Silo               `json:"silo"`
	Agent            Agent              `json:"agent"`
	AgentStats       AgentStats         `json:"agent_stats"`
	AvailableActions []PlayerActionMeta `json:"available_actions,omitempty"`
	World            *WorldView         `json:"world,omitempty"`
	GameOver         bool               `json:"game_over"`
	EndingNarrative  string             `json:"ending_narrative,omitempty"`
	PendingEvents    int                `json:"pending_events"`
	PendingOperation string             `json:"pending_operation,omitempty"`
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

// ACTION_COSTS 通用动作 AP 成本
var ACTION_COSTS = map[AgentActionType]float64{
	ActionGatherInfo:        10,
	ActionShareInfo:         20,
	ActionBuildConnection:   15,
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

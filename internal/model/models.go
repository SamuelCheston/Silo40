package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	IdeologyProForeign = "pro_foreign"
	IdeologyLoyalty    = "loyalty"
)

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

// Silo 地堡核心模型
type Silo struct {
	ID                 uint               `gorm:"primarykey" json:"id"`
	Name               string             `gorm:"size:100" json:"name"`
	Traits             []string           `gorm:"type:text;serializer:json" json:"traits"` // 地堡特质 (abundant/leak/psychoactive_meds)
	SafeguardRisk      float64            `gorm:"default:0.0" json:"safeguard_risk"`       // IT 专属风险系数
	TotalPopulation    int                `gorm:"default:10000" json:"total_population"`
	Legitimacy         float64            `gorm:"default:1.0" json:"legitimacy"`                             // 合法性值
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

// VictoryStatus 胜利状态
type VictoryStatus struct {
	IsWon       bool       `json:"is_won"`
	Type        string     `json:"type"` // NONE/INFORMATION/TIME/REBELLION/EXCLUSIONIST/DEATH/AGENT_COMPROMISED
	Description string     `json:"description"`
	Score       *GameScore `json:"score,omitempty"`
}

// GameScore 评分结算
type GameScore struct {
	Total           int     `json:"total"`
	SurvivalPoints  int     `json:"survival_points"`
	DiversityPoints int     `json:"diversity_points"`
	HeritagePoints  int     `json:"heritage_points"`
	IdeologyPoints  int     `json:"ideology_points"`
	Multiplier      float64 `json:"multiplier"`
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
	Alive             bool               `gorm:"default:true" json:"alive"`
	UpdatedAt         time.Time          `json:"updated_at"`
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
	UpdatedAt                time.Time      `json:"updated_at"`
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
	Title       string    `gorm:"size:120" json:"title"`
	Description string    `gorm:"type:text" json:"description"`
	Type        string    `gorm:"size:32" json:"type"` // SOCIAL / TECHNICAL / EXTERNAL
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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
)

// AgentAction 特工/部门动作请求
type AgentAction struct {
	Type             AgentActionType `json:"type"`
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
	Title       string `json:"title"`
	Description string `json:"description"`
	Type        string `json:"type"` // SOCIAL / TECHNICAL / EXTERNAL
}

// CreateGameRequest 新建游戏请求
type CreateGameRequest struct {
	SiloName   string   `json:"silo_name"`
	StartYear  int      `json:"start_year"`
	TraitIds   []string `json:"trait_ids"`
	AgentName  string   `json:"agent_name"`
	Profession string   `json:"profession"`
}

// GameState 后端权威游戏状态快照 (返回给前端)
type GameState struct {
	Silo              Silo                   `json:"silo"`
	Agent             Agent                  `json:"agent"`
	GameOver          bool                   `json:"game_over"`
	EndingNarrative   string                 `json:"ending_narrative,omitempty"`
	VictoryStatus     *VictoryStatus         `json:"victory_status,omitempty"`
	ProfessionActions []ProfessionActionMeta `json:"profession_actions,omitempty"`
}

// TickResult 时间推进结果
type TickResult struct {
	Silo            Silo         `json:"silo"`
	Agent           Agent        `json:"agent"`
	Logs            []string     `json:"logs"`
	Stories         []StoryEvent `json:"stories"`
	GameOver        bool         `json:"game_over"`
	EndingNarrative string       `json:"ending_narrative,omitempty"`
}

// ActionOutcome 动作执行结果 (含最新状态)
type ActionOutcome struct {
	Silo            Silo         `json:"silo"`
	Agent           Agent        `json:"agent"`
	Result          ActionResult `json:"result"`
	Logs            []string     `json:"logs"`
	Stories         []StoryEvent `json:"stories"`
	GameOver        bool         `json:"game_over"`
	EndingNarrative string       `json:"ending_narrative,omitempty"`
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
	ActionProfession:        0, // 实际成本由职业行动注册表决定
}

// ACTION_DURATIONS 通用动作耗时 (月)
var ACTION_DURATIONS = map[AgentActionType]int{
	ActionGatherInfo:        0,
	ActionShareInfo:         0,
	ActionBuildConnection:   1,
	ActionInciteRebellion:   2,
	ActionConductPropaganda: 1,
	ActionProfession:        1,
}

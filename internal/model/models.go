package model

import (
	"time"

	"gorm.io/gorm"
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

// Agent 特工模型
type Agent struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	UserID            uint           `gorm:"index" json:"user_id"`
	Name              string         `gorm:"size:64" json:"name"`
	Profession        string         `gorm:"size:50" json:"profession"`               // 职业 (如: IT, Sheriff, Mechanical)
	Traits            []string       `gorm:"type:text;serializer:json" json:"traits"` // 特质 (如: "地堡土著", "一号地堡密使")
	PoliticalPrestige float64        `gorm:"default:0.0" json:"political_prestige"`   // 政治威望
	PoliticalPoints   float64        `gorm:"default:0.0" json:"political_points"`     // 政治点数
	PropagandaLevel   float64        `gorm:"default:0.0" json:"propaganda_level"`     // 宣传力度
	Connections       []Connection   `gorm:"foreignKey:AgentID" json:"connections"`
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
	ID              uint           `gorm:"primarykey" json:"id"`
	Name            string         `gorm:"size:100" json:"name"`
	TotalPopulation int            `gorm:"default:10000" json:"total_population"`
	Legitimacy      float64        `gorm:"default:1.0" json:"legitimacy"`     // 合法性值
	Cohesion        float64        `gorm:"default:1.0" json:"cohesion"`       // 凝聚力值
	Rebellion       float64        `gorm:"default:0.0" json:"rebellion"`      // 地堡叛乱值
	HistoryBurden   float64        `gorm:"default:0.0" json:"history_burden"` // 历史包袱值 (罪恶/荣誉)
	EventTrigger    float64        `gorm:"default:0.0" json:"event_trigger"`  // 外部事件触发值
	CurrentYear     int            `gorm:"default:122" json:"current_year"`   // 当前年份
	CountdownYears  float64        `gorm:"default:500.0" json:"countdown"`    // 500年倒计时 (时间胜利)
	InfoFragments   int            `gorm:"default:0" json:"info_fragments"`   // 信息碎片 (信息胜利)
	Resources       []Resource     `gorm:"foreignKey:SiloID" json:"resources"`
	Professions     []Profession   `gorm:"foreignKey:SiloID" json:"professions"`
	Floors          []Floor        `gorm:"foreignKey:SiloID" json:"floors"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// Resource 资源模型
type Resource struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	SiloID     uint      `gorm:"index" json:"silo_id"`
	Type       string    `gorm:"size:20" json:"type"` // Food, Energy, Medical, etc.
	Amount     float64   `gorm:"default:0" json:"amount"`
	NetBalance float64   `gorm:"default:0" json:"net_balance"` // 资源结余 (万人单位)
	UpdatedAt  time.Time `json:"updated_at"`
}

// Profession 职业/部门模型
type Profession struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	SiloID        uint      `gorm:"index" json:"silo_id"`
	Name          string    `gorm:"size:50" json:"name"`
	Population    int       `gorm:"default:0" json:"population"`
	IdeologyValue float64   `gorm:"default:0.5" json:"ideology_value"` // 亲外/排外思潮值 (0-1)
	PanicValue    float64   `gorm:"default:0.0" json:"panic_value"`    // 恐慌值
	Productivity  float64   `gorm:"default:1.0" json:"productivity"`   // 生产力值 (各部门劳动力产出)
	PowerLevel    int       `gorm:"default:1" json:"power_level"`
	Zone          string    `gorm:"size:20" json:"zone"`
	UpdatedAt     time.Time `json:"updated_at"`
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

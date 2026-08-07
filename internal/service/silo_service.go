package service

import (
	"silo40/internal/model"

	"gorm.io/gorm"
)

type SiloService struct {
	db *gorm.DB
}

func NewSiloService(db *gorm.DB) *SiloService {
	return &SiloService{db: db}
}

// InitSilo 根据原著设定初始化一个 144 层地堡
func (s *SiloService) InitSilo(name string) (*model.Silo, error) {
	silo := &model.Silo{
		Name:            name,
		TotalPopulation: 10000,
		Legitimacy:      1.0,
		Cohesion:        1.0,
		Rebellion:       0.0,
		HistoryBurden:   0.0,
		EventTrigger:    0.0,
		CountdownYears:  500.0,
		InfoFragments:   0,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(silo).Error; err != nil {
			return err
		}

		// 1. 初始化 144 层楼层
		if err := s.initFloors(tx, silo.ID); err != nil {
			return err
		}

		// 2. 初始化核心部门
		if err := s.initProfessions(tx, silo.ID); err != nil {
			return err
		}

		// 3. 初始化基础资源
		if err := s.initResources(tx, silo.ID); err != nil {
			return err
		}

		return nil
	})

	return silo, err
}

// InitAgent 为用户初始化一个特工
func (s *SiloService) InitAgent(userID uint, name string, profession string, traits []string) (*model.Agent, error) {
	agent := &model.Agent{
		UserID:     userID,
		Name:       name,
		Profession: profession,
		Traits:     traits,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(agent).Error; err != nil {
			return err
		}

		// 获取该地堡的所有部门，初始化人脉值
		var siloID uint
		if err := tx.Model(&model.User{}).Where("id = ?", userID).Select("silo_id").Scan(&siloID).Error; err != nil {
			return err
		}

		var professions []model.Profession
		if err := tx.Where("silo_id = ?", siloID).Find(&professions).Error; err != nil {
			return err
		}

		for _, p := range professions {
			// 默认初始人脉值，如果职业匹配则更高
			val := 0.1
			if p.Name == profession {
				val = 0.5
			}
			conn := model.Connection{
				AgentID:      agent.ID,
				ProfessionID: p.ID,
				Value:        val,
			}
			if err := tx.Create(&conn).Error; err != nil {
				return err
			}
		}

		return nil
	})

	return agent, err
}

func (s *SiloService) initFloors(tx *gorm.DB, siloID uint) error {
	floorConfigs := []struct {
		start, end int
		function   string
		zone       string
	}{
		{1, 10, "Administrative", "Upper"},
		{11, 20, "Public Facilities", "Upper"},
		{21, 30, "Residential A", "Upper"},
		{31, 60, "Residential B", "Mid"},
		{61, 75, "Hydroponics", "Mid"},
		{76, 80, "Medical", "Mid"},
		{81, 90, "Cafeteria & Supply", "Mid"},
		{91, 120, "Residential C", "Lower"},
		{121, 135, "Mechanical", "Lower"},
		{136, 140, "Maintenance", "Lower"},
		{141, 144, "Mines", "Lower"},
	}

	for _, config := range floorConfigs {
		for i := config.start; i <= config.end; i++ {
			floor := model.Floor{
				SiloID:    siloID,
				Level:     i,
				Function:  config.function,
				Zone:      config.zone,
				Stability: 1.0,
			}
			if err := tx.Create(&floor).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SiloService) initProfessions(tx *gorm.DB, siloID uint) error {
	professions := []model.Profession{
		{SiloID: siloID, Name: "Mayor", PowerLevel: 10, Zone: "Upper", Population: 200, IdeologyValue: 0.5, PanicValue: 0.0, Productivity: 1.0},
		{SiloID: siloID, Name: "Judicial", PowerLevel: 9, Zone: "Upper", Population: 400, IdeologyValue: 0.5, PanicValue: 0.0, Productivity: 1.0},
		{SiloID: siloID, Name: "IT", PowerLevel: 9, Zone: "Upper", Population: 600, IdeologyValue: 0.5, PanicValue: 0.0, Productivity: 1.0},
		{SiloID: siloID, Name: "Sheriff", PowerLevel: 8, Zone: "Upper", Population: 300, IdeologyValue: 0.5, PanicValue: 0.0, Productivity: 1.0},
		{SiloID: siloID, Name: "Medical", PowerLevel: 7, Zone: "Mid", Population: 800, IdeologyValue: 0.5, PanicValue: 0.0, Productivity: 1.0},
		{SiloID: siloID, Name: "Supply", PowerLevel: 6, Zone: "Mid", Population: 1200, IdeologyValue: 0.5, PanicValue: 0.0, Productivity: 1.0},
		{SiloID: siloID, Name: "Mechanical", PowerLevel: 8, Zone: "Lower", Population: 1500, IdeologyValue: 0.5, PanicValue: 0.0, Productivity: 1.0},
		{SiloID: siloID, Name: "Maintenance", PowerLevel: 5, Zone: "Lower", Population: 1000, IdeologyValue: 0.5, PanicValue: 0.0, Productivity: 1.0},
		{SiloID: siloID, Name: "Mines", PowerLevel: 4, Zone: "Lower", Population: 2000, IdeologyValue: 0.5, PanicValue: 0.0, Productivity: 1.0},
		{SiloID: siloID, Name: "Agricultural", PowerLevel: 6, Zone: "Mid", Population: 2000, IdeologyValue: 0.5, PanicValue: 0.0, Productivity: 1.0},
	}

	for _, p := range professions {
		if err := tx.Create(&p).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *SiloService) initResources(tx *gorm.DB, siloID uint) error {
	resources := []model.Resource{
		{SiloID: siloID, Type: "Food", Amount: 1000, NetBalance: 0},
		{SiloID: siloID, Type: "Energy", Amount: 5000, NetBalance: 0},
		{SiloID: siloID, Type: "Water", Amount: 2000, NetBalance: 0},
		{SiloID: siloID, Type: "Materials", Amount: 500, NetBalance: 0},
	}

	for _, r := range resources {
		if err := tx.Create(&r).Error; err != nil {
			return err
		}
	}
	return nil
}

package engine

import (
	"math/rand"

	"silo40/internal/model"
)

// ============ 地堡/特工初始化 (新建游戏) ============

func CreateInitialSilo(name string, initialYear int, traitIds []string) *model.Silo {
	silo := &model.Silo{
		ID:              0,
		Name:            name,
		Traits:          traitIds,
		SafeguardRisk:   0.0,
		TotalPopulation: 10000,
		Legitimacy:      1.0,
		Cohesion:        1.0,
		Rebellion:       0.0,
		HistoryBurden:   0.0,
		EventTrigger:    0.0,
		CurrentYear:     initialYear,
		CurrentMonth:    1,
		Countdown:       500.0,
		Silo1Destroyed:  false,
		Resources:       []model.Resource{},
		Professions:     []model.Profession{},
		Floors:          []model.Floor{},
		Cohorts:         []model.PopulationCohort{},
		Residents:       []model.Resident{},
		Factions:        []model.Faction{},
	}

	// 1. Initialize 144 Floors
	silo.Floors = initFloors()

	// 2. Initialize Core Professions
	silo.Professions = initProfessions()

	// Initialize NPC-NPC relations and traits based on specific logic
	for i := range silo.Professions {
		prof := &silo.Professions[i]
		if prof.Relations == nil {
			prof.Relations = map[string]float64{}
		}

		// IT: Knows all fragments initially
		if prof.Name == "IT" {
			prof.KnownFragments = append([]string{}, model.ALL_FRAGMENTS...)
		}

		for j := range silo.Professions {
			otherProf := &silo.Professions[j]
			if prof.ID == otherProf.ID {
				continue
			}
			baseRelation := 0.0

			if prof.Name == "Mayor" || prof.Name == "Judicial" {
				if otherProf.ClassType == "ELITE" {
					baseRelation = 0.20
				} else {
					baseRelation = 0.15
				}
			} else if prof.Name == "Police" || prof.Name == "Medical" {
				if otherProf.ClassType == "ELITE" {
					baseRelation = 0.15
				} else {
					baseRelation = 0.10
				}
			} else if prof.ClassType == "COMMONER" {
				if otherProf.ClassType == "COMMONER" {
					baseRelation = 0.15
				}
			}

			// Specific overrides
			if prof.Name == "Mayor" && otherProf.Name == "Police" {
				baseRelation = 1.0
			}
			if prof.Name == "Police" && otherProf.Name == "Mayor" {
				baseRelation = 1.0
			}
			if prof.Name == "IT" && otherProf.Name == "Judicial" {
				baseRelation = 1.0
			}
			if prof.Name == "Judicial" && otherProf.Name == "IT" {
				baseRelation = 1.0
			}

			prof.Relations[otherProf.Name] = baseRelation
		}
	}

	// Initialize NPC Actor 经济体系 (与玩家特工同构)
	for i := range silo.Professions {
		prof := &silo.Professions[i]
		prof.ActionPoints = 50
		prof.SuspicionLevel = 0
		prof.PoliticalPrestige = 0
		prof.PropagandaLevel = 0
		prof.OrganizationFactor = 1.0
		if prof.Traits == nil {
			prof.Traits = []string{}
		}
	}

	// 3. Initialize Base Resources
	silo.Resources = initResources()

	// Apply Silo Traits
	for _, trait := range traitIds {
		switch trait {
		case "abundant":
			for i := range silo.Resources {
				r := &silo.Resources[i]
				if r.Type == "Supplies" || r.Type == "Energy" {
					r.Amount *= 2
				}
			}
		case "leak":
			for i := range silo.Professions {
				p := &silo.Professions[i]
				if p.ClassType == "COMMONER" {
					p.Ideologies[model.IdeologyProForeign] = min1(p.Ideologies[model.IdeologyProForeign] + 0.3)
				}
			}
		}
	}

	// 4. Initialize aggregated population model and key residents after world traits settle
	initResidentsAndFactions(silo)

	return silo
}

func initFloors() []model.Floor {
	var floors []model.Floor
	configs := []struct {
		start, end int
		fn, zone   string
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
	for _, cfg := range configs {
		for i := cfg.start; i <= cfg.end; i++ {
			floors = append(floors, model.Floor{
				ID:         0,
				SiloID:     0,
				Level:      i,
				Function:   cfg.fn,
				Zone:       cfg.zone,
				Stability:  1.0,
				Population: 0,
			})
		}
	}
	return floors
}

func initProfessions() []model.Profession {
	return []model.Profession{
		{ID: 1, SiloID: 0, Name: "Mayor", ClassType: "ELITE", PowerLevel: 10, Zone: "Upper", Population: 1, Ideologies: map[string]float64{model.IdeologyProForeign: 0.05, model.IdeologyDemocracy: 0.2}, PanicValue: 0.0, Productivity: 1.0, KnownFragments: []string{"Mayor_1", "Mayor_2", "Mayor_3", "Mayor_4", "Mayor_5"}, Relations: map[string]float64{}},
		{ID: 2, SiloID: 0, Name: "Judicial", ClassType: "ELITE", PowerLevel: 9, Zone: "Upper", Population: 200, Ideologies: map[string]float64{model.IdeologyProForeign: 0.05, model.IdeologyDemocracy: 0.1}, PanicValue: 0.0, Productivity: 1.0, KnownFragments: []string{"Judicial_1", "Judicial_2", "Judicial_3", "Judicial_4", "Judicial_5"}, Relations: map[string]float64{}},
		{ID: 3, SiloID: 0, Name: "IT", ClassType: "ELITE", PowerLevel: 9, Zone: "Upper", Population: 150, Ideologies: map[string]float64{model.IdeologyProForeign: 0.05, model.IdeologyDemocracy: 0.1}, PanicValue: 0.0, Productivity: 1.0, KnownFragments: []string{"IT_1", "IT_2", "IT_3", "IT_4", "IT_5"}, Relations: map[string]float64{}},
		{ID: 4, SiloID: 0, Name: "Police", ClassType: "ELITE", PowerLevel: 8, Zone: "Upper", Population: 250, Ideologies: map[string]float64{model.IdeologyProForeign: 0.05, model.IdeologyDemocracy: 0.05}, PanicValue: 0.0, Productivity: 1.0, KnownFragments: []string{"Police_1", "Police_2"}, Relations: map[string]float64{}},
		{ID: 5, SiloID: 0, Name: "Medical", ClassType: "ELITE", PowerLevel: 7, Zone: "Mid", Population: 400, Ideologies: map[string]float64{model.IdeologyProForeign: 0.05, model.IdeologyDemocracy: 0.1}, PanicValue: 0.0, Productivity: 1.0, KnownFragments: []string{"Medical_1", "Medical_2"}, Relations: map[string]float64{}},
		{ID: 6, SiloID: 0, Name: "Supply", ClassType: "COMMONER", PowerLevel: 6, Zone: "Mid", Population: 2500, Ideologies: map[string]float64{model.IdeologyProForeign: 0.05, model.IdeologyDemocracy: 0.1}, PanicValue: 0.0, Productivity: 1.0, KnownFragments: []string{"Supply_1", "Supply_2"}, Relations: map[string]float64{}},
		{ID: 7, SiloID: 0, Name: "Mechanical", ClassType: "COMMONER", PowerLevel: 8, Zone: "Lower", Population: 3000, Ideologies: map[string]float64{model.IdeologyProForeign: 0.05, model.IdeologyDemocracy: 0.1}, PanicValue: 0.0, Productivity: 1.0, KnownFragments: []string{"Mechanical_1", "Mechanical_2"}, Relations: map[string]float64{}},
		{ID: 8, SiloID: 0, Name: "Mines", ClassType: "COMMONER", PowerLevel: 4, Zone: "Lower", Population: 1000, Ideologies: map[string]float64{model.IdeologyProForeign: 0.05, model.IdeologyDemocracy: 0.1}, PanicValue: 0.0, Productivity: 1.0, KnownFragments: []string{"Mines_1", "Mines_2"}, Relations: map[string]float64{}},
		{ID: 9, SiloID: 0, Name: "Agricultural", ClassType: "COMMONER", PowerLevel: 6, Zone: "Mid", Population: 2499, Ideologies: map[string]float64{model.IdeologyProForeign: 0.05, model.IdeologyDemocracy: 0.1}, PanicValue: 0.0, Productivity: 1.0, KnownFragments: []string{"Agricultural_1"}, Relations: map[string]float64{}},
	}
}

func initResources() []model.Resource {
	return []model.Resource{
		{ID: 0, SiloID: 0, Type: "Energy", Amount: 5000, NetBalance: 0},
		{ID: 0, SiloID: 0, Type: "Materials", Amount: 500, NetBalance: 0},
		{ID: 0, SiloID: 0, Type: "Supplies", Amount: 3000, NetBalance: 0},
	}
}

func min1(v float64) float64 {
	if v > 1.0 {
		return 1.0
	}
	return v
}

// CreateInitialAgent 创建初始特工
func CreateInitialAgent(name, profession string, traitIds []string, silo *model.Silo) *model.Agent {
	agent := &model.Agent{
		ID:                 1,
		UserID:             1,
		Name:               name,
		Profession:         profession,
		Traits:             []string{},
		PoliticalPrestige:  10,
		PoliticalPoints:    0,
		ActionPoints:       50,
		OrganizationFactor: 1.0,
		PropagandaLevel:    0.0,
		SuspicionLevel:     0.0,
		Connections:        []model.Connection{},
	}
	// 默认掌握本部门的碎片
	for _, f := range model.ALL_FRAGMENTS {
		if len(f) > len(profession) && f[:len(profession)] == profession && f[len(profession)] == '_' {
			agent.KnownFragments = append(agent.KnownFragments, f)
		}
	}

	if silo != nil {
		switch profession {
		case "Mayor", "Judicial":
			if profession == "Mayor" {
				agent.PoliticalPrestige = 80
			} else {
				agent.PoliticalPrestige = 5
			}
			for i := range silo.Professions {
				p := &silo.Professions[i]
				randomVal := 0.0
				if p.ClassType == "ELITE" {
					randomVal = 0.2 + rand.Float64()*0.05
				} else {
					randomVal = 0.15 + rand.Float64()*0.05
				}
				agent.Connections = append(agent.Connections, model.Connection{
					AgentID: agent.ID, ProfessionID: p.ID, Value: randomVal,
				})
			}
		case "IT":
			agent.PoliticalPrestige = 5
			agent.KnownFragments = append([]string{}, model.ALL_FRAGMENTS...) // Knows all fragments
			for i := range silo.Professions {
				p := &silo.Professions[i]
				switch p.Name {
				case "Judicial":
					agent.Connections = append(agent.Connections, model.Connection{AgentID: agent.ID, ProfessionID: p.ID, Value: 1.0})
				case "IT":
					agent.Connections = append(agent.Connections, model.Connection{AgentID: agent.ID, ProfessionID: p.ID, Value: 0.4})
				}
			}
		case "Police":
			for i := range silo.Professions {
				p := &silo.Professions[i]
				connValue := 0.10
				if p.ClassType == "ELITE" {
					connValue = 0.15
				}
				if p.Name == "Mayor" || p.Name == "Police" {
					connValue = 1.0
				}
				agent.Connections = append(agent.Connections, model.Connection{AgentID: agent.ID, ProfessionID: p.ID, Value: connValue})
			}
		case "Medical":
			agent.PoliticalPrestige = 5
			for i := range silo.Professions {
				p := &silo.Professions[i]
				val := 0.10
				if p.ClassType == "ELITE" {
					val = 0.15
				}
				agent.Connections = append(agent.Connections, model.Connection{AgentID: agent.ID, ProfessionID: p.ID, Value: val})
			}
		default:
			isCommoner := profession == "Supply" || profession == "Mechanical" || profession == "Mines" || profession == "Agricultural"
			if profession == "Mines" {
				agent.PoliticalPrestige = 5
			}
			for i := range silo.Professions {
				p := &silo.Professions[i]
				if p.Name == profession {
					agent.Connections = append(agent.Connections, model.Connection{AgentID: agent.ID, ProfessionID: p.ID, Value: 0.4})
				} else if isCommoner && p.ClassType == "COMMONER" {
					agent.Connections = append(agent.Connections, model.Connection{AgentID: agent.ID, ProfessionID: p.ID, Value: 0.15})
				}
			}
		}
	}

	// Apply Agent Traits
	for _, trait := range traitIds {
		switch trait {
		case "native":
			agent.Traits = append(agent.Traits, "地堡土著")
			if silo != nil {
				agentProf := findDept(silo, agent.Profession)
				if agentProf != nil {
					for i := range silo.Professions {
						p := &silo.Professions[i]
						if p.ClassType == agentProf.ClassType && p.ID != agentProf.ID {
							agent.Connections = append(agent.Connections, model.Connection{
								AgentID: agent.ID, ProfessionID: p.ID, Value: 0.10 + rand.Float64()*0.05,
							})
						}
					}
				}
			}
		case "charismatic":
			agent.Traits = append(agent.Traits, "魅力非凡")
			agent.PoliticalPrestige += 20
		case "shadowy":
			agent.Traits = append(agent.Traits, "隐秘行事")
		}
	}

	return agent
}

package service

import (
	"silo40/internal/model"
)

type GameEngine struct {
	// 基础消耗基数 (万人/小时)
	BaseConsumption map[string]float64
	// 职业威望加成系数
	ProfessionFactors map[string]float64
	// 特质威望加成系数
	TraitFactors map[string]float64
}

func NewGameEngine() *GameEngine {
	return &GameEngine{
		BaseConsumption: map[string]float64{
			"Food":      100.0,
			"Energy":    500.0,
			"Water":     200.0,
			"Materials": 50.0,
		},
		ProfessionFactors: map[string]float64{
			"Mayor":      0.5,
			"Judicial":   0.4,
			"IT":         0.3,
			"Sheriff":    0.3,
			"Mechanical": 0.2,
			"Medical":    0.2,
		},
		TraitFactors: map[string]float64{
			"地堡土著":   0.1,
			"一号地堡密使": 0.5,
			"煽动者":    0.2,
			"守旧派":    -0.1,
		},
	}
}

// UpdateAgentState 更新特工数值逻辑 (事件驱动)
func (e *GameEngine) UpdateAgentState(agent *model.Agent, deltaYears float64) {
	if deltaYears <= 0 {
		return
	}

	// 1. 计算平均人脉值
	var totalConnection float64
	count := len(agent.Connections)
	if count > 0 {
		for _, conn := range agent.Connections {
			totalConnection += conn.Value
		}
		totalConnection /= float64(count)
	}

	// 2. 计算职业修正系数
	profFactor := e.ProfessionFactors[agent.Profession]

	// 3. 计算特质修正系数
	traitFactor := 0.0
	for _, trait := range agent.Traits {
		traitFactor += e.TraitFactors[trait]
	}

	// 4. 计算政治威望 (乘法修正模型)
	// 公式: 威望 = 平均人脉值 * 100 * (1 + 职业加成) * (1 + 特质总加成)
	agent.PoliticalPrestige = totalConnection * 100 * (1 + profFactor) * (1 + traitFactor)
	if agent.PoliticalPrestige < 0 {
		agent.PoliticalPrestige = 0
	}

	// 5. 给予政治点数 (按游戏年给予，每点威望每年贡献 0.1 点数)
	pointGainRate := 0.1
	agent.PoliticalPoints += agent.PoliticalPrestige * pointGainRate * deltaYears
}

// UpdateSiloState 核心逻辑更新引擎 (事件驱动)
func (e *GameEngine) UpdateSiloState(silo *model.Silo, deltaYears float64) {
	if deltaYears <= 0 {
		return
	}

	// 1. 更新部门生产力与资源结余
	e.updateResources(silo, deltaYears)

	// 2. 更新地堡状态 (倒计时、叛乱值、事件触发等)
	e.updateSiloMetrics(silo, deltaYears)

	// 3. 更新思潮演化
	e.updateIdeology(silo, deltaYears)
}

func (e *GameEngine) updateIdeology(silo *model.Silo, deltaYears float64) {
	// 基于凝聚力 (Cohesion) 的思潮稳定性
	// 凝聚力越低，思潮越容易受恐慌值影响产生偏移
	for i := range silo.Professions {
		p := &silo.Professions[i]
		stability := silo.Cohesion

		// 如果恐慌值高且凝聚力低，思潮向极端偏移 (这里假设向 0 或 1 随机偏移)
		if p.PanicValue > 0.3 && stability < 0.5 {
			// 简单的偏移模拟
			drift := (p.PanicValue * (1.0 - stability)) * deltaYears * 0.01
			p.IdeologyValue += drift // 实际应根据剧情逻辑决定方向
		}

		// 边界处理
		if p.IdeologyValue > 1.0 {
			p.IdeologyValue = 1.0
		} else if p.IdeologyValue < 0 {
			p.IdeologyValue = 0
		}
	}
}

func (e *GameEngine) updateResources(silo *model.Silo, deltaYears float64) {
	// 1. 统计恐慌部门数量 (PanicValue > 0.5)
	panicDeptCount := 0
	for _, p := range silo.Professions {
		if p.PanicValue > 0.5 {
			panicDeptCount++
		}
	}

	// 2. 判定叛乱状态 (Rebellion > 0.7)
	isRebelling := silo.Rebellion > 0.7

	for i := range silo.Resources {
		r := &silo.Resources[i]
		baseCons := e.BaseConsumption[r.Type]

		// 结余倍率逻辑：
		// 基础结余比例为 0.2 (消耗量的20%)
		// 每个恐慌部门扣减 0.05 (消耗量的5%)
		balanceMultiplier := 0.2 - (float64(panicDeptCount) * 0.05)

		if isRebelling {
			// 叛乱发生，开始消耗储存
			// 设定为消耗基数的 -0.5 倍 (即每小时净消耗 50% 的基数)
			balanceMultiplier = -0.5
		}

		r.NetBalance = balanceMultiplier * baseCons
		r.Amount += r.NetBalance * deltaYears

		// 边界处理
		if r.Amount < 0 {
			r.Amount = 0
		}
	}
}

func (e *GameEngine) updateSiloMetrics(silo *model.Silo, deltaYears float64) {
	// 1. 时间倒计时 (1小时 = 1年)
	silo.CountdownYears -= deltaYears
	if silo.CountdownYears < 0 {
		silo.CountdownYears = 0
	}

	// 2. 外部事件触发累积
	// 每小时增加一定基础值，受地堡凝聚力 (Cohesion) 影响
	silo.EventTrigger += (1.0 - silo.Cohesion) * deltaYears * 0.1

	// 3. 叛乱值逻辑
	// 计算全局平均恐慌值
	avgPanic := 0.0
	for _, p := range silo.Professions {
		avgPanic += p.PanicValue
	}
	avgPanic /= float64(len(silo.Professions))

	// 当 (1 - Legitimacy) * PanicValue > 0.1 (阈值) 时，叛乱值上升
	threshold := 0.1
	stressFactor := (1.0 - silo.Legitimacy) * avgPanic
	if stressFactor > threshold {
		silo.Rebellion += (stressFactor - threshold) * deltaYears * 0.05 // 上升速率系数
	} else {
		silo.Rebellion -= 0.01 * deltaYears // 自然消退
	}

	if silo.Rebellion > 1.0 {
		silo.Rebellion = 1.0
	} else if silo.Rebellion < 0 {
		silo.Rebellion = 0
	}
}

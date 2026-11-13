package service

// Silo 18 身份与开局 (silo18-event-chain-design §2.1/§7-S1)。
//
// 双堡模式的第二堡固定为 18 号地堡，跑原著事件链 (作者裁决)；玩家堡 (40 号)
// 保持原创内容。18 号的身份、前史状态与具名人物全部为原著正典常量：
//
//   - 前史承载：大革命 (2212) / 17 号陷落 / Allison 清洁死亡不做成可玩节点，
//     由开局计量 (history_burden 等) 与特质承载。
//   - 具名人物是真实个体：状态落在 model.Resident (有状态、会死)，生死只由
//     链条节点权威设定 (见 silo18_chain.go)；AI 代操作只当群演，不掌其生死。
//   - 其他地堡 (玩家堡) 的居民不是正典个体，保留系统生成名。
//
// 本文件不消耗 RNG：具名人物使用固定楼层/思潮/关系，不扰动单一共享 RNG 流。

import (
	"silo40/internal/engine"
	"silo40/internal/model"
)

const (
	// silo18CanonTrait 18 号标记特质：链条调度器只对带该特质的非玩家堡激活。
	// 旧双堡存档 (通用邻堡) 没有该特质，读档后链条保持休眠、行为不变。
	silo18CanonTrait = "silo18_canon"

	// silo18Name 18 号地堡的固定名称 (作者裁决：第二堡就是 18 号)。
	silo18Name = "18 号地堡"

	// silo18AgentName 18 号 AI 代理操作者的中性称呼。
	silo18AgentName = "18 号 AI 操作员"

	// silo18NamedTag 具名正典人物的居民标签 (生死由链条节点权威设定)。
	silo18NamedTag = "named"
)

// silo18NamedPerson 正典具名人物定义 (正篇前夜状态)。
type silo18NamedPerson struct {
	Name       string
	Profession string
	Role       string  // 角色标签
	Floor      int     // 固定居住楼层 (不消耗 RNG)
	ProForeign float64 // 固定亲外思潮
	Loyalty    float64 // 固定忠诚思潮
	Influence  float64 // 固定影响力
}

// silo18NamedCast 18 号具名人物表。出处：docs/research (Wool/Dust 篇)。
// 生死由链条节点设定：Holston(N1)、Jahns/Marnes(N2)、George(N4)、Lukas(N8)。
var silo18NamedCast = []silo18NamedPerson{
	{Name: "Jahns", Profession: "Mayor", Role: "mayor", Floor: 3, ProForeign: 0.10, Loyalty: 0.75, Influence: 0.60},
	{Name: "Marnes", Profession: "Mayor", Role: "deputy_mayor", Floor: 3, ProForeign: 0.08, Loyalty: 0.72, Influence: 0.40},
	{Name: "Bernard", Profession: "IT", Role: "it_head", Floor: 5, ProForeign: 0.05, Loyalty: 0.85, Influence: 0.55},
	{Name: "Lukas Kyle", Profession: "IT", Role: "it_engineer", Floor: 5, ProForeign: 0.20, Loyalty: 0.55, Influence: 0.30},
	{Name: "Holston", Profession: "Judicial", Role: "sheriff", Floor: 8, ProForeign: 0.30, Loyalty: 0.50, Influence: 0.45},
	{Name: "Juliette", Profession: "Mechanical", Role: "mechanic", Floor: 122, ProForeign: 0.55, Loyalty: 0.35, Influence: 0.50},
	{Name: "Walker", Profession: "Mechanical", Role: "mechanic_elder", Floor: 124, ProForeign: 0.35, Loyalty: 0.45, Influence: 0.35},
	{Name: "Knox", Profession: "Mechanical", Role: "mechanic", Floor: 125, ProForeign: 0.30, Loyalty: 0.50, Influence: 0.30},
	{Name: "Shirly", Profession: "Mechanical", Role: "mechanic", Floor: 126, ProForeign: 0.25, Loyalty: 0.55, Influence: 0.30},
	{Name: "George", Profession: "Mechanical", Role: "mechanic", Floor: 127, ProForeign: 0.20, Loyalty: 0.50, Influence: 0.20},
}

// createSilo18 按原著正篇前夜构建 18 号地堡。标准骨架来自 CreateInitialSilo，
// 正典覆写包括：名称、标记特质、前史计量与具名人物。不消耗 RNG。
func createSilo18(startYear int) *model.Silo {
	silo := engine.CreateInitialSilo(silo18Name, startYear, nil)
	silo.Traits = append(silo.Traits, silo18CanonTrait)

	// 前史状态 (设计 §2.1)：大革命只剩传说、17 号陷落、Allison 清洁死亡带来
	// 的"想出去"禁忌紧张，全部由开局计量承载。
	silo.HistoryBurden = 0.55
	silo.SetMayorLegitimacy(0.72)
	silo.Cohesion = 0.78
	silo.Rebellion = 0.08
	silo.DeptTension = 0.25
	silo.ClassFragmentation = 0.22
	silo.EventTrigger = 0.05

	appendSilo18NamedCast(silo)
	engine.AssignMayorHolder(silo, "Jahns")
	return silo
}

// appendSilo18NamedCast 把正典具名人物追加为真实 Resident 个体。
// 不挂 Cohort (独立于聚合人口动力学)：其状态由链条节点权威维护。
func appendSilo18NamedCast(silo *model.Silo) {
	profByName := map[string]*model.Profession{}
	for i := range silo.Professions {
		profByName[silo.Professions[i].Name] = &silo.Professions[i]
	}
	nextID := uint(len(silo.Residents) + 1)
	for _, person := range silo18NamedCast {
		prof := profByName[person.Profession]
		if prof == nil {
			continue
		}
		resident := model.Resident{
			ID:           nextID,
			SiloID:       silo.ID,
			Name:         person.Name,
			ProfessionID: prof.ID,
			Profession:   prof.Name,
			HomeFloor:    person.Floor,
			Tags:         []string{silo18NamedTag, silo18CanonTrait, person.Role},
			Loyalty:      0.5,
			Ambition:     0.2,
			Ideologies: map[string]float64{
				model.IdeologyProForeign: person.ProForeign,
				model.IdeologyLoyalty:    person.Loyalty,
			},
			Influence: person.Influence,
			Relations: map[string]float64{},
			Alive:     true,
		}
		silo.Residents = append(silo.Residents, resident)
		nextID++
	}
}

// silo18Canon 世界内带正典标记的 18 号；单堡 / 通用邻堡配置返回 nil。
func (s *GameService) silo18Canon() *model.Silo {
	if s.world == nil || !s.multiSilo() {
		return nil
	}
	for _, silo := range s.silos {
		if silo == nil || silo.ID == s.world.PlayerSiloID {
			continue
		}
		if residentSiloHasTrait(silo, silo18CanonTrait) {
			return silo
		}
	}
	return nil
}

func residentSiloHasTrait(silo *model.Silo, trait string) bool {
	for _, t := range silo.Traits {
		if t == trait {
			return true
		}
	}
	return false
}

// namedResident 按名查找具名正典人物。
func namedResident(silo *model.Silo, name string) *model.Resident {
	if silo == nil {
		return nil
	}
	for i := range silo.Residents {
		if silo.Residents[i].Name == name && residentHasTag(&silo.Residents[i], silo18NamedTag) {
			return &silo.Residents[i]
		}
	}
	return nil
}

func residentHasTag(resident *model.Resident, tag string) bool {
	for _, t := range resident.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

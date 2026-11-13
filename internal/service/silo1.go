package service

import (
	"fmt"
	"strings"

	"silo40/internal/engine"
	"silo40/internal/model"
)

const (
	silo1FullTrait  = "silo1_full_simulation"
	silo1AgentName  = "1 号 AI 操作员"
	silo1SignalID   = "silo1_department_signal"
	silo1SignalName = "1 号地堡"
)

// createSilo1 builds a peer silo with the same local rules as every other
// fully simulated silo. Its Archives department and different power balance
// give it a distinct internal agenda without granting world-level privileges.
func createSilo1(startYear int) *model.Silo {
	silo := engine.CreateInitialSilo(silo1SignalName, startYear, nil)
	silo.Traits = append(silo.Traits, silo1FullTrait)
	silo.HistoryBurden = 0.35
	silo.SetMayorLegitimacy(0.86)
	silo.Cohesion = 0.82
	silo.Rebellion = 0.04
	silo.DeptTension = 0.18

	for i := range silo.Professions {
		if silo.Professions[i].Name == "Supply" {
			silo.Professions[i].Population -= 300
		}
		silo.Professions[i].Relations["Archives"] = 0.2
	}
	archives := model.Profession{
		ID:             10,
		Name:           "Archives",
		ClassType:      "ELITE",
		Population:     300,
		Ideologies:     map[string]float64{model.IdeologyProForeign: 0.12, model.IdeologyLoyalty: 0.8},
		PanicValue:     0.02,
		Productivity:   1,
		PowerLevel:     8,
		Zone:           "Upper",
		KnownFragments: []string{"Mayor_3", "IT_4"},
		Relations:      map[string]float64{},
		ActionPoints:   50,
		Traits:         []string{"record_keepers"},
	}
	for _, profession := range silo.Professions {
		archives.Relations[profession.Name] = 0.2
	}
	silo.Professions = append(silo.Professions, archives)
	for i := range silo.Floors {
		if silo.Floors[i].Level >= 11 && silo.Floors[i].Level <= 15 {
			silo.Floors[i].Function = "Archives"
		}
	}
	return silo
}

// fullSilo1 returns the opt-in peer implementation. Legacy games still use the
// World.Silo1 script state and therefore return nil here.
func fullSilo1In(silos []*model.Silo) *model.Silo {
	for _, silo := range silos {
		if residentSiloHasTrait(silo, silo1FullTrait) {
			return silo
		}
	}
	return nil
}

func (s *GameService) fullSilo1() *model.Silo {
	return fullSilo1In(s.silos)
}

// syncSilo1Peer keeps the legacy world fact alias consistent with the real
// silo. Silo 18's canon terminal signal can still destroy Silo 1 through the
// same world fact used by the original two-silo scenario.
func (s *GameService) syncSilo1Peer() {
	silo := s.fullSilo1()
	if silo == nil || s.world == nil {
		return
	}
	if !s.world.Silo1.Alive && (silo.VictoryStatus == nil || !strings.EqualFold(silo.VictoryStatus.Type, "death")) {
		// The canon world fact is authoritative even if Silo 1 reached another
		// local ending first; never resurrect it through peer synchronization.
		silo.VictoryStatus = &model.VictoryStatus{
			IsWon:       false,
			Type:        "DEATH",
			Description: "1 号地堡在外部终局信号后停止运转。",
		}
	}
	destroyed := silo.VictoryStatus != nil && strings.EqualFold(silo.VictoryStatus.Type, "death")
	s.world.Silo1.Alive = !destroyed
	if destroyed {
		s.world.Flags[model.WorldFlagSilo1Destroyed] = true
	}
}

// captureSilo1Signals exposes a throttled intelligence product derived from
// Silo 1's actual local departments. It changes no target state and consumes
// no RNG, so observation cannot become a privileged control channel.
func (s *GameService) captureSilo1Signals(logs *[]string, stories *[]model.StoryEvent) {
	target := s.fullSilo1()
	if target == nil || s.world == nil || !s.world.Silo1.Alive || s.gameOver() {
		return
	}
	now := gameTimestamp(s.world.CurrentYear, s.world.CurrentMonth)
	if s.world.Silo1.LastSignalTimestamp != 0 && now-s.world.Silo1.LastSignalTimestamp < 3*daysPerMonth {
		return
	}

	selected := ""
	highestPanic := 0.0
	for _, profession := range target.Professions {
		if profession.PanicValue > highestPanic {
			selected = profession.Name
			highestPanic = profession.PanicValue
		}
	}
	report := "Archives 与 IT 完成例行数据交换；1 号各部门仍按本地议程运行，未检测到全域指令。"
	if selected != "" {
		report = fmt.Sprintf("%s 的本地恐慌达到 %s 档，1 号整体叛乱风险为 %s 档；信号来自其月度部门结算。",
			selected, rebellionBand(highestPanic), rebellionBand(target.Rebellion))
	}
	story := &model.StoryEvent{
		ID:          silo1SignalID,
		Category:    "special",
		Title:       "截获：1 号地堡部门信号",
		Description: report,
		Type:        "EXTERNAL",
	}
	s.currentSilo = s.silo
	s.currentAgent = s.agent
	s.recordStories([]*model.StoryEvent{story}, "INTEL:SILO1")
	s.currentSilo = nil
	s.currentAgent = nil
	*stories = append(*stories, *story)
	*logs = append(*logs, "[Intel] Captured a Silo 1 department signal")
	s.world.Silo1.LastSignalTimestamp = now
	s.world.Silo1.LastReport = report
}

func (s *GameService) silo1DepartmentNames() []string {
	target := s.fullSilo1()
	if target == nil {
		return nil
	}
	names := make([]string, 0, len(target.Professions))
	for _, profession := range target.Professions {
		names = append(names, profession.Name)
	}
	return names
}

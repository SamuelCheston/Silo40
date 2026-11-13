package service

import (
	"fmt"
	"strings"

	"silo40/internal/engine"
	"silo40/internal/model"
)

const (
	Silo1OpIntel      = "SILO1_INTEL"
	Silo1OpInfiltrate = "SILO1_INFILTRATE"
	Silo1OpDeceive    = "SILO1_DECEIVE"
	Silo1OpTrade      = "SILO1_TRADE"
)

type silo1OperationDef struct {
	ActionID         string
	Label            string
	Description      string
	Effect           string
	APCost           int
	DelayMonths      int
	Exposure         float64
	OpeningSuspicion float64
	TraceSuspicion   float64
}

var silo1OperationDefinitions = []silo1OperationDef{
	{
		ActionID: Silo1OpIntel, Label: "渗透情报网", APCost: 10, DelayMonths: 1,
		Description: "1 个月后确认 Silo 1 的精确合法性、凝聚力、叛乱与部门状态。暴露 +4%，结算时怀疑度 +0.01。",
		Effect:      "获取目标实测报告", Exposure: 0.04, OpeningSuspicion: 0.002, TraceSuspicion: 0.01,
	},
	{
		ActionID: Silo1OpDeceive, Label: "投放伪造档案", APCost: 20, DelayMonths: 2,
		Description: "2 个月后 Silo 1 叛乱 +0.15，各部门恐慌与亲外思潮上升。暴露 +10%，结算时怀疑度 +0.04。",
		Effect:      "推动目标叛乱与亲外思潮", Exposure: 0.10, OpeningSuspicion: 0.006, TraceSuspicion: 0.04,
	},
}

func silo1OperationDefinition(actionID string) (silo1OperationDef, bool) {
	for _, def := range silo1OperationDefinitions {
		if def.ActionID == strings.TrimSpace(actionID) {
			return def, true
		}
	}
	return silo1OperationDef{}, false
}

func silo1OperationAllowed(actionID string) bool {
	return actionID == Silo1OpIntel || actionID == Silo1OpDeceive
}

func (s *GameService) silo1OperationActions() []model.PlayerActionMeta {
	enabled := s.fullSilo1() != nil && s.stationSilo() != nil && s.world != nil && s.world.Silo1.Alive && !s.gameOver()
	actions := make([]model.PlayerActionMeta, 0, 2)
	for _, def := range silo1OperationDefinitions {
		if !silo1OperationAllowed(def.ActionID) {
			continue
		}
		actions = append(actions, model.PlayerActionMeta{
			ID: def.ActionID, Source: "silo1", Label: def.Label,
			Description: def.Description, Group: "silo1",
			ActionType: model.ActionSilo1Operation, TargetType: "NONE",
			APCost: def.APCost, DurationMonths: 0, Enabled: enabled,
		})
	}
	return actions
}

// scheduleSilo1Operation starts a real delayed operation against the peer.
// AP and the opening trace are applied immediately; target mutation only occurs
// when the operation reaches its due month.
func (s *GameService) scheduleSilo1Operation(action model.AgentAction) model.ActionResult {
	def, ok := silo1OperationDefinition(action.ActionID)
	if !ok || !silo1OperationAllowed(action.ActionID) {
		return model.ActionResult{Executed: false, Message: "Silo 1 channel permits only intelligence and deception."}
	}
	target := s.fullSilo1()
	if target == nil || s.world == nil || !s.world.Silo1.Alive {
		return model.ActionResult{Executed: false, Message: "Silo 1 is not available as a live target."}
	}
	if s.agent.ActionPoints < action.Cost {
		return model.ActionResult{Executed: false, Message: "Not enough Action Points (AP)."}
	}

	s.agent.ActionPoints -= action.Cost
	s.agent.SuspicionLevel += def.OpeningSuspicion
	s.world.Silo1.Exposure = clampSilo1Unit(s.world.Silo1.Exposure + def.Exposure)
	s.world.Silo1.NextOperationID++
	due := engine.AdvanceTime(engine.GameTime{Year: s.world.CurrentYear, Month: s.world.CurrentMonth}, def.DelayMonths)
	op := model.Silo1Operation{
		ID: s.world.Silo1.NextOperationID, ActionID: def.ActionID, Label: def.Label,
		Status: "PENDING", StartYear: s.world.CurrentYear, StartMonth: s.world.CurrentMonth,
		DueYear: due.Year, DueMonth: due.Month, Effect: def.Effect, TraceSuspicion: def.TraceSuspicion,
	}
	s.world.Silo1.Operations = append(s.world.Silo1.Operations, op)
	story := &model.StoryEvent{
		ID: "silo1_operation_scheduled", Category: "player_action",
		Title: "跨堡指令已下达：" + def.Label,
		Description: fmt.Sprintf("行动 #%d 将在 Y%d M%d 生效。%s 当前反向追查暴露为 %.0f%%。",
			op.ID, op.DueYear, op.DueMonth, def.Description, s.world.Silo1.Exposure*100),
		Type: "EXTERNAL",
	}
	s.recordStories([]*model.StoryEvent{story}, "ACTION:SILO1_OPERATION")
	return model.ActionResult{
		Executed: true,
		Message:  fmt.Sprintf("Operation #%d scheduled against Silo 1; due Y%d M%d.", op.ID, op.DueYear, op.DueMonth),
	}
}

// resolveSilo1Operations runs after the world clock advances, so a one-month
// operation ordered at M1 resolves on the first advance into M2. Operations are
// resolved in creation order and consume no RNG.
func (s *GameService) resolveSilo1Operations(logs *[]string, stories *[]model.StoryEvent) {
	if s.world == nil || s.fullSilo1() == nil {
		return
	}
	now := engine.GameTime{Year: s.world.CurrentYear, Month: s.world.CurrentMonth}
	for i := range s.world.Silo1.Operations {
		op := &s.world.Silo1.Operations[i]
		if op.Status != "PENDING" || engine.CompareTime(engine.GameTime{Year: op.DueYear, Month: op.DueMonth}, now) > 0 {
			continue
		}
		def, ok := silo1OperationDefinition(op.ActionID)
		if !ok {
			op.Status = "FAILED"
			op.Resolution = "Unknown operation definition; no target effect applied."
			*logs = append(*logs, fmt.Sprintf("[Silo1Operation] #%d failed: unknown definition", op.ID))
			continue
		}
		op.Resolution = s.applySilo1Operation(def, op.ID)
		op.Status = "RESOLVED"
		s.agent.SuspicionLevel += def.TraceSuspicion
		counterReport := s.applySilo1Counterintelligence()
		report := fmt.Sprintf("行动 #%d 目标实测：%s 反向追查留下 %.2f 怀疑度。%s",
			op.ID, op.Resolution, def.TraceSuspicion, counterReport)
		s.world.Silo1.LastReport = report
		s.world.Silo1.LastSignalTimestamp = gameTimestamp(now.Year, now.Month)
		s.recordSilo1OperationStories(op, report, stories)
		*logs = append(*logs, fmt.Sprintf("[Silo1Operation] #%d resolved against live Silo 1", op.ID))
	}
}

func (s *GameService) applySilo1Operation(def silo1OperationDef, operationID int) string {
	target := s.fullSilo1()
	historyBefore := snapshotHistoryMetrics(target)
	switch def.ActionID {
	case Silo1OpIntel:
		resolution := fmt.Sprintf("Y%d M%d 合法性 %.2f、凝聚力 %.2f、叛乱 %.2f；部门构成 %s。",
			target.CurrentYear, target.CurrentMonth, target.MayorLegitimacy(), target.Cohesion, target.Rebellion,
			strings.Join(s.silo1DepartmentNames(), "/"))
		s.recordHistoryImpact(target, operationID, def, historyBefore, resolution)
		return resolution
	case Silo1OpDeceive:
		before := target.Rebellion
		target.Rebellion = clampSilo1Unit(target.Rebellion + 0.15)
		target.HistoryBurden = clampSilo1Unit(target.HistoryBurden + 0.02)
		for i := range target.Professions {
			target.Professions[i].PanicValue = clampSilo1Unit(target.Professions[i].PanicValue + 0.06)
			target.Professions[i].Ideologies[model.IdeologyProForeign] = clampSilo1Unit(target.Professions[i].Ideologies[model.IdeologyProForeign] + 0.04)
		}
		for i := range target.Cohorts {
			target.Cohorts[i].Ideologies[model.IdeologyProForeign] = clampSilo1Unit(target.Cohorts[i].Ideologies[model.IdeologyProForeign] + 0.04)
		}
		resolution := fmt.Sprintf("叛乱 %.2f→%.2f；全部部门恐慌 +0.06、亲外思潮 +0.04。", before, target.Rebellion)
		s.recordHistoryImpact(target, operationID, def, historyBefore, resolution)
		return resolution
	default:
		return "No effect applied."
	}
}

func (s *GameService) applySilo1Counterintelligence() string {
	stage := int(s.world.Silo1.Exposure / 0.25)
	if stage > 3 {
		stage = 3
	}
	if stage <= s.world.Silo1.CounterintelStage {
		return ""
	}
	s.world.Silo1.CounterintelStage = stage
	s.agent.SuspicionLevel += 0.08
	station := s.stationSilo()
	if station != nil {
		station.SetMayorLegitimacy(station.MayorLegitimacy() - 0.02)
	}
	return fmt.Sprintf("Silo 1 反间谍阶段升至 %d：特工怀疑度额外 +0.08，驻地合法性 -0.02。", stage)
}

func (s *GameService) recordSilo1OperationStories(op *model.Silo1Operation, report string, stories *[]model.StoryEvent) {
	targetStory := &model.StoryEvent{
		ID: "silo1_operation_effect", Category: "player_action",
		Title: "1 号本地后果：" + op.Label, Description: op.Resolution, Type: "EXTERNAL",
	}
	s.currentSilo = s.fullSilo1()
	s.currentAgent = s.agents[s.currentSilo.ID]
	s.recordStories([]*model.StoryEvent{targetStory}, "CROSS_EFFECT:SILO40")
	playerStory := &model.StoryEvent{
		ID: "silo1_operation_report", Category: "player_action",
		Title: "跨堡行动回执：" + op.Label, Description: report, Type: "EXTERNAL",
	}
	s.currentSilo = s.stationSilo()
	s.currentAgent = s.agent
	s.recordStories([]*model.StoryEvent{playerStory}, "OPERATION_REPORT:SILO1")
	s.currentSilo = nil
	s.currentAgent = nil
	*stories = append(*stories, *playerStory)
}

func clampSilo1Unit(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

package service

import (
	"fmt"
	"strings"

	"silo40/internal/engine"
	"silo40/internal/model"
)

const (
	StationOpIntel      = "STATION_INTEL"
	StationOpInfiltrate = "STATION_INFILTRATE"
	StationOpDeceive    = "STATION_DECEIVE"
	StationOpTrade      = "STATION_TRADE"
)

var stationOperationDefinitions = []silo1OperationDef{
	{
		ActionID: StationOpIntel, Label: "渗透驻地情报网", APCost: 10, DelayMonths: 1,
		Description: "1 个月后确认驻地的精确合法性、凝聚力、叛乱与部门状态。暴露 +4%，结算时怀疑度 +0.01。",
		Effect:      "获取驻地实测报告", Exposure: 0.04, OpeningSuspicion: 0.002, TraceSuspicion: 0.01,
	},
	{
		ActionID: StationOpInfiltrate, Label: "策反驻地机工线人", APCost: 25, DelayMonths: 2,
		Description: "2 个月后驻地合法性 -0.12、凝聚力 -0.08，并持续提高部门紧张。暴露 +14%，结算时怀疑度 +0.05。",
		Effect:      "削弱驻地合法性与凝聚力", Exposure: 0.14, OpeningSuspicion: 0.008, TraceSuspicion: 0.05,
	},
	{
		ActionID: StationOpDeceive, Label: "在驻地投放伪造档案", APCost: 20, DelayMonths: 2,
		Description: "2 个月后驻地叛乱 +0.15，各部门恐慌 +0.06、亲外思潮 +0.04。暴露 +12%，结算时怀疑度 +0.05。",
		Effect:      "推动驻地叛乱与亲外思潮", Exposure: 0.12, OpeningSuspicion: 0.006, TraceSuspicion: 0.05,
	},
	{
		ActionID: StationOpTrade, Label: "驻地内部黑市交易", APCost: 15, DelayMonths: 1,
		Description: "1 个月后驻地消耗 100 Supplies，合法性与凝聚力上升，特工威望 +3；不向 40 号转移资源。暴露 +6%，结算时怀疑度 +0.02。",
		Effect:      "以驻地物资换取稳定与威望", Exposure: 0.06, OpeningSuspicion: 0.004, TraceSuspicion: 0.02,
	},
}

func stationOperationDefinition(actionID string) (silo1OperationDef, bool) {
	for _, def := range stationOperationDefinitions {
		if def.ActionID == strings.TrimSpace(actionID) {
			return def, true
		}
	}
	return silo1OperationDef{}, false
}

func (s *GameService) stationOperationActions() []model.PlayerActionMeta {
	target := s.stationSilo()
	enabled := target != nil && target.VictoryStatus == nil && !s.gameOver()
	actions := make([]model.PlayerActionMeta, 0, len(stationOperationDefinitions))
	for _, def := range stationOperationDefinitions {
		actions = append(actions, model.PlayerActionMeta{
			ID: def.ActionID, Source: "station", Label: def.Label,
			Description: def.Description, Group: "station",
			ActionType: model.ActionStationOperation, TargetType: "NONE",
			APCost: def.APCost, DurationMonths: 0, Enabled: enabled,
		})
	}
	return actions
}

func (s *GameService) scheduleStationOperation(action model.AgentAction) model.ActionResult {
	def, ok := stationOperationDefinition(action.ActionID)
	if !ok {
		return model.ActionResult{Executed: false, Message: "Unknown station operation."}
	}
	target := s.stationSilo()
	if target == nil || target.VictoryStatus != nil || s.world == nil {
		return model.ActionResult{Executed: false, Message: "No valid assigned station target."}
	}
	if s.agent.ActionPoints < action.Cost {
		return model.ActionResult{Executed: false, Message: "Not enough Action Points (AP)."}
	}

	s.agent.ActionPoints -= action.Cost
	s.agent.SuspicionLevel += def.OpeningSuspicion
	s.world.StationOps.Exposure = clampSilo1Unit(s.world.StationOps.Exposure + def.Exposure)
	s.world.StationOps.NextOperationID++
	due := engine.AdvanceTime(engine.GameTime{Year: s.world.CurrentYear, Month: s.world.CurrentMonth}, def.DelayMonths)
	op := model.Silo1Operation{
		ID: s.world.StationOps.NextOperationID, TargetSiloID: target.ID,
		ActionID: def.ActionID, Label: def.Label, Status: "PENDING",
		StartYear: s.world.CurrentYear, StartMonth: s.world.CurrentMonth,
		DueYear: due.Year, DueMonth: due.Month, Effect: def.Effect, TraceSuspicion: def.TraceSuspicion,
	}
	s.world.StationOps.Operations = append(s.world.StationOps.Operations, op)
	story := &model.StoryEvent{
		ID: "station_operation_scheduled", Category: "player_action",
		Title: "驻地指令已下达：" + def.Label,
		Description: fmt.Sprintf("行动 #%d 将在 Y%d M%d 对驻地 %s 生效。当前反向追查暴露为 %.0f%%。",
			op.ID, op.DueYear, op.DueMonth, target.Name, s.world.StationOps.Exposure*100),
		Type: "EXTERNAL",
	}
	s.recordStories([]*model.StoryEvent{story}, "ACTION:STATION_OPERATION")
	return model.ActionResult{
		Executed: true,
		Message:  fmt.Sprintf("Operation #%d scheduled at %s; due Y%d M%d.", op.ID, target.Name, op.DueYear, op.DueMonth),
	}
}

func (s *GameService) resolveStationOperations(logs *[]string, stories *[]model.StoryEvent) {
	if s.world == nil || s.stationSilo() == nil {
		return
	}
	now := engine.GameTime{Year: s.world.CurrentYear, Month: s.world.CurrentMonth}
	for i := range s.world.StationOps.Operations {
		op := &s.world.StationOps.Operations[i]
		if op.Status != "PENDING" || engine.CompareTime(engine.GameTime{Year: op.DueYear, Month: op.DueMonth}, now) > 0 {
			continue
		}
		def, ok := stationOperationDefinition(op.ActionID)
		if !ok {
			op.Status = "FAILED"
			op.Resolution = "Unknown operation definition; no target effect applied."
			*logs = append(*logs, fmt.Sprintf("[StationOperation] #%d failed: unknown definition", op.ID))
			continue
		}
		if target := s.stationSilo(); target == nil || target.VictoryStatus != nil {
			op.Status = "SUPERSEDED"
			op.Resolution = "驻地已先完成结局结算，本次行动未生效。"
		} else {
			op.Resolution = s.applyStationOperation(def, op.ID)
			op.Status = "RESOLVED"
		}
		s.agent.SuspicionLevel += def.TraceSuspicion
		counterReport := s.applyStationCounterintelligence()
		report := fmt.Sprintf("行动 #%d 的驻地实测：%s 反向追查留下 %.2f 怀疑度。%s",
			op.ID, op.Resolution, def.TraceSuspicion, counterReport)
		s.world.StationOps.LastReport = report
		s.world.StationOps.LastSignalTimestamp = gameTimestamp(now.Year, now.Month)
		s.recordStationOperationStories(op, report, stories)
		*logs = append(*logs, fmt.Sprintf("[StationOperation] #%d resolved against assigned station", op.ID))
	}
}

func (s *GameService) applyStationOperation(def silo1OperationDef, operationID int) string {
	target := s.stationSilo()
	historyBefore := snapshotHistoryMetrics(target)
	switch def.ActionID {
	case StationOpIntel:
		resolution := fmt.Sprintf("Y%d M%d 合法性 %.2f、凝聚力 %.2f、叛乱 %.2f；部门构成 %s。",
			target.CurrentYear, target.CurrentMonth, target.MayorLegitimacy(), target.Cohesion, target.Rebellion,
			strings.Join(stationDepartmentNames(target), "/"))
		s.recordHistoryImpact(target, operationID, def, historyBefore, resolution)
		return resolution
	case StationOpInfiltrate:
		beforeLegitimacy, beforeCohesion := target.MayorLegitimacy(), target.Cohesion
		target.SetMayorLegitimacy(target.MayorLegitimacy() - 0.12)
		target.Cohesion = clampSilo1Unit(target.Cohesion - 0.08)
		target.DeptTension = clampSilo1Unit(target.DeptTension + 0.08)
		for i := range target.Cohorts {
			target.Cohorts[i].Loyalty = clampSilo1Unit(target.Cohorts[i].Loyalty - 0.04)
		}
		for i := range target.Professions {
			if target.Professions[i].Name == "Mechanical" {
				target.Professions[i].PanicValue = clampSilo1Unit(target.Professions[i].PanicValue + 0.08)
			}
		}
		resolution := fmt.Sprintf("合法性 %.2f→%.2f，凝聚力 %.2f→%.2f，部门紧张升至 %.2f。",
			beforeLegitimacy, target.MayorLegitimacy(), beforeCohesion, target.Cohesion, target.DeptTension)
		s.recordHistoryImpact(target, operationID, def, historyBefore, resolution)
		return resolution
	case StationOpDeceive:
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
	case StationOpTrade:
		before := siloResourceAmount(target, "Supplies")
		for i := range target.Resources {
			if target.Resources[i].Type == "Supplies" {
				consume := 100.0
				if target.Resources[i].Amount < consume {
					consume = target.Resources[i].Amount
				}
				target.Resources[i].Amount -= consume
				break
			}
		}
		after := siloResourceAmount(target, "Supplies")
		target.SetMayorLegitimacy(target.MayorLegitimacy() + 0.03)
		target.Cohesion = clampSilo1Unit(target.Cohesion + 0.05)
		s.agent.PoliticalPrestige += 3
		resolution := fmt.Sprintf("驻地黑市消耗 %.0f Supplies（%.0f→%.0f）；合法性 %.2f、凝聚力 %.2f，特工威望 +3。",
			before-after, before, after, target.MayorLegitimacy(), target.Cohesion)
		s.recordHistoryImpact(target, operationID, def, historyBefore, resolution)
		return resolution
	default:
		return "No effect applied."
	}
}

func stationDepartmentNames(target *model.Silo) []string {
	names := make([]string, 0, len(target.Professions))
	for _, profession := range target.Professions {
		names = append(names, profession.Name)
	}
	return names
}

func (s *GameService) applyStationCounterintelligence() string {
	stage := int(s.world.StationOps.Exposure / 0.25)
	if stage > 3 {
		stage = 3
	}
	if stage <= s.world.StationOps.CounterintelStage {
		return ""
	}
	s.world.StationOps.CounterintelStage = stage
	s.agent.SuspicionLevel += 0.08
	station := s.stationSilo()
	station.SetMayorLegitimacy(station.MayorLegitimacy() - 0.02)
	return fmt.Sprintf("驻地反间谍阶段升至 %d：特工怀疑度额外 +0.08，驻地合法性 -0.02。", stage)
}

func (s *GameService) recordStationOperationStories(op *model.Silo1Operation, report string, stories *[]model.StoryEvent) {
	target := s.stationSilo()
	targetStory := &model.StoryEvent{
		ID: "station_operation_effect", Category: "player_action",
		Title: "驻地本地后果：" + op.Label, Description: op.Resolution, Type: "EXTERNAL",
	}
	s.currentSilo = target
	s.currentAgent = s.agent
	s.recordStories([]*model.StoryEvent{targetStory}, "ACTION:STATION_OPERATION")
	playerStory := &model.StoryEvent{
		ID: "station_operation_report", Category: "player_action",
		Title: "驻地行动回执：" + op.Label, Description: report, Type: "EXTERNAL",
	}
	s.recordStories([]*model.StoryEvent{playerStory}, "OPERATION_REPORT:STATION")
	s.currentSilo = nil
	s.currentAgent = nil
	*stories = append(*stories, *playerStory)
}

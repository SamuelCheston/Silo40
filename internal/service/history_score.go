package service

import (
	"fmt"
	"math"
	"strings"

	"silo40/internal/model"
)

type historyMetricSnapshot struct {
	legitimacy float64
	cohesion   float64
	rebellion  float64
	proForeign float64
	supplies   float64
}

func snapshotHistoryMetrics(silo *model.Silo) historyMetricSnapshot {
	return historyMetricSnapshot{
		legitimacy: silo.MayorLegitimacy(),
		cohesion:   silo.Cohesion,
		rebellion:  silo.Rebellion,
		proForeign: averageSiloProForeign(silo),
		supplies:   siloResourceAmount(silo, "Supplies"),
	}
}

func averageSiloProForeign(silo *model.Silo) float64 {
	if len(silo.Professions) == 0 {
		return 0
	}
	total := 0.0
	for _, profession := range silo.Professions {
		total += profession.Ideologies[model.IdeologyProForeign]
	}
	return total / float64(len(silo.Professions))
}

func siloResourceAmount(silo *model.Silo, resourceType string) float64 {
	for _, resource := range silo.Resources {
		if resource.Type == resourceType {
			return resource.Amount
		}
	}
	return 0
}

// recordHistoryImpact persists only the measured before/after delta of a
// resolved operation. Natural simulation and canon overrides are not included.
func (s *GameService) recordHistoryImpact(target *model.Silo, operationID int, def silo1OperationDef, before historyMetricSnapshot, summary string) {
	if s.world == nil || target == nil {
		return
	}
	after := snapshotHistoryMetrics(target)
	impact := model.HistoryImpact{
		ID: len(s.world.HistoryImpacts) + 1, OperationID: operationID,
		TargetSiloID: target.ID, TargetName: target.Name, ActionID: def.ActionID, Label: def.Label,
		Year: target.CurrentYear, Month: target.CurrentMonth,
		LegitimacyDelta: after.legitimacy - before.legitimacy,
		CohesionDelta:   after.cohesion - before.cohesion,
		RebellionDelta:  after.rebellion - before.rebellion,
		ProForeignDelta: after.proForeign - before.proForeign,
		SuppliesDelta:   after.supplies - before.supplies,
		Summary:         summary,
	}
	impact.Points = historyImpactPoints(impact)
	s.world.HistoryImpacts = append(s.world.HistoryImpacts, impact)
}

func historyImpactPoints(impact model.HistoryImpact) int {
	points := math.Abs(impact.LegitimacyDelta)*100 +
		math.Abs(impact.CohesionDelta)*100 +
		math.Abs(impact.RebellionDelta)*120 +
		math.Abs(impact.ProForeignDelta)*80 +
		math.Abs(impact.SuppliesDelta)/100
	return int(math.Round(points))
}

// applyHistoryInfluenceScore is idempotent so repeated saves and resumes never
// add the history bonus twice. The original ending and its base score remain
// unchanged; this bonus is computed only after that ending has been reached.
func (s *GameService) applyHistoryInfluenceScore() {
	if !s.gameOver() || s.world == nil || s.silo.VictoryStatus == nil {
		return
	}
	if s.silo.VictoryStatus.Score == nil {
		s.silo.VictoryStatus.Score = &model.GameScore{}
	}
	if s.silo.VictoryStatus.Score.HistoryInfluence != nil {
		return
	}
	history := s.buildHistoryInfluenceScore()
	s.silo.VictoryStatus.Score.HistoryInfluence = &history
	s.silo.VictoryStatus.Score.Total += history.Total
}

func (s *GameService) buildHistoryInfluenceScore() model.HistoryInfluenceScore {
	result := model.HistoryInfluenceScore{Silos: []model.HistorySiloScore{}}
	targets := []*model.Silo{s.fullSilo1(), s.stationSilo()}
	for _, target := range targets {
		if target == nil {
			continue
		}
		breakdown := model.HistorySiloScore{SiloID: target.ID, Name: target.Name, Impacts: []model.HistoryImpact{}}
		parts := []string{}
		for _, impact := range s.world.HistoryImpacts {
			if impact.TargetSiloID != target.ID {
				continue
			}
			breakdown.OperationCount++
			breakdown.Points += impact.Points
			breakdown.Impacts = append(breakdown.Impacts, impact)
			parts = append(parts, fmt.Sprintf("Y%d M%d「%s」%s（%d 分）", impact.Year, impact.Month, impact.Label, impact.Summary, impact.Points))
		}
		if breakdown.Points > 100 {
			breakdown.Points = 100
		}
		if len(parts) == 0 {
			breakdown.Summary = "未记录到玩家造成的可量化历史变化，0 分。"
		} else {
			breakdown.Summary = strings.Join(parts, "；")
		}
		result.Total += breakdown.Points
		result.Silos = append(result.Silos, breakdown)
	}
	result.Grade = historyGrade(result.Total)
	result.Summary = fmt.Sprintf("历史引导 %d 分，评级 %s。仅统计已结算玩家行动的实测变化；自然演化与正典强制节点不计入。", result.Total, result.Grade)
	return result
}

func historyGrade(total int) string {
	switch {
	case total >= 150:
		return "S"
	case total >= 100:
		return "A"
	case total >= 50:
		return "B"
	case total > 0:
		return "C"
	default:
		return "D"
	}
}

package engine

import (
	"fmt"
	"strings"

	"silo40/internal/model"
)

// 准备度账本机制 (16 §2/§6.2，Layer 1)。
//
// 纪律：
//   - 账本是事实账本，不是成功率系数：每条记录必须有明确生产者
//     (玩家动作/内容事件)，禁止月度被动积累与 AP 直接兑换。
//   - 可数实物获取即付成本：资源划拨即扣减、暴露即累积。
//   - 未知字段 / 非法档位 / 余额不足 = 显式失败，不降级为假成功。
//   - 空账本 = 无准备 = 旧行为逐位不变；仲裁不读账本 (Layer 1 纪律：
//     准备度只在出发结算里塑形逃出人数，不进任何资格判定或加权)。

// 字段种类：整数量 / 非负量 / 0-1 程度量 / 布尔 / 档位 / 功能位。
const (
	prepFieldInt      = "int"
	prepFieldFloat    = "float"
	prepFieldUnit     = "unit" // 0-1
	prepFieldBool     = "bool"
	prepFieldStage    = "stage"
	prepFieldPlanRole = "plan_role"
)

// preparationWriteFields 可写字段白名单 (未知字段 = 校验失败)。
// plan_completeness 是派生量，只读不写。
var preparationWriteFields = map[string]string{
	"suits_functional": prepFieldInt,
	"supplies_packed":  prepFieldFloat,
	"materials_packed": prepFieldFloat,
	"drill_level":      prepFieldUnit,
	"care_ready":       prepFieldUnit,
	"prep_exposure":    prepFieldFloat,
	"package_quality":  prepFieldUnit, // 外溢账本字段
	"production_start": prepFieldBool,
	"reception_stage":  prepFieldStage,
	"plan_role":        prepFieldPlanRole,
}

// receptionStageOrder 接应档位只允许前进 (事实不可回退)。
var receptionStageOrder = map[string]int{
	model.ReceptionNone:      0,
	model.ReceptionSurveyed:  1,
	model.ReceptionConfirmed: 2,
	model.ReceptionStocked:   3,
}

var planRoleWhitelist = map[string]bool{
	model.PlanRoleMechanical:   true,
	model.PlanRoleMines:        true,
	model.PlanRoleSupply:       true,
	model.PlanRoleMedical:      true,
	model.PlanRoleAgricultural: true,
	model.PlanRoleOrder:        true,
}

var spilloverChannelKinds = map[string]float64{
	// 档位基础暴露成本 (建立即记)。TODO(balance)
	model.SpilloverChannelITLine:          0.12,
	model.SpilloverChannelCrossSignal:     0.06,
	model.SpilloverChannelPhysicalHandoff: 0.03,
}

// validatePreparationDelta 加载期校验 preparation_delta。
func validatePreparationDelta(effect ContentEffect) error {
	kind, ok := preparationWriteFields[effect.Metric]
	if !ok {
		return fmt.Errorf("preparation_delta field %q is not in the ledger whitelist", effect.Metric)
	}
	switch kind {
	case prepFieldStage:
		if _, ok := receptionStageOrder[effect.Flag]; !ok {
			return fmt.Errorf("preparation_delta reception_stage %q is not a valid stage", effect.Flag)
		}
	case prepFieldPlanRole:
		if !planRoleWhitelist[effect.Profession] {
			return fmt.Errorf("preparation_delta plan role %q is not in the R-2 roster", effect.Profession)
		}
	}
	return nil
}

// applyPreparationDelta 按白名单写账本；非法写入返回错误 (显式失败)。
func applyPreparationDelta(silo *model.Silo, effect ContentEffect) error {
	if silo == nil {
		return nil
	}
	kind, ok := preparationWriteFields[effect.Metric]
	if !ok {
		return fmt.Errorf("preparation_delta field %q is not in the ledger whitelist", effect.Metric)
	}
	prep := &silo.EscapePrep
	switch kind {
	case prepFieldInt:
		v := prep.SuitsFunctional + int(effect.Value)
		if v < 0 {
			v = 0
		}
		prep.SuitsFunctional = v
	case prepFieldFloat:
		switch effect.Metric {
		case "supplies_packed":
			prep.SuppliesPacked = clampMin0(prep.SuppliesPacked + effect.Value)
		case "materials_packed":
			prep.MaterialsPacked = clampMin0(prep.MaterialsPacked + effect.Value)
		case "prep_exposure":
			prep.PrepExposure = clampMin0(prep.PrepExposure + effect.Value)
		}
	case prepFieldUnit:
		switch effect.Metric {
		case "drill_level":
			prep.DrillLevel = clampUnit(prep.DrillLevel + effect.Value)
		case "care_ready":
			prep.CareReady = clampUnit(prep.CareReady + effect.Value)
		case "package_quality":
			silo.SpilloverPrep.PackageQuality = clampUnit(silo.SpilloverPrep.PackageQuality + effect.Value)
		}
	case prepFieldBool:
		if effect.Metric == "production_start" {
			prep.ProductionStart = effect.BoolValue
		}
	case prepFieldStage:
		current := prep.ReceptionStage
		if current == "" {
			current = model.ReceptionNone
		}
		if receptionStageOrder[effect.Flag] <= receptionStageOrder[current] {
			return fmt.Errorf("preparation_delta reception_stage cannot move %q -> %q (stages only advance)", current, effect.Flag)
		}
		prep.ReceptionStage = effect.Flag
	case prepFieldPlanRole:
		role := effect.Profession
		if !planRoleWhitelist[role] {
			return fmt.Errorf("preparation_delta plan role %q is not in the R-2 roster", role)
		}
		has := false
		for _, r := range prep.PlanRoles {
			if r == role {
				has = true
				break
			}
		}
		if effect.Value > 0 && !has {
			prep.PlanRoles = append(prep.PlanRoles, role)
		} else if effect.Value < 0 && has {
			kept := prep.PlanRoles[:0]
			for _, r := range prep.PlanRoles {
				if r != role {
					kept = append(kept, r)
				}
			}
			prep.PlanRoles = kept
		}
		prep.RecomputePlanCompleteness()
	}
	prep.Initialized = true
	return nil
}

// applyPreparationTransfer 资源划拨：Supplies/Materials → 封存账本。
// 划拨即扣减：筹备时刻真实扣减堡内资源，机会成本立即生效 (喂大
// supplies_shortage 一类既有规则)。余额不足 = 显式失败，不部分划拨。
// 换算系数 1:1 为基线。TODO(balance)
func applyPreparationTransfer(silo *model.Silo, resource string, amount float64) error {
	if silo == nil {
		return nil
	}
	// 资源类型大小写不敏感：脚本效果归一化会把 metric 小写化。
	canonical := ""
	switch strings.ToLower(resource) {
	case "supplies":
		canonical = "Supplies"
	case "materials":
		canonical = "Materials"
	default:
		return fmt.Errorf("preparation_transfer resource %q is not whitelisted (Supplies|Materials)", resource)
	}
	if amount <= 0 {
		return fmt.Errorf("preparation_transfer requires positive amount, got %v", amount)
	}
	for i := range silo.Resources {
		if silo.Resources[i].Type != canonical {
			continue
		}
		if silo.Resources[i].Amount < amount {
			return fmt.Errorf("preparation_transfer %s %v exceeds silo balance %v", canonical, amount, silo.Resources[i].Amount)
		}
		silo.Resources[i].Amount -= amount
		switch canonical {
		case "Supplies":
			silo.EscapePrep.SuppliesPacked += amount // TODO(balance): 人·月换算
		case "Materials":
			silo.EscapePrep.MaterialsPacked += amount
		}
		silo.EscapePrep.Initialized = true
		return nil
	}
	return fmt.Errorf("preparation_transfer: silo has no %s resource row", canonical)
}

// applySpilloverChannelAdd 登记一条受限信道 (建立侧)。投递与独立接收
// 属 Layer 3。Count = 目标堡 ID，Value = 可靠度，Flag = 种类。
// 建立即记暴露成本 (档位基础值，TODO(balance))——外溢侧暴露进入
// Silo 1 注意力轨道的接线属 Layer 3，本期只入账本。
func applySpilloverChannelAdd(silo *model.Silo, kind string, targetSilo int, reliability float64) error {
	if silo == nil {
		return nil
	}
	baseExposure, ok := spilloverChannelKinds[kind]
	if !ok {
		return fmt.Errorf("spillover_channel_add kind %q is not one of it_line|cross_signal|physical_handoff", kind)
	}
	if targetSilo <= 0 {
		return fmt.Errorf("spillover_channel_add requires a positive target silo id")
	}
	if reliability < 0 || reliability > 1 {
		return fmt.Errorf("spillover_channel_add reliability %v out of [0,1]", reliability)
	}
	silo.SpilloverPrep.Channels = append(silo.SpilloverPrep.Channels, model.SpilloverChannel{
		Kind:         kind,
		TargetSilo:   uint(targetSilo),
		Reliability:  reliability,
		ExposureCost: baseExposure,
		Year:         silo.CurrentYear,
		Month:        silo.CurrentMonth,
	})
	return nil
}

// readPreparationMetric 读账本数值字段 (含派生量 plan_completeness)，
// 供 preparation_gte 触发器与结算输入层使用。
func readPreparationMetric(silo *model.Silo, field string) (float64, bool) {
	if silo == nil {
		return 0, false
	}
	prep := silo.EscapePrep
	switch field {
	case "suits_functional":
		return float64(prep.SuitsFunctional), true
	case "supplies_packed":
		return prep.SuppliesPacked, true
	case "materials_packed":
		return prep.MaterialsPacked, true
	case "drill_level":
		return prep.DrillLevel, true
	case "care_ready":
		return prep.CareReady, true
	case "prep_exposure":
		return prep.PrepExposure, true
	case "plan_completeness":
		return prep.PlanCompleteness, true
	case "package_quality":
		return silo.SpilloverPrep.PackageQuality, true
	}
	return 0, false
}

// receptionStageAtLeast 接应档位比较 (供脚本 canTrigger 细读)。
func receptionStageAtLeast(silo *model.Silo, stage string) bool {
	if silo == nil {
		return false
	}
	current := silo.EscapePrep.ReceptionStage
	if current == "" {
		current = model.ReceptionNone
	}
	if _, ok := receptionStageOrder[stage]; !ok {
		return false
	}
	return receptionStageOrder[current] >= receptionStageOrder[stage]
}

func clampMin0(v float64) float64 {
	if v < 0 {
		return 0
	}
	return v
}

package service

// Θ 行为化 (22 黑箱学习系列 04 卷 §3、02 卷 §3.5)。自保档位的世界侧行为后果。
//
// 读权限口径：本文件读 Silo.BlackBox.Theta 真值是**世界侧合法**——与 4.55 结算
// (engine.UpdateBlackBoxMonthly) 同属世界侧簿记。黑箱壁 (AST 守卫锁定) 禁止的
// 是 1 号决策侧 (silo1_blackbox.go) 读真值，不禁止世界侧行为结算。
//
// 档位后果 (00 卷 §2.1 语义档)：
//   - 阳奉阴违带 Θ ∈ [0.30, θ_defy)：产生「指令拖延/复核空转」类已发生事实日志
//     (英文短日志，审计渠道)；不发故事——避免文本墙 (04 卷 §3.d)。
//   - 叫板带 Θ ≥ θ_defy：发中文故事事件 (主体是「对上层指令的抵制」，不点名
//     1 号，04 卷 §4 合规) + 该堡叛乱漂移。
//
// 跨档去重 (二选一之「月度只在 Θ 首次越档时发」)：叫板集状态 (episode) 存世界侧
// 台账 Experiment.Ledger (键 "defy_episode:<siloID>"，models.go 不加字段时的
// 通用已结算标记载体，与防重复计费标记同口径)。Θ 首次越过 θ_defy 的月份发故事
// 并置集标记；Θ 停留叫板带的后续月份只计叛乱漂移、不重复发故事；Θ 回落到
// θ_defy 之下时集结束 (标记清零)，再次越档再发。故事事件 ID 同时含堡号与年月，
// 每次越档的 ID 唯一，审计回放无歧义。
//
// 零 RNG：全部是阈值比较、确定性算术与顺序追加。

import (
	"fmt"
	"strconv"

	"silo40/internal/engine"
	"silo40/internal/model"
)

const (
	// blackBoxBehaviorEvasiveOnset 阳奉阴违带行为下界 (03 卷 §2.1：B2 阳奉阴违
	// [0.30,0.50) 与 B3 抵制 [0.50,0.70) 在行为面合并为「表面照办、底下敷衍」
	// 的已发生事实)。TODO(balance)
	blackBoxBehaviorEvasiveOnset = 0.30
	// blackBoxBehaviorDefyDriftScale / Span 叫板带叛乱漂移幅度：
	// +0.01·(Θ−θ_defy)/0.30 (Θ=1 时月漂移 +0.01)。TODO(balance)
	blackBoxBehaviorDefyDriftScale = 0.01
	blackBoxBehaviorDefyDriftSpan  = 0.30
)

// blackBoxBehaviorDefyEventIDPrefix 叫板带故事事件 ID 前缀 (完整 ID 形如
// silo1_defiance_2_122_6：堡号 + 年月)。
const blackBoxBehaviorDefyEventIDPrefix = "silo1_defiance_"

// blackBoxBehaviorDefyLedgerKey 叫板集 (episode) 台账键。
func blackBoxBehaviorDefyLedgerKey(siloID uint) string {
	return "defy_episode:" + strconv.FormatUint(uint64(siloID), 10)
}

// runBlackBoxBehaviorMonth Θ 行为化月度回合 (2b' 固定顺序第四位)。
// 门控同调度器 (功能关闭/1 号覆灭早退)；叙事只陈述已发生事实 (04 卷 §3.d)。
// 逐堡按 s.silos 序 (开局后固定 = SiloID 升序) 结算，确定性。
func (s *GameService) runBlackBoxBehaviorMonth(logs *[]string, stories *[]model.StoryEvent) {
	if !s.silo1BlackBoxOn() || s.world == nil || !s.world.Silo1.Alive {
		return
	}
	if !s.world.Silo1Stewardship.Initialized {
		return
	}
	exp := &s.world.Silo1Stewardship.Experiment
	stewardshipExperimentEnsureMaps(exp)

	year, month := s.world.CurrentYear, s.world.CurrentMonth
	for _, silo := range s.silos {
		if siloEnded(silo) || !silo.BlackBox.Initialized {
			continue
		}
		theta := silo.BlackBox.Theta
		key := blackBoxBehaviorDefyLedgerKey(silo.ID)
		switch {
		case theta >= engine.BlackBoxThetaDefy:
			s.runBlackBoxDefyMonth(silo, theta, year, month, exp, logs, stories)
		case theta >= blackBoxBehaviorEvasiveOnset:
			// 阳奉阴违带：只记已发生事实 (英文短日志)，不发故事 (避免文本墙)。
			if logs != nil {
				*logs = append(*logs, fmt.Sprintf(
					"[BlackBoxBehavior] silo %d: order receipts lag the schedule; recheck rounds spin idle", silo.ID))
			}
			exp.Ledger[key] = 0 // 自叫板带回落：集结束
		default:
			exp.Ledger[key] = 0 // 服从/摩擦带：无集
		}
	}
}

// runBlackBoxDefyMonth 叫板带行为后果：叛乱漂移每月计；故事只在 Θ 首次越档月发
// (集标记见文件头)。
func (s *GameService) runBlackBoxDefyMonth(silo *model.Silo, theta float64, year, month int, exp *model.ExperimentStewardship, logs *[]string, stories *[]model.StoryEvent) {
	// 叛乱漂移：叫板带的执行链公开抵制指令，堡内秩序压力上行。TODO(balance)
	drift := blackBoxBehaviorDefyDriftScale * (theta - engine.BlackBoxThetaDefy) / blackBoxBehaviorDefyDriftSpan
	silo.Rebellion = clampUnit01(silo.Rebellion + drift)

	key := blackBoxBehaviorDefyLedgerKey(silo.ID)
	if exp.Ledger[key] == 1 {
		return // 集进行中：不重复发故事
	}
	exp.Ledger[key] = 1

	// 世界观合规 (04 卷 §4)：主体是「上层指令」与「各部门」，不点名 1 号、
	// 不断言背后有人；只陈述已发生事实，不给概率、不点名结局。
	story := model.StoryEvent{
		ID:       blackBoxBehaviorDefyEventIDPrefix + strconv.FormatUint(uint64(silo.ID), 10) + "_" + strconv.Itoa(year) + "_" + strconv.Itoa(month),
		Category: "crisis",
		Title:    "上层指令受阻",
		Description: "本月的上层指令没有得到执行。各部门以人手不足为由互相推诿，" +
			"回执迟迟没有送出。指令被搁在案头——没有人公开拒绝，也没有人真的动手。",
		Type:     "SOCIAL",
		FireMode: engine.ContentFireModeOnce, // 跨档转折：邻堡值得经谣言渠道传出
		SiloID:   silo.ID,
		SiloName: silo.Name,
	}
	// 记录到玩家情报记录 (先例：captureSilo1Signals 的上下文切换)。
	s.currentSilo = s.silo
	s.currentAgent = s.agent
	s.recordStories([]*model.StoryEvent{&story}, "SILO1_STEWARD:DEFIANCE")
	s.currentSilo = nil
	s.currentAgent = nil
	if stories != nil {
		*stories = append(*stories, story)
	}
	if logs != nil {
		*logs = append(*logs, fmt.Sprintf(
			"[BlackBoxBehavior] silo %d: open resistance to upper directives (rebellion drift +%.4f)", silo.ID, drift))
	}
}

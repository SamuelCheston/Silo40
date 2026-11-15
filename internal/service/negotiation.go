package service

import (
	"fmt"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// 谈判/妥协层编排 (18 设计卷、ADR-0011)。
//
// 纪律：
//   - 本层只改变"谁在阻碍谁"：产出写入顶层立场的权重输入与组织事实，
//     不发放胜利、不写资格；最终结算仍只看逃出去多少人。
//   - 全部随机决策经引擎单一 RNG 流 (固定取样顺序在 engine.NegotiationRound)；
//     未开信道的运行不进入任何取样，RNG 消耗逐位不变。
//   - 失败与拖延都是合法结果：叙事明说，不伪装成进展。

// 妥协后接管的数值为初始校准值。TODO(balance)
const (
	// 接管时委员会正统性 = 基础 + 叛乱水平 + 保障可信度 (夹 [0,1])。
	negotiationTakeoverLegitimacyBase      = 0.30
	negotiationTakeoverLegitimacyRebellion = 0.40
	negotiationTakeoverLegitimacyGuarantee = 0.20
	negotiationTakeoverRebellionDelta      = -0.15 // 接管：诉求兑现，叛乱回落
	// 不接管：外溢信道可靠度增量 (夹 [0,1])。
	negotiationHandsOffChannelDelta = 0.10
)

// negotiationLeanLabel 市长倾向的玩家可感知口径 (读 MayorLean，不读立场本身)。
func negotiationLeanLabel(lean float64) string {
	switch {
	case lean < 0.3: // TODO(balance)
		return "强硬"
	case lean < 0.5: // TODO(balance)
		return "动摇"
	case lean < 0.7: // TODO(balance)
		return "倾向妥协"
	default:
		return "准备妥协"
	}
}

// negotiationDispositionLabel 市长性情的中文口径。
func negotiationDispositionLabel(disposition string) string {
	switch disposition {
	case model.MayorDispositionHardliner:
		return "强硬派"
	case model.MayorDispositionPragmatist:
		return "务实派"
	case model.MayorDispositionSurvivor:
		return "求生派"
	default:
		return "未知"
	}
}

// runNegotiationMonth 月度低强度通气：仅当信道畅通、未崩塌、妥协未成立时
// 执行一个回合。未开信道的堡不进入本函数，不消耗任何 RNG (18 §6)。
func (s *GameService) runNegotiationMonth(silo *model.Silo) ([]string, []model.StoryEvent) {
	if silo == nil || !silo.Negotiation.NegotiationActive() {
		return nil, nil
	}
	result := engine.NegotiationRound(silo, s.world, engine.NegotiationKindMonth)
	if result == nil {
		return nil, nil
	}
	stories := s.negotiationRoundStories(silo, result)
	logs := []string{fmt.Sprintf("[Negotiation] %d-%d 回合#%d：市长路径=%s%s",
		silo.CurrentYear, silo.CurrentMonth, silo.Negotiation.Rounds, result.MayorPath,
		negotiationRoundResultSuffix(result))}
	return logs, stories
}

func negotiationRoundResultSuffix(result *engine.NegotiationRoundResult) string {
	suffix := ""
	if result.SuppressBrokeChannel {
		suffix += "，信道被镇压打断"
	}
	if result.CompromiseEstablished {
		suffix += "，妥协成立"
	} else if result.RebelAccept != nil && !*result.RebelAccept {
		suffix += "，提案被叛乱方拒绝"
	}
	if result.ITExposedChannel {
		suffix += "，信道被 IT 曝光"
	}
	return suffix
}

// negotiationRoundStories 把一个回合的结构化结果转成玩家可见的中文叙事。
// 谁提了什么、谁在拖延、市长此刻的倾向都如实呈现；失败与拖延不伪装成进展。
func (s *GameService) negotiationRoundStories(silo *model.Silo, result *engine.NegotiationRoundResult) []model.StoryEvent {
	stories := []model.StoryEvent{}
	year, month := silo.CurrentYear, silo.CurrentMonth
	lean := negotiationLeanLabel(silo.Negotiation.MayorLean)
	add := func(key, title, description string) {
		stories = append(stories, model.StoryEvent{
			ID:          fmt.Sprintf("negotiation:%d:%d:%d:%s", year, month, silo.ID, key),
			Category:    "SOCIAL",
			Title:       title,
			Description: description,
			Type:        "SOCIAL",
			SiloID:      silo.ID,
			SiloName:    silo.Name,
		})
	}

	switch result.MayorPath {
	case model.NegotiationPathSuppress:
		if result.SuppressBrokeChannel {
			add("suppress_break", "谈判信道被镇压打断",
				"市长选择了镇压。执法队封锁了中间人的接头点，秘密信道被切断——这一轮谈判到此为止。镇压暂时压住了叛乱，但背信也让顶层的出路更窄了。")
		} else {
			add("suppress_hold", "市长选择镇压，信道勉强维持",
				fmt.Sprintf("市长选择了镇压：执法队带走了几名中间人，叛乱被暂时压回。信道侥幸没有断，但对方的信任受了创。市长此刻的倾向：%s。", lean))
		}
	case model.NegotiationPathTalk:
		add("talk", "市长在拖延",
			fmt.Sprintf("市长办公室的回复仍然只有『需要更多时间』。这一轮没有产生任何决议——这是拖延，不是进展。市长此刻的倾向：%s。", lean))
	case model.NegotiationPathConcede:
		if result.CompromiseEstablished {
			add("compromise", "妥协成立",
				"妥协成立：赦免名单、职能保留、不清算求生者。顶层正在重新掂量立场。接下来，是否接管市长职位，由你们决定。")
		} else if result.RebelAccept != nil && !*result.RebelAccept {
			add("offer_rejected", "叛乱方拒绝了市长的提案",
				"市长开出妥协提案，但叛乱方领导层拒绝了它——提案作废。对方的立场可能因此强硬起来。拖延与拒绝都是谈判的合法结果。")
		} else {
			add("offer_made", "市长开出妥协提案",
				fmt.Sprintf("市长示意愿意让步：赦免名单、职能保留、不清算求生者被摆上了桌面。叛乱方领导层正在权衡。市长此刻的倾向：%s。", lean))
		}
	}

	if result.RebelPosture != "" {
		switch result.RebelPosture {
		case model.NegotiationPosturePress:
			add("rebels_press", "叛乱方施压",
				"叛乱方没有放松压力：更多的部门加入了不合作的行列。")
		case model.NegotiationPostureProbe:
			add("rebels_probe", "叛乱方试探市长倾向",
				fmt.Sprintf("中间人又递了一轮话，试探市长办公室的口风。市长此刻的倾向：%s。", negotiationLeanLabel(silo.Negotiation.MayorLean)))
		case model.NegotiationPostureOffer:
			add("rebels_offer", "叛乱方酝酿递出条款",
				"叛乱方领导层认为时机接近成熟，正在酝酿一份正式的妥协条款——也许该把它递上桌面了。")
		}
	}

	switch result.ITReaction {
	case model.NegotiationITUndermine:
		if result.ITExposedChannel {
			add("it_exposure", "谈判信道被 IT 曝光",
				"IT 把秘密接头泄给了监控网络——信道崩塌，全堡哗然。谈判结束了，但恐慌与叛乱都在上升。")
		} else {
			add("it_undermine", "IT 在暗中破坏谈判",
				"IT 在暗中做手脚：中间人被盯上，承诺的可信度受损。信道还在，但保障的分量轻了。")
		}
	}

	return stories
}

// applyNegotiationOpen 开信道 (内容效果 negotiation_open 的服务侧落点)。
func (s *GameService) applyNegotiationOpen(silo *model.Silo) []model.StoryEvent {
	if silo == nil {
		return nil
	}
	result := engine.OpenNegotiationChannel(silo)
	if result == nil {
		return nil
	}
	story := model.StoryEvent{
		ID:       fmt.Sprintf("negotiation:%d:%d:%d:open", silo.CurrentYear, silo.CurrentMonth, silo.ID),
		Category: "SOCIAL",
		Type:     "SOCIAL",
		SiloID:   silo.ID,
		SiloName: silo.Name,
	}
	if result.AlreadyOpen {
		if silo.Negotiation.Closed {
			story.Title = "信道已经崩塌"
			story.Description = "这条谈判信道已经崩塌，无法重新开通。"
		} else {
			story.Title = "信道已经建立"
			story.Description = "谈判信道已经建立，不必重复开通。"
		}
		return []model.StoryEvent{story}
	}
	story.Title = "谈判信道建立"
	story.Description = fmt.Sprintf("中间人搭上了市长办公室的秘密线路。传回来的口风是：%s。谈判是持续的过程——可以试探、可以施压、可以递条款；市长可能一直拖延，也可能在最后一刻妥协。",
		negotiationDispositionLabel(result.Disposition))
	return []model.StoryEvent{story}
}

// applyNegotiationResolve 妥协后二选一 (内容效果 negotiation_resolve 的服务侧落点)。
// 接管与不接管都只改变"谁在阻碍谁"与组织事实，不发放胜利。
func (s *GameService) applyNegotiationResolve(silo *model.Silo, kind string) []model.StoryEvent {
	if silo == nil {
		return nil
	}
	negotiation := &silo.Negotiation
	story := model.StoryEvent{
		ID:       fmt.Sprintf("negotiation:%d:%d:%d:resolve:%s", silo.CurrentYear, silo.CurrentMonth, silo.ID, kind),
		Category: "SOCIAL",
		Type:     "SOCIAL",
		SiloID:   silo.ID,
		SiloName: silo.Name,
	}
	if !negotiation.CompromiseEstablished {
		story.Title = "尚无妥协"
		story.Description = "妥协尚未成立，这一步还无从谈起。"
		return []model.StoryEvent{story}
	}
	if negotiation.Takeover != "" {
		story.Title = "抉择已定"
		story.Description = "妥协之后的抉择已经做出，不能反悔。"
		return []model.StoryEvent{story}
	}

	switch kind {
	case model.NegotiationTakeover:
		negotiation.Takeover = model.NegotiationTakeover
		legitimacy := clampUnit(negotiationTakeoverLegitimacyBase +
			negotiationTakeoverLegitimacyRebellion*silo.Rebellion +
			negotiationTakeoverLegitimacyGuarantee*silo.TopStance.Guarantee) // TODO(balance)
		silo.SetMayorLegitimacy(legitimacy)
		silo.Rebellion = clampUnit(silo.Rebellion + negotiationTakeoverRebellionDelta) // TODO(balance)
		engine.ReevalTopStanceAfterTakeover(silo, s.world)
		negotiation.AppendLogEntry(model.NegotiationLogEntry{
			Year: silo.CurrentYear, Month: silo.CurrentMonth, Decider: "rebels", Decision: model.NegotiationTakeover,
			Factors: engine.NegotiationFactorSummary(silo),
			Outcome: "接管市长职位，顶层改由叛乱方主导",
		})
		story.Title = "叛乱方接管市长职位"
		story.Description = "委员会接管了市长办公室——顶层改由叛乱方主导，旧市长退居幕后。对离开的立场正在重新估量。这不意味着胜利：能带走多少人，仍由出发结算说了算。"
	case model.NegotiationHandsOff:
		negotiation.Takeover = model.NegotiationHandsOff
		for i := range silo.SpilloverPrep.Channels {
			silo.SpilloverPrep.Channels[i].Reliability = clampUnit(silo.SpilloverPrep.Channels[i].Reliability + negotiationHandsOffChannelDelta) // TODO(balance)
		}
		negotiation.AppendLogEntry(model.NegotiationLogEntry{
			Year: silo.CurrentYear, Month: silo.CurrentMonth, Decider: "rebels", Decision: model.NegotiationHandsOff,
			Factors: engine.NegotiationFactorSummary(silo),
			Outcome: "不接管市长职位，带组织能力转向离开",
		})
		story.Title = "不接管，带组织离开"
		story.Description = "叛乱方选择不接管市长职位：组织完整保留，方向从对抗转向离开。出发动员获得组织加成，外溢信道的可靠度也上升了一点。市长留任，但不再阻碍。"
	default:
		story.Title = "未知抉择"
		story.Description = "这不是一个可走的抉择。"
	}
	return []model.StoryEvent{story}
}

package service

// 见证者成就树 (ADR-0004)：国策树即成就系统。
//
// 设计口径 (the Witness / ADR-0004)：
//  1. 点亮节点的是可核验的成就——驻地里真实发生、别人也能确认的事实，
//     不是玩家按钮或意愿。因此本树完全由既有事实派生：内容事件完成键、
//     结局资格事实 (与仲裁 P0/P1 读同一组)、跨堡投递日志。不另造计数器，
//     不另造并行判定，不消耗 RNG。
//  2. 互斥与锁定：锁的条件 = 对应结局资格的否定，树与仲裁读同一组事实，
//     所以不会出现"树锁了、仲裁却还能给这个结局"。互斥点选项 (覆灭前夕
//     三选一、开窗方式二选一) 一旦点亮其一，其余同局永久锁定；被锁节点
//     保留在树上标记"被事实锁定"。锁全部由单调事实推出 (完成键、宿主覆灭、
//     覆灭前夕快照)，因此不需要任何新的持久化状态——同局天然永久，新开
//     一局事实重置、树随之重置。
//  3. 只做行动指南，不做奖励：节点不给资源、不给加成，只报"还差什么"。
//  4. 树不裁决结局：树锁分支，仲裁给结局。

import (
	"fmt"

	"silo40/internal/model"
)

// witnessTreeShadow 树的影子：判据四 (the Witness §树的影子)。不是节点，
// 是所有节点的永久前提；失败结局不上树，只作影子被提及 (征兆由
// LinesProgress.Omens 呈现，不在树上重复)。
const witnessTreeShadow = "树的影子（判据四，永久前提）：出现下列任何一项，已点亮的节点全部失效——" +
	"建立新的总部、继任体系或跨堡指挥链；以复仇为名制造居民死亡；向居民许诺外面安全；" +
	"辨认、试探或招募流散后代；让真相只能依附于你继续存在。你可以被忘记，目标必须在你缺席时继续成立。"

// witnessHostFallen 宿主 (玩家驻地) 是否已经倒下。读仲裁 P0 的同一组
// 单调事实键；按 P0 顺序 (终止/打击 → 重置 → 断层) 取首个成立者作为
// 锁定说明。宿主倒下时，一切宿主类节点在构造上失去结局资格 (P0 先于 P1)。
func (s *GameService) witnessHostFallen(fact func(string) bool) (bool, string) {
	switch {
	case fact(terminationFactKey):
		return true, "远程终止已经执行"
	case fact(strikeFactKey):
		return true, "物理打击已经落到驻地"
	case fact(resetFactKey):
		return true, "重置已经完成，记忆不在了"
	case fact(fractureFactKey):
		return true, "维存崩溃已成不可逆事实"
	}
	return false, ""
}

// witnessTrunkBranch 主干：四条完成判据的前三条。判据四是影子，不是节点。
func (s *GameService) witnessTrunkBranch(fact func(string) bool) model.WitnessBranch {
	station := s.silo
	independent := station.Authority.Judicial.IndependentPublicFacts

	t1 := model.WitnessNode{
		ID:    "trunk_t1",
		Title: "事实脱离自述",
		Guide: "先求互证，再求扩散。",
	}
	if independent >= 2 {
		t1.Lit = true
		t1.Note = fmt.Sprintf("登记簿上已有 %d 项合格独立公共事实，来源能够互相印证。", independent)
	} else {
		t1.Missing = []string{fmt.Sprintf("合格独立公共事实 %d/2——登记簿里能互相印证的独立来源还不足两类", independent)}
	}

	t2 := model.WitnessNode{
		ID:    "trunk_t2",
		Title: "真相已有本地承接者",
		Guide: "找在意程序与证据的人，不找崇拜者。",
	}
	if fact("player_actions:receiver_verify_accompany") {
		t2.Lit = true
		t2.Note = "第一位接收者独立完成核验——一成就点两节点，同时推进《共证》的感染。"
	} else {
		t2.Missing = []string{"尚无接收者独立完成核验（陪同回查是起点）"}
	}

	t3 := model.WitnessNode{
		ID:    "trunk_t3",
		Title: "终止链失效",
		Guide: "链的消失要可核验；渠道反制成立时，记下它免疫的是哪一条渠道。",
	}
	switch {
	case fact("player_action:reset_countermeasure_done"):
		t3.Lit = true
		t3.Note = "重置链反制成立：终止命令的副本分散在多人手中，单方命令不再好使。"
	case fact("special:termination_channel_disrupted"):
		t3.Lit = true
		t3.Note = "远程终止信道被干扰：本地应答散成杂讯。只免疫这一条远程渠道，挡不住物理手段。"
	default:
		t3.Missing = []string{"终止链仍在正常运转——渠道反制（值夜检修、副本分发托管）尚未成立"}
	}

	return model.WitnessBranch{
		Branch: "trunk",
		Title:  "主干 · 完成判据",
		Note:   "判据四是树的影子，不是节点：见树影。",
		Nodes:  []model.WitnessNode{t1, t2, t3},
	}
}

// witnessInfoBranch 分支一《共证》：让链活下来。感染至公共事实为宿主类
// 节点 (驻地是链的宿主)；外溢是链离开宿主的方向，宿主倒下前已送达的事实
// 不因宿主覆灭而失效。
func (s *GameService) witnessInfoBranch(fact func(string) bool, hostFallen bool, hostNote string) model.WitnessBranch {
	station := s.silo
	hostLock := func(node *model.WitnessNode) {
		if hostFallen && !node.Lit {
			node.Locked = true
			node.LockNote = "宿主已倒下：" + hostNote
			node.Missing = nil
		}
	}

	infect := model.WitnessNode{
		ID:    "info_infect",
		Title: "感染",
	}
	if fact("player_actions:receiver_verify_accompany") {
		infect.Lit = true
		infect.Note = "第一位接收者独立完成核验。"
	} else {
		infect.Missing = []string{"尚无接收者独立完成核验"}
	}
	hostLock(&infect)

	replicate := model.WitnessNode{
		ID:    "info_replicate",
		Title: "复制",
	}
	if fact("special:replication_progress") {
		replicate.Lit = true
		replicate.Count = s.contentStates[station.ID]["special:replication_progress"].TriggerCount
		replicate.Note = "接收者培养出下一位接收者。"
	} else {
		replicate.Missing = []string{"链尚未长出下一节：分享被接收才算复制"}
	}
	hostLock(&replicate)

	living := model.WitnessNode{
		ID:    "info_living",
		Title: "活链",
	}
	if fact("special:living_chain_confirmed") {
		living.Lit = true
		living.Note = "链不需要你逐日维持也能复制。"
	} else {
		living.Missing = []string{"链仍需要你逐日维持"}
	}
	hostLock(&living)

	public := model.WitnessNode{
		ID:    "info_public",
		Title: "公共事实",
	}
	if fact("special:public_fact_cross_dept") {
		public.Lit = true
		public.Note = "核心真相成为跨部门、可独立复核的公共事实——《共证》资格的最终投影。"
	} else {
		public.Missing = []string{"真相尚未站到驻地中央（跨部门公共事实未成立）"}
	}
	hostLock(&public)

	spillover := model.WitnessNode{
		ID:    "info_spillover",
		Title: "外溢",
		Guide: "逃跑的条件没达成，就回退为外溢：进展改道流入把下一环送出驻地的方向。",
	}
	received := 0
	if s.world != nil {
		for _, signal := range s.world.OutboundSignals {
			if signal.Spillover && signal.Received && signal.RootSilo == station.ID {
				received++
			}
		}
	}
	if received > 0 {
		spillover.Lit = true
		spillover.Count = received
		spillover.Note = fmt.Sprintf("包裹已到达并被独立接收 %d 次。评分项，不是资格：外溢为零同样可以判胜。", received)
	} else if hostFallen {
		spillover.Locked = true
		spillover.LockNote = "宿主已倒下：" + hostNote + "——再无包裹可以发出"
	} else {
		spillover.Missing = []string{"尚无包裹到达并被独立接收（信道、封装、投递三步都还是空的）"}
	}

	return model.WitnessBranch{
		Branch: "info",
		Title:  "分支一 ·《共证》：让链活下来",
		Nodes:  []model.WitnessNode{infect, replicate, living, public, spillover},
	}
}

// witnessRebellionBranch 分支二《越岭》：让人离开。结算只看逃出去了多少人
// (裁决 F)：立场只塑形输入，不是资格；树上也只读逃出群体事实。
func (s *GameService) witnessRebellionBranch(fact func(string) bool, hostFallen bool, hostNote string) model.WitnessBranch {
	station := s.silo

	will := model.WitnessNode{
		ID:    "reb_will",
		Title: "意愿",
	}
	if fact("special:departure_willingness") {
		will.Lit = true
		will.Note = "想走的人出现且可联络——离开意愿不是队伍。"
	} else {
		will.Missing = []string{"想走的人尚未成形"}
	}

	route := model.WitnessNode{
		ID:    "reb_route",
		Title: "路线",
	}
	if fact("special:route_confirmed") {
		route.Lit = true
		route.Note = "先遣往返确认通行——路线是事实，不是计划。"
	} else {
		route.Missing = []string{"尚无先遣往返确认的通行事实"}
	}

	window := model.WitnessNode{
		ID:    "reb_window",
		Title: "开窗",
		Guide: "无路径的叛乱锁死本节点：必须有已确认的路线事实与组织网络，叛乱水平达到开窗阈值。",
	}
	if fact("special:escape_window_open") {
		window.Lit = true
		window.Note = "有组织反抗打开了撤离窗口。"
	} else {
		if !fact("special:route_confirmed") {
			window.Missing = append(window.Missing, "路线未确认")
		}
		if !fact("special:underground_coordination") {
			window.Missing = append(window.Missing, "地下协调网络未成立")
		}
		window.Missing = append(window.Missing, "叛乱水平须达开窗阈值")
	}

	method := model.WitnessNode{
		ID:    "reb_window_method",
		Title: "开窗方式",
	}
	orderly := fact("player_actions:escape_window_orderly")
	riot := fact("player_actions:escape_window_riot")
	method.Choices = []model.WitnessChoice{
		{ID: "orderly", Title: "秩序开窗", Lit: orderly, Locked: riot},
		{ID: "riot", Title: "暴乱开窗", Lit: riot, Locked: orderly},
	}
	if orderly {
		method.Lit = true
		method.Note = "撤离窗口按秩序维持——存活率走秩序档。"
	} else if riot {
		method.Lit = true
		method.Note = "撤离窗口在暴乱中打开——存活率走暴乱档。"
	} else {
		method.Missing = []string{"窗口打开后须二选一：秩序或暴乱，点亮其一另一个同局永久锁定"}
	}
	if hostFallen && !method.Lit {
		method.Locked = true
		method.LockNote = "宿主已倒下：" + hostNote
		method.Missing = nil
	}

	depart := model.WitnessNode{
		ID:    "reb_depart",
		Title: "离开",
		Guide: "结算只看逃出去了多少人：市长与 IT 是否配合不是资格，只塑形输入。",
	}
	group := station.EscapedGroup
	if group.SurvivingEscapeGroup() {
		depart.Lit = true
		depart.Note = fmt.Sprintf("成规模群体实际离开并存续：%d 人出发，%d 人存续。", group.Departed, group.Population)
	} else if hostFallen {
		depart.Locked = true
		depart.LockNote = "撤离窗口已关闭：宿主先于离开倒下（" + hostNote + "）——逃离方向未消费的进展回退为外溢方向"
		depart.Missing = nil
	} else {
		depart.Missing = []string{"成规模群体尚未实际离开并存续"}
	}

	note := ""
	if hostFallen && group.SurvivingEscapeGroup() {
		note = "宿主已倒下而逃出群体存续：原驻地的毁灭只作惨胜尾声。"
	}
	return model.WitnessBranch{
		Branch: "rebellion",
		Title:  "分支二 ·《越岭》：让人离开",
		Note:   note,
		Nodes:  []model.WitnessNode{will, route, window, method, depart},
	}
}

// witnessTimeBranch 分支三《无人裁决》：让开放活到结算。互斥点一在覆灭
// 前夕抉择里：三选一，点亮其一其余同局永久锁定；思潮闸的锁写在结局判定
// 里 (纯种地堡思潮绝对排除该结局)，树只是把它显形。
func (s *GameService) witnessTimeBranch(fact func(string) bool, hostFallen bool, hostNote string) model.WitnessBranch {
	station := s.silo

	sustain := model.WitnessNode{
		ID:       "time_sustain",
		Title:    "维持",
		Reserved: true,
		Note:     "驻地在危机中存续是状态而非事件——暂无对应事实键，节点预留，不假装可点亮。",
	}

	open := model.WitnessNode{
		ID:    "time_open",
		Title: "开放",
	}
	if fact("special:open_discourse_milestone") {
		open.Lit = true
		open.Count = s.contentStates[station.ID]["special:open_discourse_milestone"].TriggerCount
		open.Note = "开放进入普通 Cohort 的公共表达；里程碑可重复点亮。"
	} else {
		open.Missing = []string{"公开讨论尚未成为普通人的公共表达"}
	}

	choice := model.WitnessNode{
		ID:    "time_choice",
		Title: "覆灭前夕抉择",
	}
	loyalty := fact("player_actions:pre_fall_loyalty_mobilization")
	totalRiot := fact("player_actions:pre_fall_total_riot")
	inquiry := fact("player_actions:pre_fall_open_inquiry")
	anyChosen := loyalty || totalRiot || inquiry
	choice.Choices = []model.WitnessChoice{
		{ID: "loyalty", Title: "忠诚动员", Lit: loyalty, Locked: anyChosen && !loyalty},
		{ID: "total_riot", Title: "全面暴乱", Lit: totalRiot, Locked: anyChosen && !totalRiot},
		{ID: "inquiry", Title: "维持质询", Lit: inquiry, Locked: anyChosen && !inquiry},
	}
	switch {
	case loyalty:
		choice.Lit = true
		choice.Note = "以全面忠诚动员换取秩序：思潮正在凝成纯种地堡的形状。"
	case totalRiot:
		choice.Lit = true
		choice.Note = "全面暴乱：倾向《断层》的影子。"
	case inquiry:
		choice.Lit = true
		choice.Note = "维持最低秩序，公开质询照常——最难的一条路。"
	case fact("special:pre_fall_choice"):
		choice.Note = "覆灭前夕已到：驻地面对同一个问题，先听谁的。三选一，点亮其一其余永久锁定。"
		choice.Missing = []string{"抉择时刻已到，尚未作出选择"}
	default:
		choice.Missing = []string{"覆灭前夕尚未到来"}
	}

	branchNote := ""
	snap := station.PreFallSnapshot
	if snap.Taken {
		if snap.OpenQualified {
			branchNote = "覆灭前夕思潮闸通过：硬条件已在覆灭瞬间定格。"
		} else {
			branchNote = "覆灭前夕思潮闸未通过（纯种地堡思潮主导）：《无人裁决》被事实永久锁定——这把锁写在结局判定里，树只是把它显形。"
			// 锁成立时未亮的选项一并标记锁定：资格在构造上已不可能。
			if !choice.Lit {
				choice.Locked = true
				choice.LockNote = "思潮闸未通过：硬条件已在覆灭瞬间定格"
				choice.Missing = nil
			}
			for i := range choice.Choices {
				if !choice.Choices[i].Lit {
					choice.Choices[i].Locked = true
				}
			}
		}
	} else if hostFallen {
		// 宿主覆灭而快照未取 (非 1 号覆灭路径)：整支失去前提。
		branchNote = "宿主已倒下：" + hostNote
		for i := range choice.Choices {
			if !choice.Choices[i].Lit {
				choice.Choices[i].Locked = true
			}
		}
		if !choice.Lit {
			choice.Locked = true
			choice.LockNote = "宿主已倒下：" + hostNote
			choice.Missing = nil
		}
		if !open.Lit {
			open.Locked = true
			open.LockNote = "宿主已倒下：" + hostNote
			open.Missing = nil
		}
	}

	return model.WitnessBranch{
		Branch: "time",
		Title:  "分支三 ·《无人裁决》：让开放活到结算",
		Note:   branchNote,
		Nodes:  []model.WitnessNode{sustain, open, choice},
	}
}

// witnessConvergenceBranch 收敛层：两种中性秩序。软互斥——不设二选一的
// 硬锁，两者可部分并存，同分按序让《公议》 (既有裁决)。
func (s *GameService) witnessConvergenceBranch(fact func(string) bool) model.WitnessBranch {
	station := s.silo

	pact := model.WitnessNode{
		ID:    "conv_pact",
		Title: "《公议》约章锁定",
	}
	if station.Authority.Judicial.PactLocked {
		pact.Lit = true
		pact.Note = "公开约章已锁定：Judicial 附着约章，不可被单方收回。"
	} else {
		for _, key := range []string{
			"special:order_chain_breach",
			"player_actions:charter_contact_accept",
			"special:charter_draft_convened",
			"special:charter_lock_day",
		} {
			if !fact(key) {
				pact.Missing = append(pact.Missing, s.lineMilestoneTitle(key))
			}
		}
		if len(pact.Missing) > 0 {
			pact.Missing = []string{"约章之路还差：" + joinChinese(pact.Missing)}
		}
	}

	personal := model.WitnessNode{
		ID:    "conv_personal",
		Title: "《人治》警察服从＋正统性",
	}
	if eligible, reason := personalRuleEligible(station); eligible {
		personal.Lit = true
		personal.Note = "警察服从＋市长个人正统性成立，司法部不在 IT 手里。"
	} else {
		personal.Missing = []string{reason}
	}

	return model.WitnessBranch{
		Branch: "convergence",
		Title:  "收敛层 · 两种中性秩序",
		Note:   "软互斥：两者可部分并存，不设二选一的硬锁；同分按序让《公议》。",
		Nodes:  []model.WitnessNode{pact, personal},
	}
}

// joinChinese 用顿号连接条目 (确定性，不引入随机源)。
func joinChinese(items []string) string {
	out := ""
	for i, item := range items {
		if i > 0 {
			out += "、"
		}
		out += item
	}
	return out
}

// witnessTree 汇总见证者成就树快照。只读事实，不消耗 RNG，不写任何状态；
// 同一组事实必然得到同一棵树 (确定性)。
func (s *GameService) witnessTree() *model.WitnessTree {
	if s.silo == nil {
		return nil
	}
	states := s.contentStates[s.silo.ID]
	fact := func(key string) bool { return states[key].Triggered }
	hostFallen, hostNote := s.witnessHostFallen(fact)

	return &model.WitnessTree{
		Branches: []model.WitnessBranch{
			s.witnessTrunkBranch(fact),
			s.witnessInfoBranch(fact, hostFallen, hostNote),
			s.witnessRebellionBranch(fact, hostFallen, hostNote),
			s.witnessTimeBranch(fact, hostFallen, hostNote),
			s.witnessConvergenceBranch(fact),
		},
		Shadow: witnessTreeShadow,
	}
}

package service

import (
	"reflect"
	"strings"
	"testing"

	"silo40/internal/model"
)

// witnessNode 取出树上指定 ID 的节点；缺失即测试失败。
func witnessNode(t *testing.T, tree *model.WitnessTree, id string) *model.WitnessNode {
	t.Helper()
	for i := range tree.Branches {
		for j := range tree.Branches[i].Nodes {
			node := &tree.Branches[i].Nodes[j]
			if node.ID == id {
				return node
			}
		}
	}
	t.Fatalf("node %q missing from tree: %+v", id, tree)
	return nil
}

func witnessBranch(t *testing.T, tree *model.WitnessTree, branch string) *model.WitnessBranch {
	t.Helper()
	for i := range tree.Branches {
		if tree.Branches[i].Branch == branch {
			return &tree.Branches[i]
		}
	}
	t.Fatalf("branch %q missing from tree", branch)
	return nil
}

// TestWitnessTreeFreshGame 锁定开局形态：无一节点点亮、无锁定，主干报缺，
// 无事实键的「维持」只作预留节点存在——不假装可点亮。
func TestWitnessTreeFreshGame(t *testing.T) {
	svc := linesFixture()
	tree := svc.witnessTree()
	if tree == nil {
		t.Fatal("witnessTree returned nil for a live session")
	}
	for i := range tree.Branches {
		for _, node := range tree.Branches[i].Nodes {
			if node.Lit {
				t.Fatalf("fresh game must not light %q", node.ID)
			}
			if node.Locked {
				t.Fatalf("fresh game must not lock %q: %s", node.ID, node.LockNote)
			}
		}
	}
	sustain := witnessNode(t, tree, "time_sustain")
	if !sustain.Reserved {
		t.Fatalf("维持 has no fact key and must stay reserved: %+v", sustain)
	}
	if tree.Shadow == "" {
		t.Fatal("tree must carry the criterion-four shadow")
	}
	t1 := witnessNode(t, tree, "trunk_t1")
	if len(t1.Missing) == 0 || !strings.Contains(t1.Missing[0], "0/2") {
		t.Fatalf("T1 must report the missing independent-fact count: %+v", t1)
	}
}

// TestWitnessTreeTrunkFacts 锁定主干三判据的点亮来源：登记簿合格独立事实、
// 第一位接收者独立核验、终止链渠道反制 (并报免疫的是哪条渠道)。
func TestWitnessTreeTrunkFacts(t *testing.T) {
	svc := linesFixture()
	states := svc.statesMapFor(svc.silo.ID)

	svc.silo.Authority.Judicial.IndependentPublicFacts = 2
	states["player_actions:receiver_verify_accompany"] = litState(122, 4)
	tree := svc.witnessTree()
	if !witnessNode(t, tree, "trunk_t1").Lit {
		t.Fatal("two qualifying independent public facts must light T1")
	}
	t2 := witnessNode(t, tree, "trunk_t2")
	if !t2.Lit || !strings.Contains(t2.Note, "一成就点两节点") {
		t.Fatalf("T2 must light via the first receiver and say so: %+v", t2)
	}
	if witnessNode(t, tree, "trunk_t3").Lit {
		t.Fatal("T3 must not light before a channel countermeasure")
	}

	states["special:termination_channel_disrupted"] = litState(122, 5)
	tree = svc.witnessTree()
	t3 := witnessNode(t, tree, "trunk_t3")
	if !t3.Lit || !strings.Contains(t3.Note, "远程终止信道") {
		t.Fatalf("T3 must light via the jammed channel and name it: %+v", t3)
	}

	// 重置链反制 (副本分发托管) 是另一条独立的点亮来源。
	svc2 := linesFixture()
	svc2.statesMapFor(svc2.silo.ID)["player_action:reset_countermeasure_done"] = litState(123, 2)
	t3b := witnessNode(t, svc2.witnessTree(), "trunk_t3")
	if !t3b.Lit || !strings.Contains(t3b.Note, "副本分散") {
		t.Fatalf("T3 must light via distributed custody and name the channel: %+v", t3b)
	}
}

// TestWitnessTreeExclusionPointOne 锁定互斥点一：覆灭前夕三选一，点亮其一
// 其余同局永久锁定；被锁选项保留在节点上可见。
func TestWitnessTreeExclusionPointOne(t *testing.T) {
	svc := linesFixture()
	states := svc.statesMapFor(svc.silo.ID)
	states["special:pre_fall_choice"] = litState(129, 1)
	states["player_actions:pre_fall_loyalty_mobilization"] = litState(129, 2)

	choice := witnessNode(t, svc.witnessTree(), "time_choice")
	if !choice.Lit {
		t.Fatalf("choice moment with a taken side must be lit: %+v", choice)
	}
	var loyalty, inquiry, riot *model.WitnessChoice
	for i := range choice.Choices {
		switch choice.Choices[i].ID {
		case "loyalty":
			loyalty = &choice.Choices[i]
		case "inquiry":
			inquiry = &choice.Choices[i]
		case "total_riot":
			riot = &choice.Choices[i]
		}
	}
	if loyalty == nil || inquiry == nil || riot == nil {
		t.Fatalf("choice node must carry all three sides: %+v", choice)
	}
	if !loyalty.Lit || loyalty.Locked {
		t.Fatalf("loyalty mobilization must be lit and unlocked: %+v", loyalty)
	}
	if !inquiry.Locked || !riot.Locked {
		t.Fatalf("the unchosen sides must be locked by fact: %+v / %+v", inquiry, riot)
	}

	// 反向：维持质询点亮时，忠诚动员与全面暴乱被锁。
	svc2 := linesFixture()
	states2 := svc2.statesMapFor(svc2.silo.ID)
	states2["special:pre_fall_choice"] = litState(129, 1)
	states2["player_actions:pre_fall_open_inquiry"] = litState(129, 2)
	choice2 := witnessNode(t, svc2.witnessTree(), "time_choice")
	for _, side := range choice2.Choices {
		switch side.ID {
		case "inquiry":
			if !side.Lit || side.Locked {
				t.Fatalf("inquiry must be lit and unlocked: %+v", side)
			}
		default:
			if !side.Locked {
				t.Fatalf("%s must be locked once inquiry is chosen: %+v", side.ID, side)
			}
		}
	}
}

// TestWitnessTreeTimeGateLock 锁定《无人裁决》的分支锁：思潮闸未通过 = 资格
// 否定，树把它显形为永久锁定；锁写在结局判定里，树与仲裁读同一组事实。
func TestWitnessTreeTimeGateLock(t *testing.T) {
	svc := linesFixture()
	svc.silo.Silo1Destroyed = true
	svc.silo.PreFallSnapshot = model.PreFallIdeologySnapshot{
		Taken: true, OpenQualified: false, DominantAxis: model.IdeologyLoyalty,
	}

	tree := svc.witnessTree()
	branch := witnessBranch(t, tree, "time")
	if branch.Note == "" || !strings.Contains(branch.Note, "永久锁定") {
		t.Fatalf("failed ideology gate must read as a permanent branch lock: %+v", branch)
	}
	choice := witnessNode(t, tree, "time_choice")
	if !choice.Locked || choice.LockNote == "" {
		t.Fatalf("unchosen moment under a failed gate must be locked: %+v", choice)
	}
	for _, side := range choice.Choices {
		if !side.Locked {
			t.Fatalf("gate failed: side %s must be locked: %+v", side.ID, side)
		}
	}

	// 思潮闸通过时不得出现分支锁。
	svc.silo.PreFallSnapshot.OpenQualified = true
	branch = witnessBranch(t, svc.witnessTree(), "time")
	if strings.Contains(branch.Note, "永久锁定") {
		t.Fatalf("passed gate must not lock the branch: %+v", branch)
	}
}

// TestWitnessTreeHostFallLocks 锁定互斥点二的宿主侧：宿主倒下时宿主类节点
// 被事实锁定；离开未成立则撤离窗口关闭、进展回退外溢；逃出群体存续则离开
// 点亮、原驻地毁灭只作惨胜尾声。
func TestWitnessTreeHostFallLocks(t *testing.T) {
	svc := linesFixture()
	states := svc.statesMapFor(svc.silo.ID)
	states["special:reset_completed"] = litState(128, 6)

	tree := svc.witnessTree()
	for _, id := range []string{"info_infect", "info_replicate", "info_living", "info_public", "time_open", "time_choice"} {
		node := witnessNode(t, tree, id)
		if !node.Locked || node.LockNote == "" {
			t.Fatalf("host fall must lock host-type node %q with the reason: %+v", id, node)
		}
	}
	depart := witnessNode(t, tree, "reb_depart")
	if !depart.Locked || !strings.Contains(depart.LockNote, "回退为外溢") {
		t.Fatalf("window closed without departure must lock 离开 with the fallback: %+v", depart)
	}
	// 已点亮的宿主类成就不被追溯熄灭：事实在先，覆灭在后。
	states["special:living_chain_confirmed"] = litState(125, 3)
	living := witnessNode(t, svc.witnessTree(), "info_living")
	if !living.Lit || living.Locked {
		t.Fatalf("an achievement that happened before the fall stays lit: %+v", living)
	}

	// 逃出群体存续：离开点亮，宿主覆灭只作惨胜尾声。
	svc.silo.EscapedGroup = model.EscapeGroup{
		Initialized: true, Surviving: true, Population: 12, Departed: 20,
	}
	tree = svc.witnessTree()
	depart = witnessNode(t, tree, "reb_depart")
	if !depart.Lit || depart.Locked {
		t.Fatalf("surviving group must light 离开 even after the host falls: %+v", depart)
	}
	if !strings.Contains(depart.Note, "20 人出发") || !strings.Contains(depart.Note, "12 人存续") {
		t.Fatalf("settlement only counts escapees: %+v", depart)
	}
	if note := witnessBranch(t, tree, "rebellion").Note; !strings.Contains(note, "惨胜尾声") {
		t.Fatalf("host fall with a surviving group reads as a pyrrhic epilogue: %q", note)
	}
}

// TestWitnessTreeDepartureTracksQualification 锁定离开节点与仲裁同一口径：
// 只读逃出群体事实 (结算只看逃出去多少人)，群体覆灭则资格丧失、节点熄灭。
func TestWitnessTreeDepartureTracksQualification(t *testing.T) {
	svc := linesFixture()
	group := &svc.silo.EscapedGroup
	group.Initialized = true
	group.Departed = 30
	group.Population = 18
	group.Surviving = true
	if node := witnessNode(t, svc.witnessTree(), "reb_depart"); !node.Lit {
		t.Fatalf("surviving escape group must light 离开: %+v", node)
	}
	group.Surviving = false
	if node := witnessNode(t, svc.witnessTree(), "reb_depart"); node.Lit || node.Locked {
		t.Fatalf("a perished group loses the qualification — the node goes dark, not locked: %+v", node)
	}
}

// TestWitnessTreeWindowMethodExclusion 锁定开窗方式二选一：点亮秩序则暴乱
// 永久锁定，反之亦然。
func TestWitnessTreeWindowMethodExclusion(t *testing.T) {
	svc := linesFixture()
	states := svc.statesMapFor(svc.silo.ID)
	states["special:escape_window_open"] = litState(126, 2)
	states["player_actions:escape_window_orderly"] = litState(126, 3)

	method := witnessNode(t, svc.witnessTree(), "reb_window_method")
	if !method.Lit {
		t.Fatalf("window method must be lit: %+v", method)
	}
	for _, side := range method.Choices {
		switch side.ID {
		case "orderly":
			if !side.Lit || side.Locked {
				t.Fatalf("orderly must be lit and unlocked: %+v", side)
			}
		case "riot":
			if !side.Locked {
				t.Fatalf("riot must be locked once orderly is chosen: %+v", side)
			}
		}
	}
}

// TestWitnessTreeSpilloverLights 锁定外溢节点只读投递日志里「到达且独立
// 接收」的事实：通用信号与未接收的包裹不计。
func TestWitnessTreeSpilloverLights(t *testing.T) {
	svc := linesFixture()
	svc.world.OutboundSignals = []model.OutboundSignal{
		{Seq: 1, SourceSilo: 1, TargetSilo: 2, Spillover: true, Received: true, RootSilo: 1, Generation: 1},
		{Seq: 2, SourceSilo: 1, TargetSilo: 3, Spillover: true, Received: false, RootSilo: 1, Generation: 1},
		{Seq: 3, SourceSilo: 1, TargetSilo: 2, Spillover: false, Received: true},
		{Seq: 4, SourceSilo: 2, TargetSilo: 3, Spillover: true, Received: true, RootSilo: 2, Generation: 2},
	}
	node := witnessNode(t, svc.witnessTree(), "info_spillover")
	if !node.Lit || node.Count != 1 {
		t.Fatalf("only the arrived-and-independently-received first-generation package counts: %+v", node)
	}
	if !strings.Contains(node.Note, "不是资格") {
		t.Fatalf("spillover must stay a score item, not qualification: %+v", node)
	}
}

// TestWitnessTreeConvergence 锁定收敛层读资格事实：约章锁定与《人治》成立
// 各自点亮，软互斥不设硬锁。
func TestWitnessTreeConvergence(t *testing.T) {
	svc := linesFixture()
	station := svc.silo
	station.Authority.Judicial.PactLocked = true
	tree := svc.witnessTree()
	if !witnessNode(t, tree, "conv_pact").Lit {
		t.Fatal("locked charter must light 《公议》")
	}
	personal := witnessNode(t, tree, "conv_personal")
	if personal.Lit {
		t.Fatalf("personal rule must not light without its facts: %+v", personal)
	}
	if len(personal.Missing) == 0 {
		t.Fatal("unlit personal rule must say what is missing")
	}
	if personal.Locked {
		t.Fatal("soft exclusion means no hard lock on the convergence layer")
	}

	station.Mayor.HolderResidentID = 9
	station.Mayor.HolderName = "Named Mayor"
	station.Authority.W = 20
	station.Authority.PoliceObedience = 0.7
	station.Authority.Judicial.Regime = "SWING"
	station.SetMayorLegitimacy(0.6)
	if node := witnessNode(t, svc.witnessTree(), "conv_personal"); !node.Lit {
		t.Fatalf("personal-rule facts must light the node: %+v", node)
	}
}

// TestWitnessTreeDeterminism 锁定确定性：同一组事实两次构建逐字段一致，
// 树的构建不消耗 RNG、不写状态。
func TestWitnessTreeDeterminism(t *testing.T) {
	svc := linesFixture()
	states := svc.statesMapFor(svc.silo.ID)
	states["player_actions:receiver_verify_accompany"] = litState(122, 4)
	states["special:replication_progress"] = model.ContentEventState{
		Triggered: true, TriggerCount: 2, LastTriggeredTimestamp: gameTimestamp(122, 6),
	}
	states["special:pre_fall_choice"] = litState(129, 1)
	states["player_actions:pre_fall_open_inquiry"] = litState(129, 2)
	svc.silo.Authority.Judicial.IndependentPublicFacts = 3

	first := svc.witnessTree()
	second := svc.witnessTree()
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("tree must be a pure derivation of facts:\n%+v\n%+v", first, second)
	}
}

package service

import (
	"testing"

	"silo40/internal/engine"
)

// 谈判"非动作结局"完成键接线验收：妥协成立与信道破裂由引擎回合产出，
// 本身没有玩家动作完成键；延迟回访事件靠这两个键做 eventTriggered /
// monthsSinceEventGte 锚点。键只记事实：不进仲裁、不改资格。

// TestNegotiationOutcomeKeysRecorded 锁定 helper 的写入契约：妥协成立写
// 妥协键，镇压/曝光打断信道写破裂键，其余回合结果不写任何键。
func TestNegotiationOutcomeKeysRecorded(t *testing.T) {
	svc := newNegotiationGame(t, "neg-outcome-keys.db", 4210)

	svc.recordNegotiationOutcomeKeys(svc.silo.ID, &engine.NegotiationRoundResult{
		CompromiseEstablished: true,
	})
	if !contentTriggered(svc, negotiationCompromiseKey) {
		t.Fatalf("compromise outcome must light %s", negotiationCompromiseKey)
	}
	if contentTriggered(svc, negotiationRuptureKey) {
		t.Fatal("compromise round must not light the rupture key")
	}

	svc2 := newNegotiationGame(t, "neg-outcome-keys-2.db", 4211)
	svc2.recordNegotiationOutcomeKeys(svc2.silo.ID, &engine.NegotiationRoundResult{
		SuppressBrokeChannel: true,
	})
	if !contentTriggered(svc2, negotiationRuptureKey) {
		t.Fatalf("crackdown rupture must light %s", negotiationRuptureKey)
	}

	svc3 := newNegotiationGame(t, "neg-outcome-keys-3.db", 4212)
	svc3.recordNegotiationOutcomeKeys(svc3.silo.ID, &engine.NegotiationRoundResult{
		ITExposedChannel: true,
	})
	if !contentTriggered(svc3, negotiationRuptureKey) {
		t.Fatalf("IT exposure rupture must light %s", negotiationRuptureKey)
	}

	svc4 := newNegotiationGame(t, "neg-outcome-keys-4.db", 4213)
	svc4.recordNegotiationOutcomeKeys(svc4.silo.ID, &engine.NegotiationRoundResult{})
	if contentTriggered(svc4, negotiationCompromiseKey) || contentTriggered(svc4, negotiationRuptureKey) {
		t.Fatal("plain rounds must not write outcome keys")
	}
}

// TestNegotiationOutcomeKeysIdempotent 锁定幂等守卫：重复的结局回合不得
// 刷新键的时间戳与计数 (延迟回访按锚点月计算间隔)。
func TestNegotiationOutcomeKeysIdempotent(t *testing.T) {
	svc := newNegotiationGame(t, "neg-outcome-idem.db", 4214)

	result := &engine.NegotiationRoundResult{CompromiseEstablished: true}
	svc.recordNegotiationOutcomeKeys(svc.silo.ID, result)
	state := svc.statesMapFor(svc.silo.ID)[negotiationCompromiseKey]

	if _, err := svc.PassTime(1); err != nil {
		t.Fatalf("advance month: %v", err)
	}
	svc.recordNegotiationOutcomeKeys(svc.silo.ID, result)

	again := svc.statesMapFor(svc.silo.ID)[negotiationCompromiseKey]
	if again.TriggerCount != state.TriggerCount {
		t.Fatalf("outcome key must stay written once: count %d -> %d",
			state.TriggerCount, again.TriggerCount)
	}
	if again.LastTriggeredTimestamp != state.LastTriggeredTimestamp {
		t.Fatalf("outcome key timestamp must not refresh: %d -> %d",
			state.LastTriggeredTimestamp, again.LastTriggeredTimestamp)
	}
}

// TestNegotiationTerminalRoundLightsOutcomeKey 在真实总线上跑：信道从开启
// 走到任一合法终局 (妥协成立或信道崩塌) 时，对应的结局完成键必须已点亮。
func TestNegotiationTerminalRoundLightsOutcomeKey(t *testing.T) {
	svc := newNegotiationGame(t, "neg-terminal.db", 4215)
	svc.silo.Rebellion = 0.6
	executePlayerEventAction(t, svc, "NEGOTIATION_OPEN_CHANNEL")

	for i := 0; i < 24 && svc.silo.Negotiation.NegotiationActive(); i++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance month %d: %v", i, err)
		}
	}

	negotiation := svc.silo.Negotiation
	switch {
	case negotiation.CompromiseEstablished:
		if !contentTriggered(svc, negotiationCompromiseKey) {
			t.Fatalf("compromise established but %s not lit: %+v",
				negotiationCompromiseKey, negotiation)
		}
	case negotiation.Closed:
		if !contentTriggered(svc, negotiationRuptureKey) {
			t.Fatalf("channel closed (%s) but %s not lit: %+v",
				negotiation.CloseReason, negotiationRuptureKey, negotiation)
		}
	default:
		// 24 个月仍在通气也合法：结局键只在终局回合出现。
	}
}

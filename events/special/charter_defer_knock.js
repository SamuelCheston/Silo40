defineEvent({
  id: "charter_defer_knock",
  title: "第二次敲门",
  description: "他又来了。这一次没带调度单，只带着你留下的那份抄本，边角已经磨软。'规程不等人，但我还认得路。'他把抄本放在桌上，没有收回去的意思。起草会的灯暗了——想重新点起来，得从第一条条款重新吵。门还留着那条缝，但这一次，你得给个答复。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 延迟回访（暂缓键 +6 月，TODO(balance)）：兑现暂缓文案的门缝承诺。
  // 点亮即经 action_gates 解锁 CHARTER_SECOND_ACCEPT / CHARTER_SECOND_DECLINE
  // 两个新完成键——换条件再来一次（动能归零，窗口重新计时），不是死胡同。
  trigger: all(
    eventTriggered("player_action", "CHARTER_CONTACT_DEFER"),
    monthsSinceEventGte("player_action", "CHARTER_CONTACT_DEFER", 6)
  ),
  script: {
    // 后果是门控解锁（action_gates 读 special:charter_defer_knock），
    // 不是数值——敲门本身只点亮状态。
    apply: function(ctx) {
      return { effects: [], emit: [] };
    },
  },
});
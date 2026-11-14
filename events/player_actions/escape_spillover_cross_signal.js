defineEvent({
  id: "escape_spillover_cross_signal",
  title: "信道：跨堡信号",
  description: "信号能力被调到极限：一段定向的窄波束，一次只够传几行字。它不如专线稳，也不如人可靠——但它不经过任何一双可能背叛的手。",
  type: "TECHNICAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 4, // TODO(balance)
  // 外溢信道建立（16 §2.3）：跨堡信号类，受 Topology 邻接约束。
  player_action: {
    id: "ESCAPE_SPILLOVER_CROSS_SIGNAL",
    label: "建立跨堡信号信道",
    description: "以既有信号能力向邻堡建立定向投递信道。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_SPILLOVER_CROSS_SIGNAL"),
  effects: [
    spilloverChannelAdd("cross_signal", 3, 0.55), // TODO(balance)
  ],
});

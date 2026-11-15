defineEvent({
  id: "escape_spillover_send_cross_signal",
  title: "投递：定向波束",
  description: "信号能力被拧到极限，一段窄波束穿过黑暗。一次只够传几行字——包裹里最重的部分只能留下，其余等下一次。波束不经过任何一双可能背叛的手，但衰减在路上。",
  type: "TECHNICAL",
  fire_mode: "REPEATABLE",
  // 外溢投递（16 §3.5 投递侧）：跨堡信号类。只带得动包裹的一部分
  // (容量 0.6)、2 月在途、路上有信号衰减；受拓扑邻接约束。
  player_action: {
    id: "ESCAPE_SPILLOVER_SEND_CROSS_SIGNAL",
    label: "经跨堡信号送出包裹",
    description: "以定向波束投递外溢包裹。只带得动一部分内容，路上还有衰减，但不经过任何中间人。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_SPILLOVER_SEND_CROSS_SIGNAL"),
  effects: [
    spilloverSend("cross_signal"),
  ],
});

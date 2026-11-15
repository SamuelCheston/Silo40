defineEvent({
  id: "negotiation_hands_off",
  title: "抉择：不接管，带组织离开",
  description: "你们选择不接管市长职位：组织完整保留，方向从对抗转向离开。出发动员获得组织加成，外溢信道的可靠度也上升了一点。市长留任，但不再阻碍——这一层只改变谁在阻碍谁。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 谈判层·妥协后二选一（18 设计卷 §3.7）：不接管市长职位，带着既有
  // 组织能力推进逃跑（或作为退路的外溢）。出发结算读 hands_off 加成，
  // 外溢信道可靠度小量上升。与 negotiation_takeover 互斥（先选者生效）。
  player_action: {
    id: "NEGOTIATION_HANDS_OFF",
    label: "不接管，带组织离开",
    description: "保留组织不接管职位，把力量转向离开本身。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 0,
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("NEGOTIATION_HANDS_OFF"),
  effects: [
    negotiationResolve("hands_off"),
  ],
});

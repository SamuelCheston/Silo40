defineEvent({
  id: "negotiation_probe",
  title: "谈判：再递一轮话",
  description: "中间人又进了一次市长办公室。报一个数字、探一句口风——市长不必立刻表态，你们也不必。通气本身就是谈判的一部分：拖延是合法结果，但每一次试探都在改变对方的掂量。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 1, // TODO(balance)
  // 谈判层·试探（21 设计卷 §5）：高强度回合。市长路径决策的权重读
  // 既有状态（正统性、叛乱、保障可信度、性情、压力），试探只通气不逼表态。
  player_action: {
    id: "NEGOTIATION_PROBE",
    label: "试探市长口风",
    description: "通过信道再递一轮话，推动市长重新掂量。可能换来拖延、镇压或让步。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 1, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("NEGOTIATION_PROBE"),
  effects: [
    negotiationRound("probe"),
  ],
});

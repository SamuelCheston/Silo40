defineEvent({
  id: "succession_pure_rule",
  title: "继承抉择：接受纯粹人治",
  description: "你决定不补继承的洞：秩序与此人同寿。驻地得到了它现在能承受的最好的统治者——一个可接受的统治者，而不是一个充分知情的社会。你换来的是时间与活人，不是一个答案。被保护的睡眠，与一个知道代价的守夜人。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // H-1 pure branch (30-nodes §4, 31-spec §1): choice P-D projection. The
  // line converges; qualification stays an engine fact (personalRuleEligible).
  player_action: {
    id: "SUCCESSION_PURE_RULE",
    label: "接受纯粹人治",
    description: "不补继承的洞；秩序与此人同寿，本线收束。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 0,            // initial calibration, TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("SUCCESSION_PURE_RULE"),
  // 抉择即时后果（注释对账）：文案承诺"驻地得到了它现在能承受的最好的
  // 统治者……被保护的睡眠"——当下稳住，凝聚 +0.02；文案同时承认"不是
  // 一个答案"——继承问题被搁置不再追问，历史包袱小幅松绑 −0.02。
  // 收束加分仍由 weighted_endings 的 PersonalRuleSuccessionBonus 读完成键。
  effects: [
    siloMetricDelta("cohesion", 0.02), // initial calibration, TODO(balance)
    siloMetricDelta("history_burden", -0.02), // initial calibration, TODO(balance)
  ],
});

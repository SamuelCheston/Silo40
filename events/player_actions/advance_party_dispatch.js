defineEvent({
  id: "advance_party_dispatch",
  title: "先遣出发",
  description: "名单上的名字都签了字：知道风险，知道回程时限，知道如果回不来，后面的人不会把他们的名字写进叛乱者的名单。先遣小组带着最轻的装备走进检修出口。驻地屏住呼吸。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 先遣派遣（13 分册 §1）：路线确认后解锁；4 个月后无条件回报，
  // 完好/受损由期间是否发生底层坍塌事故区分。
  player_action: {
    id: "ADVANCE_PARTY_DISPATCH",
    label: "派遣先遣小组",
    description: "派先遣小组沿已确认路线往返，验证通行与接应点。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ADVANCE_PARTY_DISPATCH"),
  effects: [
    scheduleEvent("advance_party_report", 4), // TODO(balance)
  ],
});

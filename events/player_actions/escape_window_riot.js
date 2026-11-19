defineEvent({
  id: "escape_window_riot",
  title: "全面暴乱",
  description: "楼层在火光里连成一片。出口确实被撞开了，但每一层都在付出人口、资源与结构的代价。独立群体还没有成立，驻地先开始碎裂——这场反抗的账单，会在撤离的路上和留下的废墟里分两次到期。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 抉择 R-D 的对立支：全面暴乱＝高损耗（楼层与资源），撤离秩序系数低；
  // 损耗进入《断层》节点群口径（13 分册 §2）。
  // 文案「账单分两次到期」已兑现：第二期见 special/escape_window_riot_bill.js
  // （完成键 +3 月，全楼层稳定再降、恐慌余波）。
  player_action: {
    id: "ESCAPE_WINDOW_RIOT",
    label: "把叛乱推到最高",
    description: "以楼层冲突掩护撤离：出口会开，人口、资源与结构也会快速损耗。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 1, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_WINDOW_RIOT"),
  effects: [
    siloMetricDelta("rebellion", 0.10),          // TODO(balance)
    floorStabilityDelta("", -0.08),              // 全部楼层，TODO(balance)
    professionMetricDeltaAll("panic_value", 0.06), // TODO(balance)
  ],
});

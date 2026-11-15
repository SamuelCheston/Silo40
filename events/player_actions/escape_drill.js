defineEvent({
  id: "escape_drill",
  title: "演练",
  description: "批次在维修夹层里无声地集合、行进、解散。没有人喊口令，计时的人掐着表。演练让撤离从一张纸变成肌肉记忆——也让筹备的痕迹一次次暴露在同一批眼睛面前。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 3, // TODO(balance)
  // 演练（16 §2.2）：按次累积演练水平；可见性成本 = 筹备暴露增量。
  player_action: {
    id: "ESCAPE_DRILL",
    label: "组织撤离演练",
    description: "以可见性为代价换取演练水平：批次集结与行进协同。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_DRILL"),
  effects: [
    preparationDelta("drill_level", 0.15),  // TODO(balance)
    preparationDelta("prep_exposure", 0.06), // 可见性成本，TODO(balance)
    // 动员范围驱动组织怀疑度：批次集结是最显眼的组织行为（20 修订/ADR-0014）。
    orgSuspicionDelta(0.06), // TODO(balance)
  ],
});

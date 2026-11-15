defineEvent({
  id: "escape_plan_medical",
  title: "预案对接：医疗",
  description: "医疗侧评估的是暴露与节奏：谁不能在暴露段急行、伤病者怎么编队、抵达后的基本照护从哪里来。他们坚持在预案里写下最坏的一栏——'如果一半人需要照顾'。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // R-2 功能位：Medical = 暴露风险与照护。
  player_action: {
    id: "ESCAPE_PLAN_MEDICAL",
    label: "对接医疗预案",
    description: "与医疗部门完成暴露风险与照护的实质对接，计入预案完备度。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 1, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_PLAN_MEDICAL"),
  effects: [
    preparationPlanRole("medical"),
    preparationDelta("prep_exposure", 0.02), // TODO(balance)
    // 动员范围驱动组织怀疑度：对接功能位＝把一个部门拉进计划（20 修订/ADR-0014）。
    orgSuspicionDelta(0.03), // TODO(balance)
  ],
});

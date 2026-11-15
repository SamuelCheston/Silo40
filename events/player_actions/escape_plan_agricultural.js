defineEvent({
  id: "escape_plan_agricultural",
  title: "预案对接：农业",
  description: "农业的人不承诺目的地立即丰收——他们只承诺两件事：能带走的种子与口粮，以及落地之后第一块田怎么开。'明天'是从这一页预案里长出来的。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // R-2 功能位：Agricultural = 可携带食物与生产起点。
  player_action: {
    id: "ESCAPE_PLAN_AGRICULTURAL",
    label: "对接农业预案",
    description: "与农业部门完成可携带食物与生产起点的实质对接，计入预案完备度。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 1, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_PLAN_AGRICULTURAL"),
  effects: [
    preparationPlanRole("agricultural"),
    preparationDelta("prep_exposure", 0.02), // TODO(balance)
    // 动员范围驱动组织怀疑度：对接功能位＝把一个部门拉进计划（20 修订/ADR-0014）。
    orgSuspicionDelta(0.03), // TODO(balance)
  ],
});

defineEvent({
  id: "escape_plan_mines",
  title: "预案对接：矿业",
  description: "矿业的人把地下结构图摊开：哪条夹层的承重够、哪一段的通行要限流、撤离本身会不会动摇驻地的根基。他们不反对离开，只反对把山当成不会塌的东西。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // R-2 功能位：Mines = 通行限制与结构影响。
  player_action: {
    id: "ESCAPE_PLAN_MINES",
    label: "对接矿业预案",
    description: "与矿业部门完成通行与结构的实质对接，计入预案完备度。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 1, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_PLAN_MINES"),
  effects: [
    preparationPlanRole("mines"),
    preparationDelta("prep_exposure", 0.02), // TODO(balance)
  ],
});

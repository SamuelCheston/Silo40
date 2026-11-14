defineEvent({
  id: "escape_plan_supply",
  title: "预案对接：补给",
  description: "补给侧把人口、行程与消耗换算成多批次的物资清单：每一批带多少、留多少、接应点囤多少。'带够粮食'这种话被从预案里划掉了——留下的全是人·月和批次号。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // R-2 功能位：Supply = 多批次物资换算。
  player_action: {
    id: "ESCAPE_PLAN_SUPPLY",
    label: "对接补给预案",
    description: "与补给部门完成多批次物资换算的实质对接，计入预案完备度。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 1, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_PLAN_SUPPLY"),
  effects: [
    preparationPlanRole("supply"),
    preparationDelta("prep_exposure", 0.02), // TODO(balance)
  ],
});

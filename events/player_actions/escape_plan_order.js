defineEvent({
  id: "escape_plan_order",
  title: "预案对接：秩序",
  description: "市长圈层与警务侧的人不是被收买的——他们是被说服的：撤离要有人维持公开秩序、避免踩踏、阻止把撤离变成派系清洗。这一页预案不是必须的，但有了它，离开才不像逃散。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // R-2 功能位：Mayor/Police = 公开秩序（可争取非必须）。
  player_action: {
    id: "ESCAPE_PLAN_ORDER",
    label: "对接秩序预案",
    description: "争取市长/警务侧维持撤离公开秩序，计入预案完备度。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 1, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_PLAN_ORDER"),
  effects: [
    preparationPlanRole("order"),
    preparationDelta("prep_exposure", 0.02), // TODO(balance)
  ],
});

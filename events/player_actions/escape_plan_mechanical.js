defineEvent({
  id: "escape_plan_mechanical",
  title: "预案对接：机械",
  description: "机械侧的答复很具体：哪几道闸门能重复开启、出口设备要换哪几个件、转移工具能撑几个批次。'门只开一次'的撤离不配叫撤离——他们把'可重复使用'写进了预案的第一页。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 预案链（16 §2.2/§6.3）：功能位对接，写预案完备度（功能位覆盖率）。
  // R-2 功能位：Mechanical = 出口与设备可重复使用。
  player_action: {
    id: "ESCAPE_PLAN_MECHANICAL",
    label: "对接机械预案",
    description: "与机械部门完成出口与设备的实质对接，计入预案完备度。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 1, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_PLAN_MECHANICAL"),
  effects: [
    preparationPlanRole("mechanical"),
    preparationDelta("prep_exposure", 0.02), // TODO(balance)
  ],
});

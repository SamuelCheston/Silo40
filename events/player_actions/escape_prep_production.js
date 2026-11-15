defineEvent({
  id: "escape_prep_production",
  title: "生产起点",
  description: "种子、工具、图纸与第一批原料被装进同一份载体清单。农业的人说：'这不是为了第一顿饭——第一顿饭靠带的补给。这是为了第二个月、第一年，为了落地之后还有明天。'",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 生产起点（16 §2.2/§3.3 M3）：Agricultural 对接 + 材料划拨载体。
  // 前置：已封存足够的装备材料（preparation_gte 读账本）。
  player_action: {
    id: "ESCAPE_PREP_PRODUCTION",
    label: "备妥生产起点",
    description: "以已封存材料组装生产起点载体。需要封存材料达标。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: all(
    playerActionIs("ESCAPE_PREP_PRODUCTION"),
    preparationGte("materials_packed", 50), // TODO(balance)
  ),
  effects: [
    preparationSetBool("production_start", true),
    preparationDelta("prep_exposure", 0.03), // TODO(balance)
    // 动员范围驱动组织怀疑度：布置生产起点要动用人手（20 修订/ADR-0014）。
    orgSuspicionDelta(0.03), // TODO(balance)
  ],
});

defineEvent({
  id: "escape_prep_suit_repair",
  title: "暗中修复防护服",
  description: "清洁服全是处刑工具：劣质热封胶带让每一套都必然失效。机械师在维修间里换优质胶带，一套一套地换，一次一次地赌审计不来。可用防护服不是存款——它随时可能被换回劣质胶带，或者被整个没收。",
  type: "TECHNICAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 2, // TODO(balance)
  // 修服链（16 §2.4 来源二）：本地清洁服转化。暗中换胶带，每次获取有
  // 暴露成本；IT 立场塑形可得性与成本（审计反制见 escape_prep_audit_seizure）。
  player_action: {
    id: "ESCAPE_PREP_SUIT_REPAIR",
    label: "暗中修复防护服",
    description: "接触供应链取得服体并换优质胶带。+1 可用防护服，累积筹备暴露。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_PREP_SUIT_REPAIR"),
  effects: [
    preparationDelta("suits_functional", 1),   // TODO(balance)
    // 修服是筹备暴露的主要来源：胶带缺口越大越显眼。
    // 0.04/次 (16 §4.2-3)：全程约 16 次修复累积 0.64，配合 IT 配合系数
    // (×0.4) 后追踪基数约 0.20，留给 Silo 1 注意力推升到 aware/trailing；
    // 原值 0.08 会让任何认真筹备直接锁死 pursuit 档，与设计意图矛盾。
    preparationDelta("prep_exposure", 0.04),   // TODO(balance)
  ],
});

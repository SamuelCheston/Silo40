defineEvent({
  id: "escape_prep_care",
  title: "照护准备",
  description: "医疗储备被分成小包：暴露段的急救、落地后的基础照护、走不动的人的随行方案。'照护能力'不是口号——它是清单上每一项都划了勾之后才存在的词。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 照护能力（16 §2.2）：Medical 对接 + 医疗储备。
  player_action: {
    id: "ESCAPE_PREP_CARE",
    label: "建立照护储备",
    description: "完成医疗储备与随行照护方案，写入照护能力。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_PREP_CARE"),
  effects: [
    preparationDelta("care_ready", 0.4),    // TODO(balance)
    preparationDelta("prep_exposure", 0.03), // TODO(balance)
  ],
});

defineEvent({
  id: "escape_spillover_package_facts",
  title: "封装：事实与来源",
  description: "包裹的第一层不是结论，是出处：核心事实、原始材料的所在、哪些解释仍有争议。只复制结论，得到的是一批跟着喊话的人；复制来源，得到的才是能自己查账的人。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 外溢包裹组装（16 §2.3，I-3 五种能力之一）：事实与来源。
  // 每种能力一份完成键，各 +0.2 包裹质量（TODO(balance)）。
  player_action: {
    id: "ESCAPE_SPILLOVER_PACKAGE_FACTS",
    label: "封装事实与来源",
    description: "把核心事实、材料出处与争议点整理进包裹。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_SPILLOVER_PACKAGE_FACTS"),
  effects: [
    preparationDelta("package_quality", 0.2), // TODO(balance)
  ],
});

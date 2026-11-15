defineEvent({
  id: "escape_spillover_package_conceal",
  title: "封装：隐匿与交接",
  description: "第四样东西没有写在任何一页的正面：分段持有、来源脱敏、例行岗位下的掩护、该停手的风险信号、以及显式的角色交接。一处清档或一名证人被带走就暴露全链的包裹，不配离开宿主。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 外溢包裹组装（16 §2.3，I-3 五种能力之四）：隐匿与交接。
  player_action: {
    id: "ESCAPE_SPILLOVER_PACKAGE_CONCEAL",
    label: "封装隐匿与交接规程",
    description: "把分段持有、脱敏与交接规程写进包裹。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_SPILLOVER_PACKAGE_CONCEAL"),
  effects: [
    preparationDelta("package_quality", 0.2), // TODO(balance)
  ],
});

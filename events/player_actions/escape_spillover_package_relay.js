defineEvent({
  id: "escape_spillover_package_relay",
  title: "封装：再传递",
  description: "包裹的最后一层是关于包裹之外的：怎样选出下一个接收者，怎样让对方亲自完成一次核验，怎样把这五种能力继续交下去。只能复制文件的链，到第二代就死了。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 外溢包裹组装（16 §2.3，I-3 五种能力之五）：再传递。
  player_action: {
    id: "ESCAPE_SPILLOVER_PACKAGE_RELAY",
    label: "封装再传递指引",
    description: "把选择下一接收者与继续传递的步骤写进包裹。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_SPILLOVER_PACKAGE_RELAY"),
  effects: [
    preparationDelta("package_quality", 0.2), // TODO(balance)
  ],
});

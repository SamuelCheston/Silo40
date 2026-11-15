defineEvent({
  id: "escape_spillover_package_counter",
  title: "封装：反制方法",
  description: "第三样东西最沉：40 号留下的可行动知识，加上驻地亲手验证过的渠道反制与它的局限。只告诉对方那边很可怕，却给不出能压低任何一条真实控制渠道的办法，等于什么也没给。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 外溢包裹组装（16 §2.3，I-3 五种能力之三）：反制方法。
  player_action: {
    id: "ESCAPE_SPILLOVER_PACKAGE_COUNTER",
    label: "封装反制方法",
    description: "把已验证的反制手段与局限写进包裹。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_SPILLOVER_PACKAGE_COUNTER"),
  effects: [
    preparationDelta("package_quality", 0.2), // TODO(balance)
  ],
});

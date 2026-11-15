defineEvent({
  id: "escape_spillover_package_verify",
  title: "封装：独立核验",
  description: "包裹里放下的第二样东西是方法：怎样查本地的账、怎样复核实物、去哪里找第二个来源、怎样识别同一份转述伪装成的两个证人。没有这一层，每个节点都只能靠上一节点亲口保证。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 外溢包裹组装（16 §2.3，I-3 五种能力之二）：独立核验。
  player_action: {
    id: "ESCAPE_SPILLOVER_PACKAGE_VERIFY",
    label: "封装独立核验方法",
    description: "把查账、复核与多源比对的步骤写进包裹。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_SPILLOVER_PACKAGE_VERIFY"),
  effects: [
    preparationDelta("package_quality", 0.2), // TODO(balance)
  ],
});

defineEvent({
  id: "negotiation_takeover",
  title: "抉择：接管市长职位",
  description: "委员会接管市长办公室——顶层改由你们主导，旧市长退居幕后。接管改变的是谁在阻碍谁，不是胜负本身：能带走多少人，仍由出发结算说了算。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 谈判层·妥协后二选一（18 设计卷 §3.7）：接管市长职位。正统性改写为
  // 委员会正统性，市长立场以"叛乱方主导顶层"的权重输入重估（仍是带权
  // 取样，不写死）。与 negotiation_hands_off 互斥（先选者生效）。
  player_action: {
    id: "NEGOTIATION_TAKEOVER",
    label: "接管市长职位",
    description: "由叛乱方委员会接管顶层，主导后续安排。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 0,
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("NEGOTIATION_TAKEOVER"),
  effects: [
    negotiationResolve("takeover"),
  ],
});

defineEvent({
  id: "escape_spillover_send_it_line",
  title: "投递：专线夹带",
  description: "包裹被拆进例行流量的形状里，顺着专线另一头的日常秩序滑出去。这是最快的路，也是最容易被注意的路——流量形状的每一次异常，都会被对岸的档案记下。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  // 外溢投递（16 §3.5 投递侧）：IT 专线类。整包夹带、1 月在途、无路上
  // 损耗；可靠度最高，暴露成本也最高。结算读建立侧账本（目标堡与当前
  // 可靠度），服务层执行可靠度抽取、暴露记账与跨堡入队。
  player_action: {
    id: "ESCAPE_SPILLOVER_SEND_IT_LINE",
    label: "经专线送出包裹",
    description: "借本地 IT 受限层把外溢包裹夹带出去。能带整包，但每一次使用都会抬高对岸的注意。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_SPILLOVER_SEND_IT_LINE"),
  effects: [
    spilloverSend("it_line"),
  ],
});

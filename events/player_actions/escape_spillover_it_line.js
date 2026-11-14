defineEvent({
  id: "escape_spillover_it_line",
  title: "信道：IT 专线夹带",
  description: "本地 IT 的受限知情层里，有人愿意在例行流量里夹带一点别的东西。这是最可靠的信道，也是最危险的——专线另一头的眼睛，迟早会注意到流量形状的异常。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 4, // TODO(balance)
  // 外溢信道建立（16 §2.3 三种信道之一）：IT 专线类。可靠度最高、
  // 暴露成本最高。投递与独立接收结算属 Layer 3，本期只登记信道。
  player_action: {
    id: "ESCAPE_SPILLOVER_IT_LINE",
    label: "建立 IT 专线信道",
    description: "借本地 IT 受限层建立跨堡专线夹带信道。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_SPILLOVER_IT_LINE"),
  effects: [
    // 目标堡 3 = 18 号（三堡布局）；可靠度与暴露成本 TODO(balance)。
    spilloverChannelAdd("it_line", 3, 0.75),
  ],
});

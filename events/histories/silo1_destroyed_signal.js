defineEvent({
  id: "silo1_destroyed_signal",
  title: "一号地堡失去联系",
  description: "所有与一号地堡的通信协议均已超时，服务器不再响应。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // 覆灭/结算锚点：Y130M1（默认开局距 96 个月）。这是固定仲裁时点，不是游戏标准时长
  // 或总长度（ADR-0018）；该时点是否被旧 96 月口径带偏列入复核，见事件计划 §1.4。
  trigger: timestampAtOrAfter(130, 1),
  effects: [
    siloFlagSet("silo1_destroyed", true),
  ],
});

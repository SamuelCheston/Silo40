defineEvent({
  id: "escape_window_orderly_yield",
  title: "轮班表的利息",
  description: "三个月过去，那张轮班表还在墙上，边角卷了，没人去揭。谁维护出口、谁按批次配给、谁照顾走不快的人——名单上的名字一个都没有换。楼层里开始有人把这件事当作理所当然：通道是通的，秩序是有人在管的。恐慌退了一小步，不是因为危险变小了，而是因为有人看见过事情被办成的样子。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 延迟回访（有序键 +3 月，TODO(balance)）：有序占用的红利不在开窗当天，
  // 而在三个月后——班次照跑、配给照发，秩序被反复看见，恐慌回落、凝聚上升。
  trigger: all(
    eventTriggered("player_action", "ESCAPE_WINDOW_ORDERLY"),
    monthsSinceEventGte("player_action", "ESCAPE_WINDOW_ORDERLY", 3)
  ),
  effects: [
    professionMetricDeltaAll("panic_value", -0.02), // TODO(balance)
    siloMetricDelta("cohesion", 0.01),              // TODO(balance)
  ],
});

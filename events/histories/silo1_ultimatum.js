defineEvent({
  id: "silo1_ultimatum",
  title: "一号地堡最后通牒",
  description: "旧协议通道里传来一号地堡的最后通牒：若叛乱继续，将执行重置或终止。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  trigger: timestampAtOrAfter(126, 1),
  effects: [
    professionMetricDeltaAll("panic_value", 0.08),
    siloMetricDelta("rebellion", 0.05),
    scheduleEvent("ultimatum_aftermath", 3),
  ],
});

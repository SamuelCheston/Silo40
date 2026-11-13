defineEvent({
  id: "silo1_fallout",
  title: "一号地堡失联余波",
  description: "一号地堡失联的消息持续扩散。控制网络断了——居民间同时涌起猜疑、狂喜与一种无人敢说出口的指望。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  trigger: siloFlagTrue("silo1_destroyed"),
  effects: [
    professionMetricDeltaAll("panic_value", 0.05),
    siloMetricDelta("rebellion", 0.05),
    siloMetricDelta("history_burden", 0.03),
  ],
});

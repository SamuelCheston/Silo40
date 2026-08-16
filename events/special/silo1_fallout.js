defineEvent({
  id: "silo1_fallout",
  title: "一号地堡失联余波",
  description: "一号地堡失联的消息持续扩散，居民间开始出现新的猜疑与不安。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  trigger: all(
    siloFlagTrue("silo1_destroyed"),
    siloMetricGte("cohesion", 0.31),
  ),
  effects: [
    siloMetricDelta("cohesion", -0.05),
    professionMetricDeltaAll("panic_value", 0.05),
  ],
});

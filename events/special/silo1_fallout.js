defineEvent({
  id: "silo1_fallout",
  title: "静默余波",
  description: "那套系统不再有回音。居民说不清断了什么，只觉得那套一直盯着他们的东西安静了下来——猜疑、隐秘的松快与一种无人敢说出口的指望，在部门间悄悄蔓延。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  trigger: siloFlagTrue("silo1_destroyed"),
  effects: [
    professionMetricDeltaAll("panic_value", 0.05),
    siloMetricDelta("rebellion", 0.05),
    siloMetricDelta("history_burden", 0.03),
  ],
});

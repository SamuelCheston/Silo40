defineEvent({
  id: "geothermal_surge",
  title: "地热脉冲",
  description: "深井传来一次季节性的地热脉冲，发电机出力短暂回升。走廊里的灯亮了一些，人们的脸色也亮了一些。",
  type: "EXTERNAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 12,
  trigger: all(
    timestampAtOrAfter(123, 6),
    // No JS helper exists for resource triggers; raw object matches the
    // engine-supported "silo_resource_lte" trigger type.
    { type: "silo_resource_lte", metric: "Energy", value: 500 },
  ),
  effects: [
    siloResourceDelta("Energy", 600),
    professionMetricDeltaAll("panic_value", -0.04),
  ],
});

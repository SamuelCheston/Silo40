defineEvent({
  id: "water_leak",
  title: "供水管线泄漏",
  description: "底层机械部报告发生严重水管泄漏，部分楼层供水中断。老化的管线又欠下一笔结构债。",
  type: "TECHNICAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 9,
  // No JS helper exists for resource triggers; raw object matches the
  // engine-supported "silo_resource_lte" trigger type.
  trigger: { type: "silo_resource_lte", metric: "Materials", value: 800 },
  effects: [
    siloResourceDelta("Supplies", -400),
    professionMetricDeltaAll("panic_value", 0.03),
    floorStabilityDelta("Lower", -0.04),
  ],
});

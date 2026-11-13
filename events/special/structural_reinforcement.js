defineEvent({
  id: "structural_reinforcement",
  title: "底层结构加固",
  description: "工程队趁材料还有富余，对底层受损的支撑做了一轮加固。楼梯间的晃动轻了一些。",
  type: "TECHNICAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 12,
  trigger: all(
    timestampAtOrAfter(124, 1),
    // No JS helper exists for resource triggers; raw object matches the
    // engine-supported "silo_resource_gte" trigger type.
    { type: "silo_resource_gte", metric: "Materials", value: 400 },
  ),
  effects: [
    siloResourceDelta("Materials", -600),
    floorStabilityDelta("Lower", 0.10),
    professionMetricDeltaAll("panic_value", -0.02),
  ],
});

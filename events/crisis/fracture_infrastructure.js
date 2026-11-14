defineEvent({
  id: "fracture_infrastructure",
  title: "维生基础失守",
  description: "无人控制的破坏终于落到维生基础上。下层管线与供电节点同时受损，抢修队已经只能挑选先放弃哪里。",
  type: "CRISIS",
  fire_mode: "ONCE",
  trigger: all(
    timestampAtOrAfter(124, 1),
    siloMetricGte("rebellion", 0.8),
  ),
  effects: [
    floorStabilityDelta("Lower", -0.25),
    siloResourceDelta("Energy", -400),
    professionMetricDeltaAll("panic_value", 0.06),
  ],
});

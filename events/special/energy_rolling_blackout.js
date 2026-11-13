defineEvent({
  id: "energy_rolling_blackout",
  title: "分区轮流停电",
  description: "能源储备跌破安全线，地堡开始分区轮流停电。黑暗沿着楼层一格一格地爬。",
  type: "TECHNICAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 12,
  // No JS helper exists for resource triggers; raw object matches the
  // engine-supported "silo_resource_lte" trigger type.
  trigger: { type: "silo_resource_lte", metric: "Energy", value: 300 },
  effects: [
    professionMetricDeltaAll("panic_value", 0.06),
    scheduleEvent("blackout_cascade", 2),
  ],
});

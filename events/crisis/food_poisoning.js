defineEvent({
  id: "food_poisoning",
  title: "群体性食物中毒",
  description: "水培区的一批农产品受到污染，引发大规模恐慌。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 9,
  // No JS helper exists for resource triggers; raw object matches the
  // engine-supported "silo_resource_lte" trigger type.
  trigger: { type: "silo_resource_lte", metric: "Supplies", value: 2000 },
  effects: [
    professionMetricDeltaAll("panic_value", 0.08),
    siloPopulationDelta(-40),
  ],
});

defineEvent({
  id: "hydroponics_mutant_strain",
  title: "变异株选育",
  description: "水培区选育出一株高产变异作物，收成一次性大幅回升。没有人问它为什么长这么快。",
  type: "TECHNICAL",
  fire_mode: "ONCE",
  trigger: all(
    timestampAtOrAfter(123, 6),
    // No JS helper exists for resource triggers; raw object matches the
    // engine-supported "silo_resource_lte" trigger type.
    { type: "silo_resource_lte", metric: "Supplies", value: 800 },
  ),
  effects: [
    siloResourceDelta("Supplies", 1000),
    scheduleEvent("strain_side_effects", 6),
  ],
});

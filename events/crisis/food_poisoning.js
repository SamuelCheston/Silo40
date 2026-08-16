defineEvent({
  id: "food_poisoning",
  title: "群体性食物中毒",
  description: "水培区的一批农产品受到污染，引发大规模恐慌。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 6,
  effects: [
    professionMetricDeltaAll("panic_value", 0.1),
    siloMetricDelta("legitimacy", -0.05),
  ],
});

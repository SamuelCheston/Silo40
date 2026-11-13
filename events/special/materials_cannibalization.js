defineEvent({
  id: "materials_cannibalization",
  title: "底层设施拆解",
  description: "工程队开始拆解底层闲置的居住与维护设施，把钢材和管线挪作应急材料。第一次是救急，往后每次都更像掠夺。",
  type: "TECHNICAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 12,
  // No JS helper exists for resource triggers; raw object matches the
  // engine-supported "silo_resource_lte" trigger type.
  trigger: { type: "silo_resource_lte", metric: "Materials", value: 200 },
  effects: [
    siloResourceDelta("Materials", 300),
    scheduleEvent("lower_deck_collapse", 4),
  ],
  script: {
    // Nth-use qualitative change: from the 5th dismantling on, crews meet
    // occupied decks and accidents multiply.
    apply: function(ctx) {
      if (ctx.runtime.trigger_count >= 5) {
        return {
          effects: [
            professionMetricDeltaAll("panic_value", 0.04),
            siloPopulationDelta(-40),
          ],
        };
      }
      return {};
    },
  },
});

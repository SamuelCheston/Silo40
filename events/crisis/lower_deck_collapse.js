defineEvent({
  id: "lower_deck_collapse",
  title: "底层结构事故",
  description: "被拆解过的住户区发生局部坍塌。人们开始怀疑：下一次塌的会不会是自己头顶那层。",
  type: "TECHNICAL",
  fire_mode: "REPEATABLE",
  // Scheduled-only event: fired via schedule_event from materials_cannibalization.
  script: {
    canTrigger: function(ctx) {
      return false;
    },
  },
  effects: [
    siloPopulationDelta(-80),
    professionMetricDeltaAll("panic_value", 0.06),
    floorStabilityDelta("Lower", -0.08),
  ],
});

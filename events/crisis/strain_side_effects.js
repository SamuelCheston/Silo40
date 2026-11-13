defineEvent({
  id: "strain_side_effects",
  title: "变异株副作用",
  description: "长期食用变异株的副作用开始显现：皮疹、失眠与短暂的认知迟滞。食堂的队伍比从前安静了。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // Scheduled-only event: fired via schedule_event from hydroponics_mutant_strain.
  script: {
    canTrigger: function(ctx) {
      return false;
    },
  },
  effects: [
    professionMetricDeltaAll("panic_value", 0.04),
    siloMetricDelta("rebellion", 0.03),
  ],
});

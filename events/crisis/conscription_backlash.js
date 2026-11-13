defineEvent({
  id: "conscription_backlash",
  title: "轮班怨气",
  description: "强制轮班的疲劳在车间里发酵。人们不反对劳动，反对的是被当作可以替换的零件。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  // Scheduled-only event: fired via schedule_event from labor_conscription.
  script: {
    canTrigger: function(ctx) {
      return false;
    },
  },
  effects: [
    siloMetricDelta("rebellion", 0.02),
    professionMetricDeltaAll("panic_value", 0.04),
  ],
});

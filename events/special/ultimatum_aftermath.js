defineEvent({
  id: "ultimatum_aftermath",
  title: "通牒余波",
  description: "恐慌在适应中回落，但怨气没有消失——它沉进了楼梯间的低声议论里。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // Scheduled-only event: fired via schedule_event from silo1_ultimatum.
  script: {
    canTrigger: function(ctx) {
      return false;
    },
  },
  effects: [
    professionMetricDeltaAll("panic_value", -0.04),
    siloMetricDelta("rebellion", 0.04),
  ],
});

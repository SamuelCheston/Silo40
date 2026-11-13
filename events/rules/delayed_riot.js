defineEvent({
  id: "delayed_riot",
  title: "暴乱爆发",
  description: "积压的恐慌终于炸开成一场暴乱。人群冲垮了几道闸门，烧掉了配给站的名册——恐慌泄了洪，但地堡记住了这件事。",
  type: "CRISIS",
  fire_mode: "REPEATABLE",
  // Scheduled-only event: fired via schedule_event from panic_storm_delay_rule.
  // canTrigger stays false so the monthly system pass never fires it early;
  // the delay queue dispatch does not evaluate triggers.
  script: {
    canTrigger: function(ctx) {
      return false;
    },
  },
  effects: [
    // Riots convert saturated panic into political pressure: rebellion rises,
    // panic vents, and a few people don't survive the night.
    siloMetricDelta("rebellion", 0.04),
    professionMetricDeltaAll("panic_value", -0.06),
    siloPopulationDelta(-80),
  ],
});

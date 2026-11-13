defineEvent({
  id: "blackout_cascade",
  title: "停电级联",
  description: "水培区的照明与温控在轮流停电中离线，一批作物报废，底层管线也在反复断电中受损。能源危机变成了食物危机。",
  type: "TECHNICAL",
  fire_mode: "REPEATABLE",
  // Scheduled-only event: fired via schedule_event from energy_rolling_blackout.
  script: {
    canTrigger: function(ctx) {
      return false;
    },
  },
  effects: [
    siloResourceDelta("Supplies", -300),
    professionMetricDeltaAll("panic_value", 0.04),
    floorStabilityDelta("Lower", -0.04),
  ],
});

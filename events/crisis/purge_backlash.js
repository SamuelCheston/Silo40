defineEvent({
  id: "purge_backlash",
  title: "清洗反弹",
  description: "被抹除的目录在楼梯间口口相传，越禁越响。清洗没有消灭历史包袱，只是给它换了一副更危险的形状。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // Scheduled-only event: fired via schedule_event from archive_purge_order.
  // canTrigger stays false so the monthly system pass never fires it early;
  // the delay queue dispatch does not evaluate triggers.
  script: {
    canTrigger: function(ctx) {
      return false;
    },
  },
  effects: [
    siloMetricDelta("rebellion", 0.04),
    siloMetricDelta("history_burden", 0.04),
    professionMetricDeltaAll("panic_value", 0.04),
  ],
});

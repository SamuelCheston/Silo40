defineEvent({
  id: "silo18_rumor_it_takeover",
  title: "谣言：邻堡市长暴毙，IT 接管",
  description: "邻堡的市长在一次下行巡视后暴毙，副手随后也死了。线路里的说法很一致：现在是 IT 的人在管事。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Cross-delivered only: fired by the Silo 18 chain (N2) via the cross-queue.
  script: {
    canTrigger: function (ctx) {
      return false;
    },
  },
  effects: [
    professionMetricDeltaAll("panic_value", 0.02),
    siloMetricDelta("history_burden", 0.01),
  ],
});

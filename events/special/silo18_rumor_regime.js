defineEvent({
  id: "silo18_rumor_regime",
  title: "谣言：邻堡政权更迭",
  description: "邻堡换了天：那个拒洁的机械师回来洗清了罪名，当选市长；IT 也换了新主管。有人的说法里带着不安——新 IT 主管的任命，是从线路深处\"远程授职\"的。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Cross-delivered only: fired by the Silo 18 chain (N5) via the cross-queue.
  script: {
    canTrigger: function (ctx) {
      return false;
    },
  },
  effects: [
    professionMetricDeltaAll("panic_value", -0.02),
    siloMetricDelta("cohesion", 0.02),
  ],
});

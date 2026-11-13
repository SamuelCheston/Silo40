defineEvent({
  id: "silo18_rumor_uprising",
  title: "谣言：邻堡起义被镇压",
  description: "邻堡的机械部门起义了，然后被血腥镇压。线路里弥漫着铁锈和血的味道。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Cross-delivered only: fired by the Silo 18 chain (N4) via the cross-queue.
  script: {
    canTrigger: function (ctx) {
      return false;
    },
  },
  effects: [
    professionMetricDeltaAll("panic_value", 0.03),
    siloMetricDelta("history_burden", 0.02),
  ],
});

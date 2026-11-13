defineEvent({
  id: "silo18_rumor_sheriff",
  title: "谣言：邻堡治安官出堡清洁",
  description: "断断续续的传闻顺着线路爬过来：相邻地堡的治安官自己要求出去，做了清洁，死在了外面。没人说得清他看到了什么。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Cross-delivered only: fired by the Silo 18 chain (N1) via the cross-queue.
  script: {
    canTrigger: function (ctx) {
      return false;
    },
  },
  effects: [
    professionMetricDeltaAll("panic_value", 0.03),
  ],
});

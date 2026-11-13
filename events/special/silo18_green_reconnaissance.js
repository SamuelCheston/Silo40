defineEvent({
  id: "silo18_green_reconnaissance",
  title: "侦察数据外泄：毒区之外有绿色",
  description: "一段无人机侦察记录经跨堡通信外泄：毒区外的山丘上出现绿色植被。记录来自外部侦察，不是公共频道播放。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Cross-delivered only: fired by the Silo 18 chain (N7) via the cross-queue.
  script: {
    canTrigger: function (ctx) {
      return false;
    },
  },
  effects: [
    professionIdeologyDeltaAll("pro_foreign", 0.08),
    professionMetricDeltaAll("panic_value", -0.04),
    siloMetricDelta("cohesion", 0.02),
  ],
});

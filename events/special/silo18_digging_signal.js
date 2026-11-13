defineEvent({
  id: "silo18_digging_signal",
  title: "IT 信号：地层掘进震动",
  description: "深层传感器记录到规律的地层震动，线路阻抗出现异动——有什么东西正在山体里横向掘进。方向不是朝着你们，但没有人睡得着。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Cross-delivered only: fired by the Silo 18 chain (N6) via the cross-queue.
  script: {
    canTrigger: function (ctx) {
      return false;
    },
  },
  effects: [
    professionMetricDeltaAll("panic_value", 0.03),
    siloMetricDelta("event_trigger", 0.05),
  ],
});

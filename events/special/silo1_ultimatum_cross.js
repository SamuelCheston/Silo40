defineEvent({
  id: "silo1_ultimatum_cross",
  title: "一号地堡通牒",
  description: "一条不容置疑的指令穿过底层网络：叛乱必须停止，否则将执行重置。这不是请求，而是倒计时。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Cross-delivered only: fired by the Silo 1 state machine via the cross-queue.
  script: {
    canTrigger: function (ctx) {
      return false;
    },
  },
  effects: [
    professionMetricDeltaAll("panic_value", 0.08),
    siloMetricDelta("rebellion", -0.05),
    siloMetricDelta("history_burden", 0.03),
  ],
});

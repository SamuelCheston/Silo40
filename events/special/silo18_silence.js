defineEvent({
  id: "silo18_silence",
  title: "邻堡静默",
  description: "邻堡的所有信道在同一时刻归于死寂。最后的残片里只有暴乱的杂音，然后是彻底的静默。有未经证实的消息说：最后一批人经隧道去了 17 号。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Cross-delivered only: fired by the Silo 18 chain (N8) via the cross-queue.
  script: {
    canTrigger: function (ctx) {
      return false;
    },
  },
  effects: [
    professionMetricDeltaAll("panic_value", 0.05),
    siloMetricDelta("history_burden", 0.03),
  ],
});

defineEvent({
  id: "silo18_refuser_crossed",
  title: "强信号：拒洁者走出了视野",
  description: "邻堡有人拒绝清洁，翻过山脊走进了毒雾深处——没有倒下。\"有人走出去却没死\"，这句话像电流一样穿过每一条线路。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Cross-delivered only: fired by the Silo 18 chain (N3) via the cross-queue.
  script: {
    canTrigger: function (ctx) {
      return false;
    },
  },
  effects: [
    professionIdeologyDeltaAll("pro_foreign", 0.05),
    professionMetricDeltaAll("panic_value", 0.03),
  ],
});

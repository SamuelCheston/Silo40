defineEvent({
  id: "reset_omen_requisition",
  title: "异常请领",
  description: "镇静类药物与配额物资被同时请领，数量远超任何已公布的巡诊或配给计划。医务、供应与 IT 的单据互相引用，却没有一张能说明最终去向。反制的窗口已经打开。",
  type: "CRISIS",
  fire_mode: "ONCE",
  script: {
    canTrigger: function(ctx) {
      return false;
    },
  },
  effects: [
    professionMetricDeltaAll("panic_value", 0.03),
    siloMetricDelta("rebellion", 0.02),
    scheduleEvent("reset_execution", 9),
  ],
});

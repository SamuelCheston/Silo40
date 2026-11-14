defineEvent({
  id: "reset_omen_census",
  title: "异常普查",
  description: "一份与既有流程都不相同的普查请求逐层下发：名册、住所、亲属与档案接收记录被放在同一张表里。答复的人只说这是例行维护，但表格没有对应的维护编号。",
  type: "INTERNAL",
  fire_mode: "ONCE",
  // Relative Y2: a new game starts in code year 122.
  trigger: all(
    siloMetricGte("history_burden", 0.2),
    timestampAtOrAfter(123, 1),
  ),
  effects: [
    professionMetricDeltaAll("panic_value", 0.04),
    scheduleEvent("reset_omen_requisition", 3),
  ],
});

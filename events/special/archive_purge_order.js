defineEvent({
  id: "archive_purge_order",
  title: "档案清洗令",
  description: "IT 部门对觉醒的追问发动了档案清洗。追问暂时被压下去，楼梯间的惊慌与怨气却越传越远。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  trigger: siloMetricGte("history_burden", 0.12),
  effects: [
    siloMetricDelta("history_burden", -0.06),
    siloMetricDelta("rebellion", 0.05),
    professionMetricDeltaAll("panic_value", 0.06),
    scheduleEvent("purge_backlash", 4),
  ],
});

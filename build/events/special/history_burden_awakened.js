defineEvent({
  id: "history_burden_awakened",
  title: "历史包袱觉醒",
  description: "封存档案引发了持续发酵的追问，多个部门开始重新审视地堡权力结构，恐慌与离心同步上升。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  trigger: siloMetricGte("history_burden", 0.08),
  effects: [
    professionMetricDeltaAll("panic_value", 0.04),
    siloMetricDelta("cohesion", -0.02),
  ],
});

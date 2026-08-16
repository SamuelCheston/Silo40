defineEvent({
  id: "water_leak",
  title: "供水管线泄漏",
  description: "底层机械部报告发生严重水管泄漏，部分楼层供水中断。",
  type: "TECHNICAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 6,
  effects: [
    siloResourceDelta("Supplies", -500),
    siloMetricDelta("cohesion", -0.05),
  ],
});

defineEvent({
  id: "last_interlock_warming",
  title: "保障机制升温",
  description: "IT 的干预越来越多、越来越重，专线那一端的保障警告正从提醒变成倒数。本地一直以为按阈值执行的只是一套系统，但警告口径的变化让少数人开始怀疑：决定终止的权力从来不在这座堡里。那一天真到来时，从专线另一端落下来的东西，不会追问是谁下的令、又是谁执行了令。",
  type: "INTERNAL",
  fire_mode: "ONCE",
  // DEATH omen, safeguard sub-path (29-failure-endings §2.5): the other two
  // death omens (ultimatum chain, strike reconnaissance) already ship; the
  // safeguard path had none. safeguard_risk is a readable silo metric, so the
  // warning uses the existing monthly channel. Deterministic threshold.
  trigger: { type: "silo_metric_gte", metric: "safeguard_risk", value: 0.6 }, // initial calibration, TODO(balance)
  effects: [
    professionMetricDeltaAll("panic_value", 0.03), // initial calibration, TODO(balance)
  ],
});

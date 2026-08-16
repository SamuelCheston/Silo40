defineEvent({
  id: "delayed_riot",
  title: "暴乱爆发",
  description: "积压的恐慌终于爆发为一场大规模暴乱。",
  type: "CRISIS",
  fire_mode: "REPEATABLE",
  effects: [
    professionMetricDeltaAll("panic_value", 0.15),
    siloMetricDelta("legitimacy", -0.1)
  ]
});

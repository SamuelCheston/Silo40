defineEvent({
  id: "underground_coordination",
  title: "地下协调网络",
  description: "各部门的激进分子开始在维修夹层里碰头。不再是各怀怨气，而是有人画出了第一张全堡的行动图。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 6,
  // Escalation gate: once rebellion itself crosses the threshold (i.e. the
  // accumulated grievance budget is already spent), the movement snowballs
  // toward the REBELLION ending. Whether a game crosses 0.68 early enough
  // for the snowball to finish before Y130 is the intended divider.
  trigger: all(
    timestampAtOrAfter(126, 1),
    siloMetricGte("rebellion", 0.62),
  ),
  effects: [
    siloMetricDelta("rebellion", 0.10),
    professionMetricDeltaAll("panic_value", 0.04),
  ],
});

defineEvent({
  id: "grief_release_ritual",
  title: "灾后互助",
  description: "灾难过后，楼层之间自发组织起互助与悼念。人们把省下的口粮递给隔壁车间——创伤没有消失，但被分摊了。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 9,
  // Relief channel: fires as a followup to resource crises instead of
  // escalating them (the inverse of the usual crisis -> escalation chain).
  // blackout_cascade fires via the delay queue; the nil-sink guard
  // (51d312f) makes subscribing to it safe, and the real-content regression
  // test TestDelayedDispatchFollowupDoesNotCrash locks that path.
  trigger: any(
    eventTriggered("crisis", "food_poisoning"),
    eventTriggered("crisis", "water_leak"),
    eventTriggered("crisis", "blackout_cascade"),
  ),
  effects: [
    professionMetricDeltaAll("panic_value", -0.08),
    siloMetricDelta("rebellion", -0.02),
  ],
});

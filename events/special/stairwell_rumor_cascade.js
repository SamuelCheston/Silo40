defineEvent({
  id: "stairwell_rumor_cascade",
  title: "楼梯间流言网",
  description: "当恐慌在太多楼层同时蔓延，楼梯间自己长出了一张流言网。事故、停电、清洗——碎片被拼成同一个故事：有人在隐瞒。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // Causal chain: rumors need a narrative seed (the awakened burden) plus
  // broad panic before they crystallize into a network.
  trigger: all(
    eventTriggered("special", "history_burden_awakened"),
    professionMetricGteCount("panic_value", 0.6, 5),
  ),
  effects: [
    siloMetricDelta("rebellion", 0.04),
    siloMetricDelta("history_burden", 0.03),
    professionMetricDeltaAll("panic_value", 0.03),
  ],
});

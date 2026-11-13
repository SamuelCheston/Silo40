defineEvent({
  id: "public_cleaning_sentence",
  title: "公开清洁判决",
  description: "当局对一名公开说'想出去'的居民执行了清洁判决，并让全堡围观。信条被重申了——用一具很快就会被风沙盖住的躯体。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  trigger: all(
    timestampAtOrAfter(124, 1),
    professionIdeologyGteCount("pro_foreign", 0.55, 2),
  ),
  effects: [
    professionMetricDeltaAll("panic_value", -0.06),
    siloMetricDelta("rebellion", 0.06),
    siloMetricDelta("history_burden", 0.03),
    scheduleEvent("cleaning_witness_unrest", 5),
  ],
});

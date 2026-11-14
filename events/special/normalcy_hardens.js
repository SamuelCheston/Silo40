defineEvent({
  id: "normalcy_hardens",
  title: "常态硬化",
  description: "又是一年。封闭忠诚已经在公共表达里取得主导：会上没有人再提那件事，走廊里的旧话题有了固定的官方版本，新来的孩子把它当作从来如此的东西接受。没有谁在镇压——常态本身就是驻军。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 12, // initial calibration, TODO(balance)
  // EXCLUSIONIST omen (17-chain-exclusionist §1, 29-failure-endings §2): the
  // readable heartbeat of the inertial slide — from relative Y3 (code 125) on,
  // whenever at least four departments' cohort-derived loyalty stands at or
  // above the public-dominance bar. Every fire says "another year without
  // change"; the count itself is the narrative. Deterministic trigger.
  trigger: all(
    timestampAtOrAfter(125, 1),
    professionIdeologyGteCount("loyalty", 0.6, 4) // initial calibration, TODO(balance)
  ),
  effects: [
    // History is being normalized: the burden settles, the edge dulls.
    siloMetricDelta("history_burden", -0.02), // initial calibration, TODO(balance)
    siloMetricDelta("rebellion", -0.02),      // initial calibration, TODO(balance)
  ],
});

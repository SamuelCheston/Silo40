defineEvent({
  id: "mayor_crisis_stage",
  title: "危机给了市长舞台",
  description: "一场危机正在展开。对 Mayor 来说这是一个合法的舞台：居民看着他，部门等着协调，命令链第一次需要有人公开负责。玩家的选择不是制造危机，而是决定——推他提前对抗，还是等舞台自己成熟。预期是后者：危机可以被担当消费，也可以先把驻地压垮。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 9, // initial calibration, TODO(balance)
  // S-1 (30-nodes §3, 31-spec §1): crises give the Mayor a legitimate stage
  // (choice P-C). Opportunity window, not a countdown — the same crisis is
  // also fuel for FRACTURE and cover for RESET; no exclusion gate. The any()
  // mixes event_triggered with a numeric branch (safe since 硬伤 #2).
  trigger: any(
    eventTriggered("crisis", "food_poisoning"),
    eventTriggered("crisis", "water_leak"),
    eventTriggered("special", "energy_rolling_blackout"),
    siloMetricGte("rebellion", 0.3) // initial calibration, TODO(balance)
  ),
  effects: [
    professionMetricDeltaAll("panic_value", 0.03), // initial calibration, TODO(balance)
  ],
});

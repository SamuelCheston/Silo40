defineEvent({
  id: "identity_trace_accumulates",
  title: "身份痕迹累积",
  description: "又一次深度行动结束了。纸面上多了一处与过去对不上的接缝：某年没有记录，某段履历来源含糊，某次在场解释不清。痕迹在开局时无关紧要——它们只是在那里，等着被某个认真的人连起来。",
  type: "INTERNAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 2, // initial calibration, TODO(balance)
  // AGENT_COMPROMISED omen, tier 1 (20-chain-agent-compromised §1): every
  // covert operation leaves a trace on the paper past. The completion keys
  // are the built-in operation action IDs (uppercase). The tiered readings
  // (3rd fire = local-law-enforcement attention, 5th = local-IT archive
  // review) are carried by the follower nodes, which read this key's
  // trigger_count. Deterministic: no random source.
  trigger: any(
    eventTriggered("player_action", "STATION_INFILTRATE"),
    eventTriggered("player_action", "STATION_DECEIVE"),
    eventTriggered("player_action", "SILO1_DECEIVE")
  ),
  effects: [
    professionMetricDeltaAll("panic_value", 0.02), // initial calibration, TODO(balance)
  ],
});

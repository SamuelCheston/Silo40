defineEvent({
  id: "replication_progress",
  title: "复制推进",
  description: "流通过的材料有了后续的公开记录：有些部门已经能够谈起副本和它的来处。历史包袱也多沉下一分。",
  type: "INTERNAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 3,
  // R-1 (24-spec): every completed Share Info keeps the replication heartbeat
  // alive; the REPEATABLE cooldown makes the count grow at most once per 3
  // months (continuous evaluation), which L-2 reads as the replication
  // milestone. Deterministic: no random source in this definition.
  trigger: eventTriggered("player_action", "action_share_info"),
  effects: [
    // profession/fragment empty: engine picks the least-fragment profession
    // (deterministic ID tie-break) and draws from the seeded engine RNG
    // stream (byte-reproducible per seed).
    professionFragmentAdd(),
    siloMetricDelta("history_burden", 0.02), // initial calibration, TODO(balance)
    // R-2 (merged): register the replication fact into the public evidence
    // registry. Re-registration of the same fact only raises fields (§3.7
    // dedupe), so repeated fires deepen provenance/coverage instead of
    // duplicating records. Provenance 0.6 = public testimony / unverified
    // records; coverage initial calibration, TODO(balance).
    publicEvidenceRegister("fact:info_chain_replication", 0.6, 0.1),
  ],
});

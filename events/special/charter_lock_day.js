defineEvent({
  id: "charter_lock_day",
  title: "锁定日",
  description: "司法部在公开证据与独立事实的支持下宣告附着公开约章。IT 的反扑压缩着窗口：选择性执法、收缴证物、指控约章支持者。你用剩余的关系与威望为签署程序争取那一段时间——证据托管、证人到场、程序完整。约章的价值恰在于它不能被单方收回，而让'不可逆'成立的这几分钟，是你亲自守下来的。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // K-1 (26-nodes §4, 27-spec §1): the lock-day decision beat (choice L-D
  // projection). The lock itself is NOT an effect of this node: PactLocked is
  // written by the engine's pact dynamics once the open window (W <= 18,
  // Q >= 0.85, >= 2 independent facts) sustains; qualification reads that
  // engine fact, never this completion key (ADR-0010).
  //
  // DEPENDENCY NOTE: reaching W <= 18 currently relies on the exposure
  // high-water mark accumulating naturally; the discrete negative-W shock
  // channels (Silo 1 fall / termination-chain failure / mass public evidence)
  // are being built on a parallel track. Content never writes W and invents
  // no W judgment of its own — this node waits for the window, nothing more.
  trigger: all(
    eventTriggered("special", "charter_draft_convened"),
    timestampAtOrAfter(127, 1) // relative Y5 = code 127, initial calibration, TODO(balance)
  ),
  effects: [
    siloMetricDelta("rebellion", -0.04),               // initial calibration, TODO(balance)
    professionMetricDeltaAll("panic_value", -0.04),    // initial calibration, TODO(balance)
  ],
  script: {
    canTrigger: function(ctx) {
      var a = ctx.authority || {};
      // Mirror the engine's PactOpenW / PactMinQ / PactMinFacts bars; if the
      // engine config moves, these reads move with it (TODO(balance) pair).
      return a.w <= 18 &&
        a.public_evidence_q >= 0.85 &&
        a.independent_public_facts >= 2;
    },
  },
});

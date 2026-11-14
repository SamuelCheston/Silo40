defineEvent({
  id: "independent_receivers_met",
  title: "独立接收者达标",
  description: "不再只有你在数这些材料了。至少有两个来源不同的本地人能够独立说明证据来自哪里、哪部分能核验、哪部分仍有缺口——并且不打算等你发话才继续核对。",
  type: "INTERNAL",
  fire_mode: "ONCE",
  // L-1 (24-spec): registration-count proxy. The narrative definition of an
  // independent receiver (explains provenance / keeps without prompting /
  // owes no allegiance) is not mechanically checkable; the proxy counts
  // distinct registered public facts. Initial calibration: >= 2 distinct
  // facts (replication fact + verification fact), TODO(balance).
  trigger: timestampAtOrAfter(122, 1),
  script: {
    canTrigger: function(ctx) {
      var ev = ctx.public_evidence || [];
      return ev.length >= 2;
    },
  },
  effects: [
    // The existence of independent receivers is itself a public fact
    // (provenance 0.6 = testimony tier). It becomes the third registered
    // fact that P-1's milestone counts.
    publicEvidenceRegister("fact:independent_receivers", 0.6, 0.1), // coverage TODO(balance)
  ],
});

defineEvent({
  id: "public_fact_cross_dept",
  title: "公共事实跨部门",
  description: "这件事不再属于某些人。它开始出现在你不在场的对话里，被引用、被核对、被当作谈话的前提——真相站到了驻地中央。",
  type: "INTERNAL",
  fire_mode: "ONCE",
  // P-1 (24-spec): narrative projection of the INFORMATION qualification.
  // The ending qualification itself is read by arbitration (fragment
  // coverage); this node marks the moment the registered public evidence
  // crosses the department-boundary milestone AND the charter gate sees a
  // positive Q (Judicial belief layer x registry).
  trigger: timestampAtOrAfter(122, 1),
  script: {
    canTrigger: function(ctx) {
      var ev = ctx.public_evidence || [];
      // Milestone tier, higher than L-1's threshold. Initial calibration,
      // TODO(balance).
      if (ev.length < 3) {
        return false;
      }
      var auth = ctx.authority || {};
      return (auth.public_evidence_q || 0) > 0;
    },
  },
  effects: [
    siloMetricDelta("history_burden", 0.03), // initial calibration, TODO(balance)
  ],
});

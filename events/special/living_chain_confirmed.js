defineEvent({
  id: "living_chain_confirmed",
  title: "活链确认",
  description: "接收者之间开始自行对账、自行纠错、自行安排下一轮核对。没有人来找你请示——链已经不需要你逐日维持了。",
  type: "INTERNAL",
  fire_mode: "ONCE",
  // L-2 (24-spec): terminal achievement of the local chain.
  // Declarative part: independent receivers met + rebellion kept under the
  // threshold that feeds Silo 1's reconnaissance/purge chains (concealment
  // proxy; the exact E threshold waits for E readability, 24-spec §3 C-1).
  trigger: all(
    eventTriggered("special", "independent_receivers_met"),
    siloMetricLte("rebellion", 0.8) // initial calibration, TODO(balance)
  ),
  script: {
    canTrigger: function(ctx) {
      // Replication milestone: replication_progress must have fired through
      // the qualitative-change tier (initial calibration 4 fires ~= 12
      // months of heartbeat), TODO(balance).
      var rep = ctx.states["special:replication_progress"];
      if (!rep || (rep.trigger_count || 0) < 4) {
        return false;
      }
      // Concealment proxy: the local purge must not have executed.
      // DEPENDENCY NOTE: true concealment reads the exposure high-water mark
      // E, which content cannot read yet (24-spec §3 C-1). Until that
      // mechanism lands, the rebellion threshold above plus this purge check
      // stand in.
      var purge = ctx.states["crisis:silo1_purge_execution"];
      if (purge && purge.triggered) {
        return false;
      }
      return true;
    },
    // The living chain is a fact, not a numeric shift: no declarative
    // effects. Extinguish semantics (living chain lost when upkeep fails)
    // wait for completion-key reversibility (24-spec §3); until then this
    // node only ever lights, and L-3 upkeep failure is expressed by the
    // recirculation node going dark, not by un-lighting this one.
    apply: function(ctx) {
      return { effects: [], emit: [] };
    },
  },
});

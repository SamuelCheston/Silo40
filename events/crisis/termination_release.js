defineEvent({
  id: "termination_release",
  title: "终止投放生效",
  description: "那条远程通道在深夜完成了一次完整的投放。没有爆炸，没有火光——只有通风系统里一阵说不清的甜腥味，和随后开始减少的应答声。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Scheduled-only execution node: fired via schedule_event from
  // termination_channel_test. Guard reads the network_jammed world flag (G11):
  // a disrupted channel no-ops; only an intact channel delivers.
  script: {
    canTrigger: function(ctx) {
      return false;
    },
    apply: function(ctx) {
      var flags = ctx.world_flags || {};
      if (flags["network_jammed"]) {
        // Channel was disrupted: the delivery handshake never completes.
        return {
          effects: [siloMetricDelta("history_burden", 0.02)],
        };
      }
      // Intact channel: the termination drives intervention risk to the hard
      // guardrail, which victory_check settles as 《寂堡》 (DEATH).
      // TODO(balance): magnitude is a sim-calibration starting point.
      return {
        effects: [
          siloMetricDelta("safeguard_risk", 1.0),
          professionMetricDeltaAll("panic_value", 0.1),
        ],
      };
    },
  },
  effects: [],
});

defineEvent({
  id: "identity_query_local",
  title: "本地执法复核来历",
  description: "本地执法开始复核你的来历。他们没有提到另一个堡，只把几次对不上的记录放在一起，要求你解释某年的去向、某份证明的签发人。问题仍然琐碎而例行，但已经不再是纸面上的瑕疵。",
  type: "INTERNAL",
  fire_mode: "ONCE",
  // AGENT_COMPROMISED omen, tier 2 (20-chain-agent-compromised §1): local
  // law enforcement rechecks the agent's stated past after the third trace.
  trigger: eventTriggered("special", "identity_trace_accumulates"),
  effects: [
    siloMetricDelta("rebellion", 0.02), // initial calibration, TODO(balance)
  ],
  script: {
    canTrigger: function(ctx) {
      var state = ctx.states["special:identity_trace_accumulates"];
      return !!state && state.trigger_count >= 3; // initial calibration, TODO(balance)
    },
  },
});

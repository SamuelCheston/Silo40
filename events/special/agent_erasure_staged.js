defineEvent({
  id: "agent_erasure_staged",
  title: "抹除已在准备",
  description: "一些不属于任何流程的小事同时发生了：你的住处被例行检查，你的职务被建议'暂时交接'，有人开始向你确认很早以前的细节。逮捕令还没有签发，但它的纸已经铺好了。抹除会以一个本地名义到来——来历存疑、刑事罪名，或者无声失踪——没有人会知道你真正是什么。",
  type: "INTERNAL",
  fire_mode: "ONCE",
  // AGENT_COMPROMISED omen, tier 3 (20-chain-agent-compromised §1): the
  // erasure staging reads the actor's suspicion level directly (ctx.actor
  // exposes it) and lights once, approaching the cap. Staging constraint
  // (06-agent-compromised §5): the erasure is always packaged under a local
  // pretext, never as a public trial of 'an agent from another silo'.
  trigger: eventTriggered("special", "identity_trace_accumulates"),
  effects: [
    siloMetricDelta("history_burden", 0.02), // initial calibration, TODO(balance)
  ],
  script: {
    canTrigger: function(ctx) {
      return (ctx.actor && ctx.actor.suspicion_level >= 0.7); // initial calibration, TODO(balance)
    },
  },
});

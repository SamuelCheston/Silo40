defineEvent({
  id: "archive_review_opened",
  title: "档案复查开始",
  description: "本地 IT 开始复查一批旧档案，理由是例行的数据完整性审计。你认得那份清单上的编号——你的纸面过去正在被逐页翻动。身份线与证据线开始一起燃烧：深查必见接缝，而你的整个来历是一次性写入的。",
  type: "INTERNAL",
  fire_mode: "ONCE",
  // AGENT_COMPROMISED omen, tier 2 (20-chain-agent-compromised §1): the
  // local IT archive review opens once the accumulated covert-operation
  // traces cross the fifth mark. Choice E-B projection: last strike before
  // the cap, or protective dismantling.
  trigger: eventTriggered("special", "identity_trace_accumulates"),
  effects: [
    siloMetricDelta("history_burden", 0.03), // initial calibration, TODO(balance)
  ],
  script: {
    canTrigger: function(ctx) {
      var state = ctx.states["special:identity_trace_accumulates"];
      return !!state && state.trigger_count >= 5; // initial calibration, TODO(balance)
    },
  },
});

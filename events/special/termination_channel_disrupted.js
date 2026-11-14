defineEvent({
  id: "termination_channel_disrupted",
  title: "终止信道被干扰",
  description: "你把链路自检的取证与反制要点分散递给了可信的人。当那条远程通道再次被点亮时，本地一侧的应答被有意的杂讯覆盖——投放指令没能完成握手。信道被干扰了，但谁都清楚，这只会让对方改用别的手段。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Countermeasure (《寂堡》链 21)：自检之后，玩家把反制知识跨过部门边界（以
  // Share Info 为代理），即可干扰该终止信道。只免疫远程终止这一条渠道，不免疫
  // 本地执行与物理打击（分渠道纪律）。
  trigger: eventTriggered("special", "termination_channel_test"),
  script: {
    canTrigger: function(ctx) {
      var probe = ctx.states["special:termination_channel_test"];
      var share = ctx.states["player_action:action_share_info"];
      if (!probe || !probe.triggered) return false;
      if (!share || !share.triggered) return false;
      // The counter-knowledge must actually cross hands after the probe.
      return share.last_triggered_timestamp >= probe.last_triggered_timestamp;
    },
  },
  effects: [
    worldFlagSet("network_jammed", true),
    siloMetricDelta("history_burden", 0.03),
  ],
});

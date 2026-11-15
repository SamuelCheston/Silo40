defineEvent({
  id: "termination_release",
  title: "终止投放生效",
  description: "那条远程通道在深夜完成了一次完整的投放。没有爆炸，没有火光——只有通风系统里一阵说不清的甜腥味，和随后开始减少的应答声。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // 远程终止渠道的结算节点 (《寂堡》链 21 §1/§5)：自检满 10 月后落地，且信道
  // 未被干扰。守卫写在触发条件里——`termination_channel_disrupted` 写入
  // network_jammed 后，本节点永不点火，投放空转，仲裁也读不到终止事实。
  // 物理渠道 (bombing_terminal) 不读这个 flag：分渠道纪律，反制成功不等于获救。
  // 完成键 crisis:termination_release 被仲裁读作 DEATH 事实。
  trigger: all(
    eventTriggered("special", "termination_channel_test"),
    monthsSinceEventGte("special", "termination_channel_test", 10),
    worldFlagNotSet("network_jammed")
  ),
  script: {
    apply: function(ctx) {
      // Settlement hook: no numerical consequence. The completion key itself is
      // the terminal fact; arbitration settles the ending.
      return {};
    },
  },
});

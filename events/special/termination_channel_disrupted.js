defineEvent({
  id: "termination_channel_disrupted",
  title: "终止信道被干扰",
  description: "那条远程通道再次被点亮时，本地一侧的应答被有意的杂讯覆盖——投放指令没能完成握手。信道被干扰了，但谁都清楚，这只会让对方改用别的手段。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // 远程终止渠道的反制结算 (《寂堡》链 21 §1/§5)。绑定在玩家动作
  // `termination_channel_jam` 之后：值夜时对配线架做的手脚在一个月后才见效，
  // 本节点由该动作调度，不参与常规触发扫描。分渠道纪律：只免疫远程终止，
  // 物理投送 (bombing_terminal) 不受影响。
  script: {
    canTrigger: function(ctx) {
      // Scheduled-only settlement hook: fired via schedule_event from
      // termination_channel_jam. The countermeasure itself is an everyday
      // action, not a random world tick — this node must never self-fire.
      return false;
    },
  },
  effects: [
    worldFlagSet("network_jammed", true),
    siloMetricDelta("history_burden", 0.03), // sim 校准起点
    // 裁决的冲击通道 (三类触发源之一)：终止链失效是一次结构性事件，直接
    // 造成离散负冲击 (量级缺省 = 引擎 DecisiveShockMagnitude，同源只发一次)。
    decisiveShock("termination_channel_disrupted"),
  ],
});

defineEvent({
  id: "termination_channel_disrupted",
  title: "终止信道被干扰",
  description: "那条远程通道再次被点亮时，本地一侧的应答被有意的杂讯覆盖——投放指令没能完成握手。信道被干扰了，但谁都清楚，这只会让对方改用别的手段。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // DECLARED-BUT-UNBOUND countermeasure hook (remote-termination channel).
  // Ruling: Share Info is NOT a countermeasure and must not act as one, so the
  // former share-info gate is removed. The real countermeasure will later be
  // hidden inside everyday small actions (contextualized actions with an object,
  // place, timing, material and a trace). Until that system lands this node is
  // intentionally unbound: canTrigger returns false, so it never fires yet.
  //
  // TODO(Countermeasure binding): wire canTrigger to the future everyday-action
  // countermeasure; on success the effect below jams the termination channel.
  trigger: eventTriggered("special", "termination_channel_test"),
  script: {
    canTrigger: function(ctx) {
      // Unbound: no countermeasure action is wired this round.
      return false;
    },
  },
  effects: [
    worldFlagSet("network_jammed", true),
    // 裁决的冲击通道 (三类触发源之一)：终止链失效是一次结构性事件，直接
    // 造成离散负冲击 (量级缺省 = 引擎 DecisiveShockMagnitude，同源只发一次)。
    decisiveShock("termination_channel_disrupted"),
  ],
});

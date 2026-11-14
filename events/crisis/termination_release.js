defineEvent({
  id: "termination_release",
  title: "终止投放生效",
  description: "那条远程通道在深夜完成了一次完整的投放。没有爆炸，没有火光——只有通风系统里一阵说不清的甜腥味，和随后开始减少的应答声。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Scheduled-only settlement hook: fired via schedule_event from
  // termination_channel_test. Ruling: the settlement writes NO safeguard-risk
  // delta — it connects directly to the ending arbitration. Its completion key
  // crisis:termination_release is read there as a DEATH fact (《寂堡》).
  //
  // TODO(Countermeasure binding): the countermeasure guard belongs here. When
  // the countermeasure-in-everyday-actions system lands, this apply must check
  // the disrupted-channel fact (e.g. network_jammed) and settle as a no-op when
  // the delivery handshake was broken. This round ships no countermeasure, so
  // the hook is unconditional.
  script: {
    canTrigger: function(ctx) {
      return false;
    },
    apply: function(ctx) {
      // Settlement hook: no numerical consequence. The completion key itself is
      // the terminal fact; arbitration settles the ending.
      return {};
    },
  },
});

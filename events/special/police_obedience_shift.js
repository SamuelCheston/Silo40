defineEvent({
  id: "police_obedience_shift",
  title: "警察转向",
  description: "变化是从一个警员开始的：一道照旧下发的调度单被拿回去问了一句'这命令合法吗'。然后是第二个人、第三个人。警察开始有选择地执行——不是抗命，是服从换了一个对象。司法部从不归市长，警察才是人治秩序真正的承接者：他们被一连串公开担当，一个月一个月地拉了过去。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // S-3 (30-nodes §3, 31-spec §1): the milestone read of the line's core
  // drama — police obedience crossing the qualification bar after the first
  // stand. Mirrors the engine's PersonalRuleMinObedience; if the engine bar
  // moves, this read moves with it (TODO(balance) pair). Pure milestone: no
  // mechanical effect, qualification stays an engine fact.
  trigger: eventTriggered("special", "mayor_public_stand"),
  script: {
    canTrigger: function(ctx) {
      var a = ctx.authority || {};
      return a.police_obedience >= 0.60; // initial calibration, TODO(balance)
    },
    apply: function(ctx) {
      return { effects: [], emit: [] };
    },
  },
});

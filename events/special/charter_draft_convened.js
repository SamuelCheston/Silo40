defineEvent({
  id: "charter_draft_convened",
  title: "起草会议召开",
  description: "约章草案第一次被完整摊在桌面上：哪些命令不可执行、裁决依据什么、证据由谁保管。条款里有妥协，有缺陷，有几处你一眼就能看出将来会吵架的缝隙——但每一个字都出自本地人之口。你在旁听席，不在起草席。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // D-2 (26-nodes §3, 27-spec §1): the drafting convention convenes once the
  // step heartbeat has advanced twice (choice L-C projection: the charter is
  // drafted by locals, the player supplies evidence and never the pen).
  trigger: eventTriggered("special", "charter_draft_step"),
  effects: [
    siloMetricDelta("history_burden", 0.03), // initial calibration, TODO(balance)
  ],
  script: {
    canTrigger: function(ctx) {
      var state = ctx.states["special:charter_draft_step"];
      return !!state && state.trigger_count >= 2; // initial calibration, TODO(balance)
    },
  },
});

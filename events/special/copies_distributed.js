defineEvent({
  id: "copies_distributed",
  title: "副本离手",
  description: "材料经过了多次分享，分发托管的反制已经就位。档案室之外终于有人接过材料，清洗队来时，抵抗有了依托。",
  type: "INTERNAL",
  fire_mode: "ONCE",
  trigger: all(
    eventTriggered("player_action", "RESET_COUNTERMEASURE_OFFER"),
    eventTriggered("player_action", "action_share_info"),
  ),
  script: {
    canTrigger: function(ctx) {
      var shares = ctx.states["player_action:action_share_info"];
      return !!(shares && shares.triggered && (shares.trigger_count || 0) >= 2);
    },
  },
  effects: [
    professionFragmentAdd(),
    professionFragmentAdd(),
    completePlayerAction("reset_countermeasure_done"),
  ],
});

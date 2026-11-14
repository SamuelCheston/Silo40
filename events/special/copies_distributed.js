defineEvent({
  id: "copies_distributed",
  title: "副本离手",
  description: "第二批副本已经进入另一组接收者手中。它们不依赖同一间档案室，也不依赖同一个人继续记得；清洗名单即使收走你手上的材料，也无法同时收回这些副本。",
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

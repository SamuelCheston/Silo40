defineEvent({
  id: "silo1_purge_exposed",
  title: "整肃图谋被揭穿",
  description: "你把请领清单、名册调阅与深夜清点的对应关系公开讲了出来。当越来越多的人开始互相转告、拒绝单独落单时，那场整肃还没动手就失去了突然性。执行者发现，他们再也无法悄悄把人带走。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Countermeasure (本地执行链)：征兆之后，玩家用公开宣传把整肃图谋讲破，使
  // 本地执行链失去突然性。只免疫本地整肃这一条渠道，不免疫远程终止与物理投送。
  trigger: eventTriggered("special", "silo1_purge_omen"),
  script: {
    canTrigger: function(ctx) {
      var omen = ctx.states["special:silo1_purge_omen"];
      var propaganda = ctx.states["player_action:action_conduct_propaganda"];
      if (!omen || !omen.triggered) return false;
      if (!propaganda || !propaganda.triggered) return false;
      return propaganda.last_triggered_timestamp >= omen.last_triggered_timestamp;
    },
  },
  effects: [
    siloMetricDelta("rebellion", 0.03),
    siloMetricDelta("history_burden", 0.02),
  ],
});

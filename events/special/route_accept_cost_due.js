defineEvent({
  id: "route_accept_cost_due",
  title: "路线的账",
  description: "五个月了。勘察队回来过，或者根本没回来过——名册上的名字和检修出口的记录对不对得上，账自己会说话。路线是事实，事实要人一步步走出来；没人走的路，只剩传闻，而传闻比闸门更容易走漏风声。",
  type: "TECHNICAL",
  fire_mode: "ONCE",
  // 延迟回访（接受键 +5 月，TODO(balance)）：按勘察心跳 route_survey_step
  // 是否走过分流——实地丈量过，代价是已经付过的动静；接了委托却没去，
  // 空头的路线传闻喂恐慌。
  trigger: all(
    eventTriggered("player_action", "ROUTE_SURVEY_ACCEPT"),
    monthsSinceEventGte("player_action", "ROUTE_SURVEY_ACCEPT", 5)
  ),
  script: {
    apply: function(ctx) {
      var step = ctx.states["special:route_survey_step"];
      if (step && step.triggered) {
        // 已勘察：路线成了事实，余波是一点压不住的动静
        // (initial calibration, TODO(balance))。
        return {
          effects: [siloMetricDelta("rebellion", 0.01)],
        };
      }
      // 没去：委托悬在半空，传闻替路线先走了一步
      // (initial calibration, TODO(balance))。
      return {
        effects: [professionMetricDeltaAll("panic_value", 0.02)],
      };
    },
  },
});
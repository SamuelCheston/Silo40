defineEvent({
  id: "route_survey_defer",
  title: "暂缓：先把别的事做完",
  description: "'给我一点时间。'他收起图纸：'时间也是成本。外面不会等我们准备好。'你们收起这次接触的记录，勘察没有开始。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 暂缓键：路线未推进，保留磋商余地。"时间也是成本"由 route_second_window
  // （+4 月）兑现——拖延的暴露记入准备账本，并解锁 ROUTE_SURVEY_REACCEPT。
  player_action: {
    id: "ROUTE_SURVEY_DEFER",
    label: "暂缓勘察",
    description: "暂搁勘察提议，结束本次接触。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 0,
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ROUTE_SURVEY_DEFER"),
  script: {
    apply: function(ctx) {
      return { effects: [], emit: [] };
    },
  },
});

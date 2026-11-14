defineEvent({
  id: "route_survey_defer",
  title: "暂缓：先把别的事做完",
  description: "'给我一点时间。'他收起图纸：'时间也是成本。外面不会等我们准备好。'暂缓不是拒绝——是把勘察排进了更长的清单。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 暂缓键：与拒绝同效（路线未推进），保留磋商余地。门控无否定谓词，
  // 三选一后兄弟选择不能隐藏（与《公议》《人治》共用缺口）。
  player_action: {
    id: "ROUTE_SURVEY_DEFER",
    label: "暂缓勘察",
    description: "推迟路线勘察，先处理其他事务。",
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

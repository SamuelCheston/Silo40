defineEvent({
  id: "route_survey_defer",
  title: "暂缓：先把别的事做完",
  description: "'给我一点时间。'他收起图纸：'时间也是成本。外面不会等我们准备好。'你们收起这次接触的记录，勘察没有开始——但等待的每一月都在记暴露，重开的窗口会比上次窄。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 暂缓键：路线未推进，保留磋商余地。拖延的暴露分两段记账：① 即时——
  // 暂缓当下记入准备账本（prep_exposure +0.01，等待短于拒绝故更轻）；
  // ② 延迟——route_second_window（+4 月）再记一笔并解锁 ROUTE_SURVEY_REACCEPT。
  // 可绕行：第二扇窗重开选择，不是死胡同。
  player_action: {
    id: "ROUTE_SURVEY_DEFER",
    label: "暂缓勘察",
    description: "暂缓勘察。窗口保留但收窄；拖延的暴露即刻记账，重开时代价更高。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 0,
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ROUTE_SURVEY_DEFER"),
  effects: [
    // 追踪档位上升（即时）：暂缓即开始等待，暴露即刻记账（短于拒绝）。
    // TODO(balance): 0.01 为暂缓即时暴露；+4 月第二扇窗另记 0.03。
    preparationDelta("prep_exposure", 0.01),
  ],
});

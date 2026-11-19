defineEvent({
  id: "route_survey_reaccept",
  title: "再接受：挑更险的时辰重新丈量",
  description: "你重新点头。他摊开图纸：'暗号换了一套——这段时间，外面的耳朵多了。'勘察从检修出口重新开始，路线的事实要重新一步步丈量。这一次你清楚，等待不是免费的。",
  type: "TECHNICAL",
  fire_mode: "ONCE",
  // 第二扇窗的接受分支（route_second_window 解锁）：新完成键重开勘察线，
  // route_survey_step 同时订阅两个接受键。AP 更贵是拖延的代价之一。
  player_action: {
    id: "ROUTE_SURVEY_REACCEPT",
    label: "重新委托勘察",
    description: "在收窄的窗口里重新委托路线勘察（代价更高）。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // initial calibration, TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ROUTE_SURVEY_REACCEPT"),
  effects: [
    siloMetricDelta("rebellion", 0.01), // initial calibration, TODO(balance)
  ],
});
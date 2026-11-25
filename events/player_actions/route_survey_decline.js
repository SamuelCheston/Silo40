defineEvent({
  id: "route_survey_decline",
  title: "拒绝：现在还不是时候",
  description: "你摇头。他并不意外：'路线不会自己长出来。那今天就谈到这里。'接触就此打住，但想走的人仍在暗处互相打量——每一次打量都是一次暴露，拖延的账从这一刻开始记。窗口还开着，只是重开时会比上次窄。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 拒绝键：路线方向暂不推进。拖延的暴露分两段记账：① 即时——拒绝当下
  // 就把等待的暴露记入准备账本（prep_exposure +0.02，喂出发追踪基数）；
  // ② 延迟——route_second_window（+8 月）再记一笔并解锁 ROUTE_SURVEY_REACCEPT
  // 新完成键（AP 2，窗口更窄）。可绕行：第二扇窗重开选择，不是死胡同。
  player_action: {
    id: "ROUTE_SURVEY_DECLINE",
    label: "拒绝勘察",
    description: "拒绝勘察。路线勘察就此搁置；拖延的暴露即刻记账，日后重开窗口更窄、代价更高。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 0,
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ROUTE_SURVEY_DECLINE"),
  effects: [
    // 追踪档位上升（即时）：拒绝让接触转入暗处等待，暴露即刻记账。
    // TODO(balance): 0.02 为拒绝即时暴露；+8 月第二扇窗另记 0.03。
    preparationDelta("prep_exposure", 0.02),
  ],
});

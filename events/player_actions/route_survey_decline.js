defineEvent({
  id: "route_survey_decline",
  title: "拒绝：现在还不是时候",
  description: "你摇头。他并不意外：'路线不会自己长出来。那今天就谈到这里。'接触就此打住，想走的人仍在暗处互相打量。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 拒绝键：路线方向暂不推进。"我还在这儿"由 route_second_window（+8 月）
  // 兑现——拖延的暴露记入准备账本，并解锁 ROUTE_SURVEY_REACCEPT 新完成键。
  player_action: {
    id: "ROUTE_SURVEY_DECLINE",
    label: "拒绝勘察",
    description: "谢绝本次勘察提议，保留接触记录。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 0,
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ROUTE_SURVEY_DECLINE"),
  script: {
    apply: function(ctx) {
      return { effects: [], emit: [] };
    },
  },
});

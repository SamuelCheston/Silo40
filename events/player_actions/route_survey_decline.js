defineEvent({
  id: "route_survey_decline",
  title: "拒绝：现在还不是时候",
  description: "你摇头。他并不意外：'路线不会自己长出来。你改主意的时候，我还在这儿。'接触就此打住，想走的人仍在暗处互相打量。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 拒绝键：路线方向暂不推进；意愿事实仍在，后续可经其他路线事实接续。
  player_action: {
    id: "ROUTE_SURVEY_DECLINE",
    label: "拒绝勘察",
    description: "暂不委托路线勘察，保留接触记录。",
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

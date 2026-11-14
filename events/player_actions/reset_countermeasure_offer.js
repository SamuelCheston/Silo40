defineEvent({
  id: "reset_countermeasure_offer",
  title: "分发托管",
  description: "你接下这项反制：把材料副本交给不同部门的接收者，让任何一次封档都无法把它们同时抹掉。接下来需要真实的分享与接收，而不是一份留在你手里的计划。",
  type: "INTERNAL",
  fire_mode: "ONCE",
  player_action: {
    id: "RESET_COUNTERMEASURE_OFFER",
    label: "组织副本托管",
    description: "接受分发托管反制，通过既有分享行动把副本交给多个接收者。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 1,
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("RESET_COUNTERMEASURE_OFFER"),
  script: {
    apply: function(ctx) {
      return { effects: [], emit: [] };
    },
  },
});

defineEvent({
  id: "legitimacy_survey",
  title: "接触：调查居民实际信任谁",
  description: "Mayor 身边的联系人同意和你谈谈。你调查的不是权力黑箱，而是居民实际信任谁：过去危机里的记录，命令在职位之外是否被执行，他是否为驻地承担过个人代价。结论很清楚——他是唯一可见的替代权力，但他不知道暗权网络，更不知道 Silo 1。他能做对的事，却不知道自己对抗的是什么。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // M-1 (30-nodes §1, 31-spec §1): standard contact card (action design 15
  // §5.3). The gate (action_gates.json) unlocks it at first_signal with a
  // minimum Mayor relation; completing the offer unlocks the P-A choice.
  // Pure contact node: no mechanical effect.
  player_action: {
    id: "LEGITIMACY_SURVEY",
    label: "调查正统性",
    description: "接触 Mayor 身边的联系人，调查居民实际信任谁。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 1,            // initial calibration, TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("LEGITIMACY_SURVEY"),
  script: {
    apply: function(ctx) {
      return { effects: [], emit: [] };
    },
  },
});

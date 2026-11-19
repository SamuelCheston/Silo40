defineEvent({
  id: "mayor_evidence_step",
  title: "喂证：责任链材料进 Mayor 圈层",
  description: "又一轮流通把责任链材料带进了 Mayor 圈层：签发栏的空白、编号的断档、被绕过的正式渠道。他不问这些东西最终指向谁——你也没说。他只需要知道哪些命令不合法，然后以本地责任接管暗权留下的职能空缺。他的声望随着每一份站得住的材料往上走。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 3, // initial calibration, TODO(balance)
  // F-1 (30-nodes §2, 31-spec §1): feeding heartbeat, same wiring as
  // charter_draft_step / replication_progress. Lives in special/ on purpose:
  // a player_actions/ file firing on Share Info would intercept the share's
  // own resolution. Choice P-B projection: the "hand it to the Mayor" branch
  // is mechanized; the charter-decline reroute (30-nodes §6) also feeds this
  // heartbeat — refused charter materials flow into the Mayor circle.
  trigger: all(
    any(
      eventTriggered("player_action", "mayor_ally_accept"),
      eventTriggered("player_action", "charter_contact_decline"),
      eventTriggered("player_action", "charter_second_decline")
    ),
    eventTriggered("player_action", "action_share_info")
  ),
  effects: [
    // Legitimacy is a persisted state (metrics_update no longer resets it):
    // the delta accumulates, and the next monthly authority advance reads it —
    // police obedience converges toward legitimacy at the engine response
    // rate. Each feeding step is one month of obedience pull plus a lasting
    // +0.02 to the mayor's personal authority (30-nodes §5).
    siloMetricDelta("legitimacy", 0.02), // initial calibration, TODO(balance)
  ],
});

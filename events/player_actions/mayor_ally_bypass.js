defineEvent({
  id: "mayor_ally_bypass",
  title: "绕过：他不知道真相，所以不可用",
  description: "你决定不碰这条线：他不知道真相，不可用。驻地从此没有一个能公开承接秩序的本地人——你转入纯秘密路线，材料留在自己手里。追问秩序的声音从这一刻起开始冷却，常态继续硬化，那是另一条线的征兆了。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // M-2 bypass (30-nodes §1, §7): closes the line for this run (the contact
  // key is consumed). Landing: the station drifts toward EXCLUSIONIST inertia.
  // Two beats: ① immediate — the public-order vacuum starts cooling the
  //   question now (history_burden −0.01, normalcy hardens from this moment);
  //   ② delayed — mayor_bypass_drift (+6 months) echoes with a one-time
  //   inertial settle (normalcy_hardens keeps firing on its own clock).
  // No reroute back into this line; other endings stay reachable. Not a trap.
  player_action: {
    id: "MAYOR_ALLY_BYPASS",
    label: "绕过 Mayor",
    description: "绕过市长。人治同盟线永久关闭；驻地失去公开承接秩序的人，常态硬化即刻开始，滑向深闭。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 0,            // initial calibration, TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("MAYOR_ALLY_BYPASS"),
  effects: [
    // 常态硬化（即时）：失去公开承接秩序的人，追问开始冷却，历史包袱小幅松绑。
    // TODO(balance): −0.01 为即时冷却；+6 月 mayor_bypass_drift 另记惯性漂移。
    siloMetricDelta("history_burden", -0.01),
  ],
});

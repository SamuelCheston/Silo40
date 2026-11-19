defineEvent({
  id: "mayor_ally_bypass",
  title: "绕过：他不知道真相，所以不可用",
  description: "你决定不碰这条线：他不知道真相，不可用。驻地从此没有一个能公开承接秩序的本地人——你转入纯秘密路线，材料留在自己手里。常态会继续硬化，那是另一条线的征兆了。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // M-2 bypass (30-nodes §1, §7): closes the line for this run (the contact
  // key is consumed). Landing: the station drifts toward EXCLUSIONIST inertia
  // (normalcy_hardens keeps firing); mayor_bypass_drift (+6 months) echoes the
  // bypass with a one-time inertial settle. No reroute back into this line.
  player_action: {
    id: "MAYOR_ALLY_BYPASS",
    label: "绕过 Mayor",
    description: "不支持 Mayor；本线关闭，转入纯秘密路线。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 0,            // initial calibration, TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("MAYOR_ALLY_BYPASS"),
  script: {
    apply: function(ctx) {
      return { effects: [], emit: [] };
    },
  },
});

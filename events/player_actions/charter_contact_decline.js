defineEvent({
  id: "charter_contact_decline",
  title: "拒绝：材料另寻出路",
  description: "你收回了材料。他点点头，没有追问：'规程这东西，等有人想要的时候才值钱。'制度内的这条门就此关上，本局没有第二次约章接触——但断点材料没有消失，抄本已经在市长圈层里传阅，变成换人的筹码，而不是约束命令的规则。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // T-2 decline (26-nodes §2, 27-spec §1): closes the charter line for this
  // run (the contact key is consumed; no second charter contact). The reroute
  // of the materials into Mayor circles (30-nodes §6) lands in two beats:
  //   ① immediate — the breakpoint copies enter Mayor circulation right now
  //      (legitimacy +0.01, the first beat of the material reroute);
  //   ② delayed — mayor_evidence_step subscribes to this key (Share Info
  //      feeding heartbeat) and charter_decline_reroute echoes at +4 months.
  // Bypass stays open: Mayor M-1 is unaffected, and succession_seed_procedure
  // can still reopen charter contact. Not a trap.
  player_action: {
    id: "CHARTER_CONTACT_DECLINE",
    label: "拒绝接触",
    description: "拒绝接触。本线就此关闭（本局无第二次约章接触）；断点材料即刻改道市长圈层，成为换人的筹码而非规则。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 0,            // initial calibration, TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("CHARTER_CONTACT_DECLINE"),
  effects: [
    // 材料流向改变（即时）：拒绝当下抄本进入市长圈层，权威小幅上行。
    // TODO(balance): 0.01 为改道第一笔；持续拉力仍由 +4 月回声与供证心跳承载。
    siloMetricDelta("legitimacy", 0.01),
  ],
});

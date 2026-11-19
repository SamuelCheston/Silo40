defineEvent({
  id: "charter_contact_decline",
  title: "拒绝：材料另寻出路",
  description: "你收回了材料。他点点头，没有追问：'规程这东西，等有人想要的时候才值钱。'断点材料还在你手里，制度内的这条门就此关上——它们往后更可能流向市长圈层，变成换人的筹码，而不是约束命令的规则。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // T-2 decline (26-nodes §2, 27-spec §1): closes the charter line. The
  // designed reroute of the materials into Mayor circles (30-nodes §6) is
  // wired: mayor_evidence_step subscribes to this key (feeding heartbeat via
  // Share Info), and charter_decline_reroute echoes the refusal at +4 months.
  player_action: {
    id: "CHARTER_CONTACT_DECLINE",
    label: "拒绝接触",
    description: "不把断点材料交给司法侧；本线关闭，材料留作他用。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 0,            // initial calibration, TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("CHARTER_CONTACT_DECLINE"),
  script: {
    apply: function(ctx) {
      return { effects: [], emit: [] };
    },
  },
});

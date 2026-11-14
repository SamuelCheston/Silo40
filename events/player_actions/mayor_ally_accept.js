defineEvent({
  id: "mayor_ally_accept",
  title: "接受：把他当部分盟友",
  description: "你决定支持他——用本地层的证据与关系，同时对他保留跨堡层。保留不是欺骗：他有权知道他职责内的事实，没有义务替你背不属于这座驻地的包袱。从这一天起，责任链材料多了一个去向：Mayor 圈层。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // M-2 accept (30-nodes §1, 31-spec §1): choice P-A projection. Accepting
  // opens the evidence-feeding track; the line's rate levers run through the
  // legitimacy deltas of mayor_evidence_step and mayor_public_stand.
  player_action: {
    id: "MAYOR_ALLY_ACCEPT",
    label: "接受：部分盟友",
    description: "用本地层证据与关系支持 Mayor，对他保留跨堡层。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 0,            // initial calibration, TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("MAYOR_ALLY_ACCEPT"),
  script: {
    apply: function(ctx) {
      return { effects: [], emit: [] };
    },
  },
});

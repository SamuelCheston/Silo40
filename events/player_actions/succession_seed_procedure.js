defineEvent({
  id: "succession_seed_procedure",
  title: "继承抉择：埋程序种子",
  description: "你决定在人治里埋一颗程序的种子：最低限度的继承安排、几份可以被引用的规则。这已经是《公议》的起点——司法侧的程序派会注意到这些种子。这一局可能不再以《人治》收束，但下一局的路已经铺好了。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // H-1 seed branch (30-nodes §4, §6, 31-spec §1): reroutes into the charter
  // line — the seed key adds an unlock path to charter_contact_offer (see
  // action_gates.json). Cost: the charter line restarts from contact, and the
  // lock day still waits for the W window; feeding and stands already spent
  // here do not carry over as charter steps.
  player_action: {
    id: "SUCCESSION_SEED_PROCEDURE",
    label: "埋程序种子",
    description: "在人治里埋下继承与规则的种子；转《公议》节点群。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 0,            // initial calibration, TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("SUCCESSION_SEED_PROCEDURE"),
  script: {
    apply: function(ctx) {
      return { effects: [], emit: [] };
    },
  },
});

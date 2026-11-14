defineEvent({
  id: "charter_contact_defer",
  title: "暂缓：先不回应",
  description: "你没有接，也没有拒。他把两份调度单收回去，抄本留在你这里：'想清楚再来找我。规程不等人，但也不记仇。'断点材料暂时压在你手里，起草会没有开起来，这扇门还留着一条缝。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // T-2 defer (26-nodes §2, 27-spec §1): the line does not start; the offer
  // key stays consumed, so re-approaching needs a new contact (not modeled
  // this round). Same PERSONAL_RULE reroute dependency as the decline branch.
  player_action: {
    id: "CHARTER_CONTACT_DEFER",
    label: "暂缓回应",
    description: "暂不回应程序派联系人；本线不启动。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 0,            // initial calibration, TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("CHARTER_CONTACT_DEFER"),
  script: {
    apply: function(ctx) {
      return { effects: [], emit: [] };
    },
  },
});

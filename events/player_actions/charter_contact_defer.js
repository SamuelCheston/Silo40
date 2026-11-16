defineEvent({
  id: "charter_contact_defer",
  title: "暂缓：先不回应",
  description: "你没有接，也没有拒。他把两份调度单收回去，抄本留在你这里：'那今天就谈到这里。'断点材料留在你手里，这次接触到此结束，起草会没有开起来。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // T-2 defer (26-nodes §2, 27-spec §1): the line does not start; the offer
  // key stays consumed, so re-approaching needs a new contact (not modeled
  // this round). Same PERSONAL_RULE reroute dependency as the decline branch.
  player_action: {
    id: "CHARTER_CONTACT_DEFER",
    label: "暂缓回应",
    description: "结束本次接触，暂不启动约章起草。",
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

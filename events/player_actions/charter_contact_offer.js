defineEvent({
  id: "charter_contact_offer",
  title: "接触：程序派联系人",
  description: "司法侧的一位程序派联系人约你在档案室外间见面。他摊开两份编号对不上的调度单：'我不问你是谁，我只问这些命令依据哪一条。'他想知道你手里还有多少这样的断点，以及你愿不愿意把它们交给一个还想按规程办事的人。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // T-1 (26-nodes §2, 27-spec §1): standard contact card (action design 15
  // §5.3). The gate (action_gates.json) unlocks it after the breach node and
  // a minimum Judicial relation; completing the offer unlocks the trilemma.
  // Pure contact node: no mechanical effect.
  player_action: {
    id: "CHARTER_CONTACT_OFFER",
    label: "见程序派联系人",
    description: "司法侧程序派联系人想谈谈你手里的规则断点材料。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 1,            // initial calibration, TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("CHARTER_CONTACT_OFFER"),
  script: {
    apply: function(ctx) {
      return { effects: [], emit: [] };
    },
  },
});

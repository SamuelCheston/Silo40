defineEvent({
  id: "receiver_verify_accompany",
  title: "陪同回查",
  description: "你陪第一位接收者把材料重新查了一遍：来源、缺口、可以核验的部分。他没有全盘接受，而是指出了一处你漏掉的旁证。",
  type: "INTERNAL",
  fire_mode: "ONCE",
  // I-3 fallback (24-spec): the preferred variant (receiver verifies on
  // their own when the department belief layer crosses a threshold) waits for
  // department-belief readability (24-spec §3, mechanism batch). This
  // fallback is a player-accompanied re-check action, wired today.
  player_action: {
    id: "RECEIVER_VERIFY_ACCOMPANY",
    label: "陪同回查",
    description: "陪同第一位接收者回查材料：来源、缺口与可核验部分。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2,            // initial calibration, TODO(balance)
    duration_months: 1,    // initial calibration, TODO(balance)
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("RECEIVER_VERIFY_ACCOMPANY"),
  effects: [
    // The re-check result enters the public evidence registry at provenance
    // 1 (verified records). This is the second distinct fact that L-1 counts.
    publicEvidenceRegister("fact:receiver_verification", 1, 0.1), // coverage TODO(balance)
  ],
});

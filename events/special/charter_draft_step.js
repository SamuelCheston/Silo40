defineEvent({
  id: "charter_draft_step",
  title: "起草步骤：供证不代笔",
  description: "起草会又吵了一轮。你把新核验的断点材料递进去：签发栏的空白、编号的断档、被绕过的正式渠道。有人引用旧规程反驳，有人拍桌子，条款在争吵里一条一条成形。你没有起草任何一个字——约章必须有本地主人。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 2, // initial calibration, TODO(balance)
  // D-1 (26-nodes §3, 27-spec §1): drafting heartbeat, same wiring as
  // replication_progress. Both completion keys persist, and the monthly
  // re-evaluation (fatal-flaw #1/#2 fix) advances the draft at the cooldown
  // pace while the player keeps the evidence circulating. Lives in special/
  // on purpose: a player_actions/ file firing on Share Info would intercept
  // and swallow the share's own resolution.
  trigger: all(
    eventTriggered("player_action", "charter_contact_accept"),
    eventTriggered("player_action", "action_share_info")
  ),
  script: {
    // Institutional feeding deepens the Judicial belief layer on the two
    // shared core dispatch facts (bailiff evidence chain) — the line's rate
    // lever (ADR-0010): supply accelerates crossing the pact-fact belief bar,
    // it does not replace it. Odd steps feed Judicial_1, even steps Judicial_2
    // (deterministic parity; trigger_count is post-fire, first fire = 1).
    apply: function(ctx) {
      var fact = (ctx.runtime.trigger_count % 2 === 1) ? "Judicial_1" : "Judicial_2";
      return {
        effects: [
          beliefObserve("Judicial", fact, 0.5), // initial calibration, TODO(balance)
        ],
      };
    },
  },
});

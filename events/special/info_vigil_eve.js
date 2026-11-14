defineEvent({
  id: "info_vigil_eve",
  title: "守夜",
  description: "最后一年。链已经不需要你逐日维持：接收者自行对账、自行纠错、自行安排下一轮核对。你剩下的事情只有一件——让宿主活着走到结算。资源、楼层、恐慌，以及不把叛乱推过那条会招来侦察的线。",
  type: "INTERNAL",
  fire_mode: "ONCE",
  // Convergence beat (25-information-endgame §1.4): lights once after P-1, in
  // the final year before the settlement window closes (relative Y7 = code
  // 129), while rebellion stays under the threshold that feeds Silo 1's
  // reconnaissance/purge chains. Pure narrative node: no numeric effects.
  trigger: all(
    eventTriggered("special", "public_fact_cross_dept"),
    timestampAtOrAfter(129, 1),
    siloMetricLte("rebellion", 0.8) // initial calibration, TODO(balance)
  ),
  script: {
    // DEPENDENCY NOTE: precise exposure management here waits for E
    // readability (24-spec §3 C-1); until then the rebellion threshold above
    // stands in. Upkeep against fragment fading (L-3 recirculation) waits for
    // the decay mechanism batch; this node does not read decay.
    apply: function(ctx) {
      return { effects: [], emit: [] };
    },
  },
});

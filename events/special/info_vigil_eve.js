defineEvent({
  id: "info_vigil_eve",
  title: "守夜",
  description: "结算已经望得见了。链不再需要你逐日维持：接收者自行对账、自行纠错、自行安排下一轮核对。你剩下的事情只有一件——让宿主活着走到结算。资源、楼层、恐慌，以及不把叛乱推过那条会招来侦察的线。",
  type: "INTERNAL",
  fire_mode: "ONCE",
  // Convergence beat (25-information-endgame §1.4): lights once after P-1,
  // while rebellion stays under the threshold that feeds Silo 1's
  // reconnaissance/purge chains. Pure narrative node: no numeric effects.
  //
  // Timing (修复4): the ending settles the month the belief gate is met, so a
  // fast line finishes before relative Y7 and never saw the original
  // final-year beat. The timing therefore moved into canTrigger: the node
  // lights in the final year (relative Y7 = code 129) OR once belief coverage
  // is within one department of the gate — the pre-ending foreshadow window.
  trigger: all(
    eventTriggered("special", "public_fact_cross_dept"),
    siloMetricLte("rebellion", 0.8) // initial calibration, TODO(balance)
  ),
  script: {
    canTrigger: function(ctx) {
      // Original window: the final year before the settlement window closes.
      if (ctx.year > 129 || (ctx.year === 129 && ctx.month >= 1)) {
        return true;
      }
      // Foreshadow window: the INFORMATION gate needs every department to
      // hold 3 MATURED facts (b >= 0.30 held for the full permeation window;
      // must stay in sync with informationGateFactCount / QualRumor /
      // PermeationMonths). Light the vigil once matured coverage is within
      // one department of the gate. TODO(balance): gap = 3 facts.
      var need = 3;
      var gap = 3;
      var missing = 0;
      var professions = ctx.professions || [];
      if (professions.length === 0) {
        return false;
      }
      for (var i = 0; i < professions.length; i++) {
        var matured = 0;
        var beliefs = professions[i].beliefs || [];
        for (var j = 0; j < beliefs.length; j++) {
          if (beliefs[j].matured) {
            matured++;
          }
        }
        if (matured < need) {
          missing += need - matured;
        }
      }
      return missing <= gap;
    },
    // DEPENDENCY NOTE: precise exposure management here waits for E
    // readability (24-spec §3 C-1); until then the rebellion threshold above
    // stands in. The node does not read decay; upkeep against fragment fading
    // (L-3 recirculation) is expressed by the progress panel's coverage.
    apply: function(ctx) {
      return { effects: [], emit: [] };
    },
  },
});

defineEvent({
  id: "silo1_purge_execution",
  title: "本地整肃执行",
  description: "整肃在一个夜里开始：名单上的人被分批带走，官方口径是转移与隔离。等清点重新公布时，人口少了一截，关于那一夜的记录也一并消失了。留下的人学会了两件事——别再问，别再记得。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Scheduled-only execution node: fired via schedule_event from
  // silo1_purge_omen. Guard reads the exposure completion key: an exposed plot
  // loses its element of surprise and no-ops; an unopposed plot carries out.
  script: {
    canTrigger: function(ctx) {
      return false;
    },
    apply: function(ctx) {
      var exposed = ctx.states["special:silo1_purge_exposed"];
      if (exposed && exposed.triggered) {
        // The plot was exposed before it could act; the crackdown stalls.
        return {
          effects: [siloMetricDelta("rebellion", 0.02)],
        };
      }
      // Unopposed crackdown: attrition + record clearing. The population loss
      // grinds toward 《寂堡》 (DEATH); the memory wipe shows up as history
      // burden dropping. TODO(balance): magnitudes are sim-calibration starts.
      return {
        effects: [
          siloPopulationDelta(-1500),
          professionMetricDeltaAll("panic_value", 0.08),
          siloMetricDelta("safeguard_risk", 0.2),
          siloMetricDelta("history_burden", -0.1),
        ],
      };
    },
  },
  effects: [],
});

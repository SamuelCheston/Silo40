defineEvent({
  id: "silo1_purge_execution",
  title: "本地整肃执行",
  description: "整肃在一个夜里开始：名单上的人被分批带走，官方口径是转移与隔离。关于那一夜的记录，随后一并消失了。留下的人学会了两件事——别再问，别再记得。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Scheduled-only settlement hook: fired via schedule_event from
  // silo1_purge_omen.
  //
  // 整肃的后果按名册是否被动过分流 (33/34 分册)：重档削人口与记录，轻档保住
  // 一部分人。后果只落人口、历史包袱、凝聚与恐慌——整肃不直接致死，削弱的
  // 宿主更容易死于其他来路。数值一律 TODO(balance)。
  script: {
    canTrigger: function(ctx) {
      return false;
    },
    apply: function(ctx) {
      var softened = false;
      if (ctx.states && ctx.states["special:silo1_purge_softened"]) {
        softened = !!ctx.states["special:silo1_purge_softened"].triggered;
      }
      var population = ctx.total_population || 0;
      var share = softened ? 0.03 : 0.08;
      return {
        effects: [
          siloPopulationDelta(-Math.round(population * share)),
          siloMetricDelta("history_burden", softened ? 0.04 : 0.08),
          siloMetricDelta("cohesion", softened ? -0.02 : -0.05),
          professionMetricDeltaAll("panic_value", softened ? 0.03 : 0.06),
        ],
      };
    },
  },
});

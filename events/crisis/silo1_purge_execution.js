defineEvent({
  id: "silo1_purge_execution",
  title: "本地整肃执行",
  description: "整肃在一个夜里开始：名单上的人被分批带走，官方口径是转移与隔离。关于那一夜的记录，随后一并消失了。留下的人学会了两件事——别再问，别再记得。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Scheduled-only settlement hook: fired via schedule_event from
  // silo1_purge_omen.
  //
  // 整肃的后果按反制分档 (33/34 分册)：名册被动过（轻档）削得最少；图谋被
  // 公开（中档）失去突然性，少带走一些人；两者都没有（重档）最重。两档同时
  // 成立时按轻档——纸面上的改动比风声更能救人。后果只落人口、历史包袱、凝聚
  // 与恐慌——整肃不直接致死，削弱的宿主更容易死于其他来路。数值一律
  // TODO(balance)。
  script: {
    canTrigger: function(ctx) {
      return false;
    },
    apply: function(ctx) {
      var softened = false;
      if (ctx.states && ctx.states["special:silo1_purge_softened"]) {
        softened = !!ctx.states["special:silo1_purge_softened"].triggered;
      }
      var exposed = false;
      if (ctx.states && ctx.states["special:silo1_purge_exposed"]) {
        exposed = !!ctx.states["special:silo1_purge_exposed"].triggered;
      }
      var population = ctx.total_population || 0;
      // E 次消费点 (22-00 裁决 1)：反应链触发后的烈度调制——整肃的实际落实
      // 力度按本堡执行强度折算 (execution_intensity 由内容上下文暴露；功能
      // 未启用时为 1.0，行为逐位不变)。触发判定不读 E，只调制触发后烈度。
      var intensity = typeof ctx.execution_intensity === "number" ? ctx.execution_intensity : 1.0;
      var share = 0.08;
      var burden = 0.08;
      var cohesion = -0.05;
      var panic = 0.06;
      if (softened) {
        share = 0.03;
        burden = 0.04;
        cohesion = -0.02;
        panic = 0.03;
      } else if (exposed) {
        share = 0.05;
        burden = 0.06;
        cohesion = -0.03;
        panic = 0.04;
      }
      share *= intensity;
      return {
        effects: [
          siloPopulationDelta(-Math.round(population * share)),
          siloMetricDelta("history_burden", burden),
          siloMetricDelta("cohesion", cohesion),
          professionMetricDeltaAll("panic_value", panic),
        ],
      };
    },
  },
});

defineMechanics([
  {
    id: "agent_stats_formula",
    formula: "agent_stats",
    apply(ctx) {
      let avgConnection = 0;
      const connections = ctx.actor.connections || [];
      if (connections.length > 0) {
        for (const entry of connections) {
          avgConnection += entry.value || 0;
        }
        avgConnection /= connections.length;
      }

      // 威望全层派生 (24 篇 §2)：50×(Ambition+Influence)，展示同口径。
      const ambition = ctx.actor.ambition || 0;
      const influence = ctx.actor.influence || 0;
      const prestige = 50 * (ambition + influence);

      return {
        stats: {
          avg_connection: avgConnection,
          ambition: ambition,
          influence: influence,
          political_prestige: prestige,
          // 与 core_systems.js agent_update 的实际恢复公式逐位一致
          // (年恢复 36 + 威望*0.05，TODO(balance))；这是展示值，不是另一套经济。
          ap_base_recovery: 36,
          ap_prestige_bonus: prestige * 0.05,
          ap_total_recovery: 36 + prestige * 0.05,
          ap_max: 100,
          propaganda_level: ctx.actor.propaganda_level || 0,
          propaganda_multiplier: 1 + (ctx.actor.propaganda_level || 0) * 0.2,
          rebellion_base_effect: 0.05 + prestige * 0.002,
          is_faction_leader: !!ctx.actor.is_faction_leader,
        },
      };
    },
  },
  {
    id: "score_formula",
    formula: "score",
    apply(ctx) {
      let survival = ctx.silo.total_population || 0;
      let diversity = 0;
      for (const profession of ctx.silo.professions || []) {
        if ((profession.productivity || 0) > 0.5) {
          diversity += 100;
        }
      }
      const heritage = Math.floor((1.0 - (ctx.silo.metrics.history_burden || 0)) * 500);
      let avgIdeology = 0;
      const professions = ctx.silo.professions || [];
      if (professions.length > 0) {
        for (const profession of professions) {
          avgIdeology += ((profession.ideologies || {}).pro_foreign || 0);
        }
        avgIdeology /= professions.length;
      }
      const ideology = Math.floor(avgIdeology * 200);
      let multiplier = 1.0;
      const victoryType = (((ctx.silo || {}).victory_status || {}).type || "").toUpperCase();
      switch (victoryType) {
        case "INFORMATION":
          multiplier = 2.0;
          break;
        case "TIME":
          multiplier = 1.5;
          break;
        case "REBELLION":
          multiplier = 1.2;
          break;
        case "EXCLUSIONIST":
          multiplier = 0.5;
          break;
        case "DEATH":
        case "AGENT_COMPROMISED":
          multiplier = 0;
          break;
      }
      return {
        score: {
          total: Math.floor((survival + diversity + heritage + ideology) * multiplier),
          survival_points: survival,
          diversity_points: diversity,
          heritage_points: heritage,
          ideology_points: ideology,
          multiplier: multiplier,
        },
      };
    },
  },
]);

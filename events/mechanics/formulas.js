defineMechanics([
  {
    id: "agent_stats_formula",
    formula: "agent_stats",
    apply(ctx) {
      function classBias(classType, delta) {
        return classType === "ELITE" ? delta : -delta * 0.35;
      }
      const professionFactors = {
        Mayor: 0.5,
        Judicial: 0.4,
        IT: 0.3,
        Police: 0.3,
        Mechanical: 0.2,
        Medical: 0.2,
      };
      const traitFactors = {
        "地堡土著": 0.1,
        "一号地堡密使": 0.5,
        "煽动者": 0.2,
        "守旧派": -0.1,
      };

      let avgConnection = 0;
      const connections = ctx.actor.connections || [];
      if (connections.length > 0) {
        for (const entry of connections) {
          avgConnection += entry.value || 0;
        }
        avgConnection /= connections.length;
      }

      const professionFactor = professionFactors[ctx.actor.profession] || 0;
      let traitFactor = 0;
      for (const trait of ctx.actor.traits || []) {
        traitFactor += traitFactors[trait] || 0;
      }
      const prestigeBase = avgConnection * 100;
      const structuralPrestige = (ctx.actor.power_level || 0) * 1.5 + classBias(ctx.actor.class_type || "", 2.0);
      let prestige = prestigeBase * (1 + professionFactor) * (1 + traitFactor) + structuralPrestige;
      if (prestige < 0) prestige = 0;

      return {
        stats: {
          avg_connection: avgConnection,
          prestige_base: prestigeBase,
          profession_factor: professionFactor,
          trait_factor: traitFactor,
          structural_prestige: structuralPrestige,
          political_prestige: prestige,
          ap_base_recovery: 10,
          ap_prestige_bonus: prestige * 0.05,
          ap_total_recovery: 10 + prestige * 0.05,
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

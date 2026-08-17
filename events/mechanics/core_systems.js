defineMechanics([
  {
    id: "agent_update",
    event_type: "AGENT_UPDATE",
    apply(ctx) {
      const deltaYears = ctx.event.delta_years || 0;
      if (deltaYears <= 0 || !ctx.actor) {
        return {};
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
      function classBias(classType, delta) {
        return classType === "ELITE" ? delta : -delta * 0.35;
      }

      const mutations = [];
      const logs = [];

      if (ctx.actor.profession === "Medical" && randomFloat() < 0.2) {
        const known = {};
        (ctx.actor.known_fragments || []).forEach((fragment) => {
          known[fragment] = true;
        });
        const available = (ctx.all_fragments || []).filter((fragment) => !known[fragment]);
        if (available.length > 0) {
          const fragment = available[randomInt(available.length)];
          mutations.push({ type: "actor_fragment_add", fragment });
          logs.push("[Medical Passive] " + ctx.actor.label + " overheard rumors about " + fragment + ".");
        }
      }

      const connections = ctx.actor.connections || [];
      let avgConnection = 0;
      if (connections.length > 0) {
        for (const entry of connections) {
          avgConnection += entry.value || 0;
        }
        avgConnection /= connections.length;
      }

      const profFactor = professionFactors[ctx.actor.profession] || 0;
      let traitFactor = 0;
      for (const trait of ctx.actor.traits || []) {
        traitFactor += traitFactors[trait] || 0;
      }

      const prestigeBase = avgConnection * 100;
      const structuralPrestige = (ctx.actor.power_level || 0) * 1.5 + classBias(ctx.actor.class_type || "", 2.0);
      let prestige = prestigeBase * (1 + profFactor) * (1 + traitFactor) + structuralPrestige;
      if (prestige < 0) {
        prestige = 0;
      }

      mutations.push({ type: "actor_metric_set", field: "political_prestige", value: prestige });
      if (ctx.actor.kind === "PLAYER") {
        mutations.push({ type: "actor_metric_delta", field: "political_points", value: prestige * 0.1 * deltaYears });
      }

      const apGainRate = 10 + prestige * 0.05;
      const nextAP = clamp(ctx.actor.action_points + apGainRate * deltaYears, 0, 100);
      mutations.push({ type: "actor_metric_set", field: "action_points", value: nextAP });

      const nextSuspicion = Math.max(0, ctx.actor.suspicion_level - 0.05 * deltaYears);
      mutations.push({ type: "actor_metric_set", field: "suspicion_level", value: nextSuspicion });

      return { mutations, logs };
    },
  },
  {
    id: "resource_update",
    event_type: "RESOURCE_UPDATE",
    apply(ctx) {
      const deltaYears = ctx.event.delta_years || 0;
      if (deltaYears <= 0) {
        return {};
      }
      const metrics = ctx.silo.metrics || {};
      const resources = ctx.silo.resources || {};
      const professions = ctx.silo.professions || [];
      const totalPop = ctx.silo.total_population || 0;
      const isRebelling = (metrics.rebellion || 0) > 0.7;

      function efficiency(name) {
        const profession = professionByName(ctx, name);
        if (!profession) {
          return 0;
        }
        return (isRebelling ? 0.3 : 1.0) * (profession.population || 0);
      }

      const prodRate = 4.5;
      const perCapita = { Supplies: 1.0, Energy: 1.0, Materials: 1.0 };

      const energyProdRate = prodRate * perCapita.Energy * efficiency("Mechanical");
      const energyConsPopRate = perCapita.Energy * totalPop;
      const materialsProdRate = prodRate * perCapita.Materials * efficiency("Mines");
      const materialsConsPopRate = perCapita.Materials * totalPop;
      const agriCapacityRate = prodRate * perCapita.Supplies * efficiency("Agricultural");
      const supplyCapacityRate = prodRate * perCapita.Supplies * efficiency("Supply");
      const suppliesConsPopRate = perCapita.Supplies * totalPop;

      const energyAvailable = Math.max(0, (resources.Energy || 0) / deltaYears + energyProdRate - energyConsPopRate);
      const actualAgriSuppliesProd = Math.min(agriCapacityRate, energyAvailable);
      const materialsAvailable = Math.max(0, (resources.Materials || 0) / deltaYears + materialsProdRate - materialsConsPopRate);
      const actualSupplySuppliesProd = Math.min(supplyCapacityRate, materialsAvailable);

      const energyNet = energyProdRate - energyConsPopRate - actualAgriSuppliesProd;
      const materialsNet = materialsProdRate - materialsConsPopRate - actualSupplySuppliesProd;
      const suppliesNet = actualAgriSuppliesProd + actualSupplySuppliesProd - suppliesConsPopRate;

      return {
        mutations: [
          { type: "resource_net_balance_set", resource: "Energy", value: energyNet },
          { type: "resource_net_balance_set", resource: "Materials", value: materialsNet },
          { type: "resource_net_balance_set", resource: "Supplies", value: suppliesNet },
          { type: "resource_delta", resource: "Energy", value: energyNet * deltaYears },
          { type: "resource_delta", resource: "Materials", value: materialsNet * deltaYears },
          { type: "resource_delta", resource: "Supplies", value: suppliesNet * deltaYears },
        ],
      };
    },
  },
  {
    id: "ideology_update",
    event_type: "IDEOLOGY_UPDATE",
    apply(ctx) {
      const deltaYears = ctx.event.delta_years || 0;
      if (deltaYears <= 0) {
        return {};
      }
      const mutations = [];
      const stability = ctx.silo.metrics.cohesion || 0;

      for (const cohort of ctx.silo.cohorts || []) {
        const profession = (ctx.silo.professions || []).find((item) => item.id === cohort.profession_id);
        if (!profession) {
          continue;
        }
        let drift = 0;
        if ((profession.panic_value || 0) > 0.3 && stability < 0.5) {
          drift += profession.panic_value * (1.0 - stability) * deltaYears * 0.01;
        }
        if ((profession.panic_value || 0) > 0) {
          drift += profession.panic_value * 0.10 * deltaYears;
        }
        if (drift !== 0) {
          mutations.push({
            type: "cohort_ideology_delta",
            ideology: "pro_foreign",
            int_value: cohort.id,
            value: drift,
          });
        }
      }

      if ((ctx.silo.traits || []).includes("psychoactive_meds")) {
        const itDept = professionByName(ctx, "IT");
        if (itDept) {
          for (const cohort of ctx.silo.cohorts || []) {
            const profession = (ctx.silo.professions || []).find((item) => item.id === cohort.profession_id);
            if (!profession || profession.name === "IT") {
              continue;
            }
            const target = (itDept.ideologies || {});
            const current = cohort.ideologies || {};
            for (const key of Object.keys(target)) {
              const diff = (target[key] || 0) - (current[key] || 0);
              if (diff !== 0) {
                mutations.push({
                  type: "cohort_ideology_delta",
                  ideology: key,
                  int_value: cohort.id,
                  value: diff * 0.05 * deltaYears,
                });
              }
            }
          }
        }
      }

      for (const profession of ctx.silo.professions || []) {
        if ((profession.panic_value || 0) > 0) {
          mutations.push({
            type: "profession_metric_set",
            profession: profession.name,
            field: "panic_value",
            value: Math.max(0, profession.panic_value - profession.panic_value * 0.10 * deltaYears),
          });
        }
      }

      mutations.push({ type: "sync_profession_ideologies_from_cohorts" });
      return { mutations };
    },
  },
  {
    id: "metrics_update",
    event_type: "METRICS_UPDATE",
    apply(ctx) {
      const deltaYears = ctx.event.delta_years || 0;
      if (deltaYears <= 0) {
        return {};
      }
      const metrics = ctx.silo.metrics || {};
      const mutations = [];

      mutations.push({
        type: "silo_metric_set",
        metric: "countdown",
        value: Math.max(0, (metrics.countdown || 0) - deltaYears),
      });
      mutations.push({
        type: "silo_metric_delta",
        metric: "event_trigger",
        value: (1.0 - (metrics.cohesion || 0)) * deltaYears * 0.1,
      });

      let totalInf = 0;
      let weightedCohesion = 0;
      let radicalInf = 0;
      for (const faction of ctx.silo.factions || []) {
        totalInf += faction.influence || 0;
        weightedCohesion += (faction.cohesion || 0) * (faction.influence || 0);
        if ((faction.cohesion || 0) < 0.4) {
          radicalInf += (faction.influence || 0) * (0.4 - (faction.cohesion || 0));
        }
      }

      let rebellion = metrics.rebellion || 0;
      if (totalInf > 0) {
        const cohesion = weightedCohesion / totalInf;
        mutations.push({ type: "silo_metric_set", metric: "cohesion", value: cohesion });
        mutations.push({ type: "silo_metric_set", metric: "legitimacy", value: cohesion });
        const stressFactor = radicalInf / totalInf;
        if (stressFactor > 0.05) {
          rebellion += (stressFactor - 0.05) * deltaYears * 0.1;
        } else {
          rebellion -= 0.02 * deltaYears;
        }
      }
      mutations.push({ type: "silo_metric_set", metric: "rebellion", value: clampUnit(rebellion) });

      let totalRel = 0;
      let relCount = 0;
      for (const profession of ctx.silo.professions || []) {
        const ideologies = profession.ideologies || {};
        const relations = profession.relations || {};
        void ideologies;
        for (const key of Object.keys(relations)) {
          totalRel += relations[key] || 0;
          relCount += 1;
        }
      }
      if (relCount > 0) {
        mutations.push({
          type: "silo_metric_set",
          metric: "dept_tension",
          value: 1.0 - totalRel / relCount,
        });
      }

      let eliteLoyalty = 0;
      let commonerLoyalty = 0;
      let eliteCount = 0;
      let commonerCount = 0;
      for (const profession of ctx.silo.professions || []) {
        let avgLoyalty = 0;
        let cohortCount = 0;
        for (const cohort of ctx.silo.cohorts || []) {
          if (cohort.profession_id !== profession.id) {
            continue;
          }
          avgLoyalty += ((cohort.ideologies || {}).loyalty || 0);
          cohortCount += 1;
        }
        if (cohortCount > 0) {
          avgLoyalty /= cohortCount;
        }
        if (profession.class_type === "ELITE") {
          eliteLoyalty += avgLoyalty;
          eliteCount += 1;
        } else {
          commonerLoyalty += avgLoyalty;
          commonerCount += 1;
        }
      }
      if (eliteCount > 0 && commonerCount > 0) {
        mutations.push({
          type: "silo_metric_set",
          metric: "class_fragmentation",
          value: Math.abs(eliteLoyalty / eliteCount - commonerLoyalty / commonerCount),
        });
      }

      let deathRate = 0.001;
      for (const value of Object.values(ctx.silo.resources || {})) {
        if (value <= 0) {
          deathRate += 0.05;
        }
      }
      if ((metrics.rebellion || 0) > 0.8) {
        deathRate += ((metrics.rebellion || 0) - 0.8) * 0.2;
      }
      const deaths = Math.floor((ctx.silo.total_population || 0) * deathRate * deltaYears);
      if (deaths > 0) {
        mutations.push({ type: "apply_population_deaths", int_value: deaths });
      }
      mutations.push({ type: "refresh_profession_population_from_cohorts" });
      return { mutations };
    },
  },
  {
    id: "victory_check",
    event_type: "VICTORY_CHECK",
    apply(ctx) {
      const silo = ctx.silo || {};
      const metrics = silo.metrics || {};
      const actor = ctx.actor || {};

      if ((metrics.safeguard_risk || silo.safeguard_risk || 0) >= 1.0) {
        return {
          mutations: [{
            type: "victory_status_set",
            field: "DEATH",
            bool_value: false,
            text: "Safeguard 协议被激活。IT部门的过度干预触发了底层核心逻辑，清理程序启动，40号地堡被彻底清洗。",
          }],
        };
      }

      if ((silo.professions || []).length > 0) {
        let allDeptsHaveFragments = true;
        for (const profession of silo.professions) {
          const unique = {};
          for (const fragment of profession.known_fragments || []) {
            unique[fragment] = true;
          }
          if (Object.keys(unique).length < 5) {
            allDeptsHaveFragments = false;
            break;
          }
        }
        if (allDeptsHaveFragments) {
          return {
            mutations: [{
              type: "victory_status_set",
              field: "INFORMATION",
              bool_value: true,
              text: "你成功让真相在所有部门间流传。全知视角的拼图终于拼凑完整，地堡的居民迎来了觉醒的黎明。",
            }],
          };
        }
      }

      if (silo.silo1_destroyed) {
        return {
          mutations: [{
            type: "victory_status_set",
            field: "TIME",
            bool_value: true,
            text: "一号地堡已经覆灭，控制网络断开。40号地堡迎来了属于自己的时间。",
          }],
        };
      }

      if ((actor.suspicion_level || 0) >= 1.0) {
        return {
          mutations: [{
            type: "victory_status_set",
            field: "AGENT_COMPROMISED",
            bool_value: false,
            text: "由于传播过多掺杂了个人意图的虚假信息，你的特工身份彻底暴露。司法部已经下达了逮捕令。",
          }],
        };
      }

      if ((metrics.rebellion || 0) >= 1.0) {
        return {
          mutations: [{
            type: "victory_status_set",
            field: "REBELLION",
            bool_value: true,
            text: "你成功组织了反抗力量并发动了叛乱。旧的统治被推翻，幸存者们冲破了封闭的牢笼。",
          }],
        };
      }

      if ((silo.total_population || 0) <= 0) {
        return {
          mutations: [{
            type: "victory_status_set",
            field: "DEATH",
            bool_value: false,
            text: "地堡内已无生命迹象。人类最后的堡垒沦为了一座寂静的坟墓。",
          }],
        };
      }

      return {};
    },
  },
]);

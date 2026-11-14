defineMechanics([
  {
    id: "action_gather_info",
    action_type: "GATHER_INFO",
    label: "Gather Info",
    description: "Probe a department for information. Success depends on fit, alertness and channels; failure raises the target's alertness.",
    target_type: "DEPT",
    ap_cost: 10,
    duration_months: 0,
    apply(ctx) {
      const target = ctx.action.target_dept;
      if (!target) {
        return { action_result: { executed: false, message: "Invalid target department." } };
      }
      const profession = professionByName(ctx, target);
      if (!profession) {
        return { action_result: { executed: false, message: "Target department not found." } };
      }
      const unknown = unknownFragmentsFrom(ctx, target);
      if (unknown.length === 0) {
        return { action_result: { executed: false, message: "Your department already knows everything about " + target + "." } };
      }
      // 单次获取 (10-it-control-credibility §3.4)：先抽碎片再滚获取概率，
      // 两次抽样都走单一 RNG 流。失败不产生确信，只结算 AP/检测/戒备。
      const fragment = unknown[randomInt(unknown.length)];
      const channel = gatherChannelFor(ctx, target);
      const ownDept = ctx.actor.profession === target;
      const deviation = ownDept ? 0 : INFO_CFG.gather_deviation;
      const mismatch = mismatchFor(ctx, target, deviation);
      const alert = profession.alertness || 0;
      const related = relatedF(ctx, fragment);
      const fProf = fragmentDomain(fragment) === ctx.actor.profession ? INFO_CFG.f_prof_match : INFO_CFG.f_prof_mismatch;
      const p = pGet(fProf, mismatch, alert, related, channel.access_c);
      const detection = detectionClass(ctx, target, false);
      const mutations = [{ type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) }];
      const sampled = detection === "SAMPLED";
      if (randomFloat() >= p) {
        mutations.push({ type: "profession_alertness_delta", profession: target, value: INFO_CFG.alert_on_failed_probe });
        if (sampled) {
          const fit = ownDept ? INFO_CFG.probe_fit_own : INFO_CFG.probe_fit_cross;
          mutations.push({
            type: "actor_metric_delta",
            field: "suspicion_level",
            value: sampledSuspicion(INFO_CFG.probe_base_suspicion, INFO_CFG.probe_visibility, fit, INFO_CFG.probe_witness, INFO_CFG.probe_record, mismatch),
          });
        }
        return {
          mutations,
          action_result: { executed: true, message: "The probe on " + target + " found nothing this time (chance " + Math.round(p * 100) + "%). The department grows more alert." },
        };
      }
      mutations.push({
        type: "actor_belief_observe",
        fragment,
        observation: "gather:" + fragment,
        reliability: channel.reliability_c,
        related_f: related,
        independence: 1,
      });
      if (sampled) {
        const fit = ownDept ? INFO_CFG.probe_fit_own : INFO_CFG.probe_fit_cross;
        mutations.push({
          type: "actor_metric_delta",
          field: "suspicion_level",
          value: sampledSuspicion(INFO_CFG.probe_base_suspicion, INFO_CFG.probe_visibility, fit, INFO_CFG.probe_witness, INFO_CFG.probe_record, mismatch),
        });
      }
      return {
        mutations,
        action_result: { executed: true, message: "Observed new material on " + fragment + " from " + target + " via " + channel.id + " (chance " + Math.round(p * 100) + "%)." },
      };
    },
  },
  {
    id: "action_share_info",
    action_type: "SHARE_INFO",
    label: "Share Info",
    description: "Share real or fake fragments with a department.",
    target_type: "DEPT",
    ap_cost: 20,
    duration_months: 0,
    apply(ctx) {
      const target = ctx.action.target_dept;
      const fragmentIds = ctx.action.fragment_ids || [];
      if (!target || fragmentIds.length === 0) {
        return { action_result: { executed: false, message: "Invalid target or no fragments selected." } };
      }
      const profession = professionByName(ctx, target);
      if (!profession) {
        return { action_result: { executed: false, message: "Target department not found." } };
      }
      const known = actorKnownSet(ctx);
      let unexplainedCount = 0;
      for (const fragment of fragmentIds) {
        if (!known[fragment]) {
          unexplainedCount += 1;
        }
        // §3.6 行动资格：B < 0.30 的材料只是私人笔记，不能作为事实分享。
        if (beliefB(ctx, fragment) < INFO_CFG.qual_rumor) {
          return { action_result: { executed: false, message: fragment + " is still private suspicion (belief below " + INFO_CFG.qual_rumor + "); it cannot be shared as a fact." } };
        }
      }
      const connectionValue = actorConnectionTo(ctx, target);
      let acceptanceRate = 0.1 + ((profession.ideologies || {}).pro_foreign || 0) + connectionValue - unexplainedCount * 0.1;
      acceptanceRate = clamp(acceptanceRate, 0.05, 1.0);

      const mutations = [{ type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) }];
      if (unexplainedCount > 0) {
        const suspicionPenalty = unexplainedCount * 0.1 + Math.pow(unexplainedCount, 1.5) * 0.05;
        mutations.push({ type: "actor_metric_delta", field: "suspicion_level", value: suspicionPenalty });
      }

      if (randomFloat() > acceptanceRate) {
        return {
          mutations,
          action_result: {
            executed: true,
            message: "Attempted to share info with " + profession.name + ", but they rejected it! (Acceptance rate was " + Math.floor(acceptanceRate * 100) + "%)",
          },
        };
      }

      let added = 0;
      const targetKnown = {};
      for (const fragment of profession.known_fragments || []) {
        targetKnown[fragment] = true;
      }
      for (const fragment of fragmentIds) {
        if (!targetKnown[fragment]) {
          mutations.push({ type: "profession_fragment_add", profession: profession.name, fragment });
          targetKnown[fragment] = true;
          added += 1;
        }
      }
      mutations.push({ type: "profession_metric_set", profession: profession.name, field: "panic_value", value: clampUnit((profession.panic_value || 0) + 0.05 + unexplainedCount * 0.05) });
      if (connectionValue >= 0.3) {
        mutations.push({ type: "profession_ideology_delta", profession: profession.name, ideology: "pro_foreign", value: 0.02 + unexplainedCount * 0.02 });
      }
      return {
        mutations,
        action_result: {
          executed: true,
          message: "Successfully shared " + added + " fragments with " + profession.name + ". (Included " + unexplainedCount + " pieces of unexplained knowledge)",
        },
      };
    },
  },
  {
    id: "action_build_connection",
    action_type: "BUILD_CONNECTION",
    label: "Build Network",
    description: "Build influence with a department over time.",
    target_type: "DEPT",
    ap_cost: 15,
    duration_months: 1,
    apply(ctx) {
      const target = ctx.action.target_dept;
      if (!target) {
        return { action_result: { executed: false, message: "Invalid target department." } };
      }
      const profession = professionByName(ctx, target);
      if (!profession) {
        return { action_result: { executed: false, message: "Target department not found." } };
      }
      let increase = 0.05 + (ctx.actor.political_prestige || 0) * 0.005;
      if ((ctx.actor.traits || []).includes("魅力非凡")) {
        increase *= 1.5;
      }
      return {
        mutations: [
          { type: "actor_connection_delta", connection_dept: target, value: increase },
          { type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) },
          suspicionDelta(ctx, 0.01),
        ],
        action_result: { executed: true, message: "Successfully built connections with " + profession.name + "." },
      };
    },
  },
  {
    id: "action_incite_rebellion",
    action_type: "INCITE_REBELLION",
    label: "Incite Rebellion",
    description: "Agitate commoner departments.",
    target_type: "NONE",
    ap_cost: 30,
    duration_months: 2,
    apply(ctx) {
      const mutations = [{ type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) }];
      for (const profession of ctx.silo.professions || []) {
        if (profession.class_type !== "COMMONER") {
          continue;
        }
        const connectionValue = actorConnectionTo(ctx, profession.name);
        const baseEffect = 0.05 + (ctx.actor.political_prestige || 0) * 0.002;
        const propagandaMultiplier = 1 + (ctx.actor.propaganda_level || 0) * 0.2;
        const finalEffect = baseEffect * (1 + connectionValue) * propagandaMultiplier;
        mutations.push({ type: "profession_metric_delta", profession: profession.name, field: "panic_value", value: finalEffect });
        mutations.push({ type: "profession_ideology_delta", profession: profession.name, ideology: "pro_foreign", value: finalEffect * 0.5 });
      }
      mutations.push(suspicionDelta(ctx, 0.05));
      return {
        mutations,
        action_result: { executed: true, message: "Incited unrest among all commoner departments." },
      };
    },
  },
  {
    id: "action_conduct_propaganda",
    action_type: "CONDUCT_PROPAGANDA",
    label: "Propaganda",
    description: "Increase your propaganda level.",
    target_type: "NONE",
    ap_cost: 20,
    duration_months: 1,
    apply(ctx) {
      return {
        mutations: [
          { type: "actor_metric_delta", field: "propaganda_level", value: 1.0 },
          { type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) },
          suspicionDelta(ctx, 0.02),
        ],
        action_result: { executed: true, message: "Conducted propaganda. Propaganda Level increased by 1.0." },
      };
    },
  },
  {
    id: "action_publicize_faction",
    action_type: "PUBLICIZE_FACTION",
    label: "Publicize Faction",
    description: "Reveal your faction to the silo and gain prestige.",
    target_type: "NONE",
    ap_cost: 25,
    duration_months: 0,
    apply(ctx) {
      const faction = ctx.actor.faction;
      if (!ctx.actor.is_representative || !faction) {
        return { action_result: { executed: false, message: "Only faction representatives can publicize their faction." } };
      }
      if (faction.is_public) {
        return { action_result: { executed: false, message: "Faction is already public." } };
      }
      return {
        mutations: [
          { type: "faction_is_public_actor", bool_value: true },
          { type: "faction_metric_delta_actor", field: "prestige", value: 15.0 },
          { type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) },
        ],
        action_result: { executed: true, message: "Successfully publicized " + faction.name + ". The silo is now aware of your presence." },
      };
    },
  },
]);

// suspicionDelta / detectionClass live in the mechanic prelude; detection is
// structural (UNSAMPLED / SAMPLED / DIRECT_PROTOCOL), not a profession factor.

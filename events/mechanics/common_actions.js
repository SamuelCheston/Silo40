defineMechanics([
  {
    id: "action_gather_info",
    action_type: "GATHER_INFO",
    label: "Gather Info",
    description: "Probe a department for information. Success depends on fit, alertness and channels; failure raises the target's alertness.",
    target_type: "DEPT",
    // AP 经济修复 (17-information-economy §3, TODO(balance)): 10 → 2。
    // 流通动词必须便宜到月收入 (~3/月) 可以支撑每月级的节奏。
    ap_cost: 2,
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
      // 复核路径 (修复6)：目标部门的新材料枯竭后，探查不再生效新碎片，但
      // 反复核验已知材料仍能加深特工自己的确信 (23 节点设计 I-1"第三次趋近
      // 确定但永不为 1")。没有这条路径，本部门域碎片不足的职业 (如 Supply/
      // Mechanical 域只有 2 份) 无法把 B_player 抬到分享吸收有效的强度，
      // 《共证》线对它们事实上不可赢。抽样结构与正常路径一致 (同一 RNG 流)。
      let recheck = false;
      let pool = unknown;
      if (pool.length === 0) {
        const knownSet = actorKnownSet(ctx);
        pool = (profession.all_fragments || []).filter(function (fragment) {
          return knownSet[fragment];
        });
        if (pool.length === 0) {
          return { action_result: { executed: false, message: "Your department already knows everything about " + target + "." } };
        }
        recheck = true;
      }
      // 单次获取 (10-it-control-credibility §3.4)：先抽碎片再滚获取概率，
      // 两次抽样都走单一 RNG 流。失败不产生确信，只结算 AP/检测/戒备。
      const fragment = pool[randomInt(pool.length)];
      const channel = gatherChannelFor(ctx, target);
      const ownDept = ctx.actor.profession === target;
      // 复核是定向核验已持有材料：偏离强度减半 (TODO(balance))。跨域复核仍比
      // 本部门复核难——不抹平职业差异，只避免警戒度螺旋把弱关系职业的深化
      // 通道彻底封死 (职业横评：Mechanical/Police 因此卡在第三份碎片)。
      const deviation = ownDept ? 0 : INFO_CFG.gather_deviation * (recheck ? 0.5 : 1);
      const mismatch = mismatchFor(ctx, target, deviation);
      const alert = profession.alertness || 0;
      const related = relatedF(ctx, fragment);
      const fProf = fragmentDomain(fragment) === ctx.actor.profession ? INFO_CFG.f_prof_match : INFO_CFG.f_prof_mismatch;
      const p = pGet(fProf, mismatch, alert, related, channel.access_c);
      const detection = detectionClass(ctx, target, false);
      const mutations = [{ type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) }];
      const sampled = detection === "SAMPLED";
      if (randomFloat() >= p) {
        // 复核失败的戒备抬升减半 (TODO(balance))：重读已见过的材料没有新痕迹。
        mutations.push({ type: "profession_alertness_delta", profession: target, value: INFO_CFG.alert_on_failed_probe * (recheck ? 0.5 : 1) });
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
        // 复核的观察 ID 带年月后缀：ObserveFact 按 observation_id 去重，
        // 固定 ID 会让重复核验被当成重放而静默无效。
        observation: recheck
          ? "gather:" + fragment + ":recheck:" + ctx.silo.current_year + "-" + ctx.silo.current_month
          : "gather:" + fragment,
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
        action_result: {
          executed: true,
          message: recheck
            ? "Re-examined the material on " + fragment + " against " + target + "'s records via " + channel.id + ". Your conviction deepens."
            : "Observed new material on " + fragment + " from " + target + " via " + channel.id + " (chance " + Math.round(p * 100) + "%).",
        },
      };
    },
  },
  {
    id: "action_share_info",
    action_type: "SHARE_INFO",
    label: "Share Info",
    description: "Share real or fake fragments with a department.",
    target_type: "DEPT",
    // AP 经济修复 (17-information-economy §3, TODO(balance)): 20 → 3。
    // 分享是流通的主动词：成本 ~ 月收入，才能维持跑赢淡忘的节奏。
    ap_cost: 3,
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
      for (const fragment of fragmentIds) {
        // §3.6 行动资格：B < 0.30 的材料只是私人笔记，不能作为事实分享。
        if (beliefB(ctx, fragment) < INFO_CFG.qual_rumor) {
          return { action_result: { executed: false, message: fragment + " is still private suspicion (belief below " + INFO_CFG.qual_rumor + "); it cannot be shared as a fact." } };
        }
      }
      // §3.6 分享接收：强度 = B_player·transmit_c·receptivity_g。接收方得到的是
      // 信念强度而不是曝光——私下确信永远不直接进 E/Q (§3.3)。
      // TODO(balance): transmit/receptivity 为 sim 起点。
      const connectionValue = actorConnectionTo(ctx, target);
      const transmit = 0.5 + 0.5 * connectionValue;
      const receptivity = 0.5 + 0.5 * ((profession.ideologies || {}).pro_foreign || 0);
      const mutations = [{ type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) }];
      for (const fragment of fragmentIds) {
        mutations.push({
          type: "profession_belief_absorb",
          profession: profession.name,
          fragment: fragment,
          source: "player",
          value: beliefB(ctx, fragment) * transmit * receptivity,
        });
      }
      return {
        mutations,
        action_result: {
          executed: true,
          message: "Shared " + fragmentIds.length + " fact(s) with " + profession.name + " (transmit " + transmit.toFixed(2) + ", receptivity " + receptivity.toFixed(2) + ").",
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
    // AP 经济修复 (17-information-economy §3, TODO(balance)): 15 → 4。
    ap_cost: 4,
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

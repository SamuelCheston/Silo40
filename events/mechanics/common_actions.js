defineMechanics([
  {
    id: "action_gather_info",
    action_type: "GATHER_INFO",
    label: "Gather Info",
    description:
      "收集可被本地核验的材料——这是整条线的起点：事实必须先在你手里成立，才谈得上脱离你的自述。探查所得会抬升你的确信（B），B ≥ 0.30 的材料才能作为事实分享；目标部门新材料枯竭后，复核已知材料仍能加深确信。成功取决于匹配度、戒备与渠道；失败会抬升目标戒备。",
    target_type: "DEPT",
    // AP 经济修复 (17-information-economy §3, TODO(balance)): 10 → 2。
    // 流通动词必须便宜到月收入 (~3/月) 可以支撑每月级的节奏。
    ap_cost: 2,
    duration_months: 0,
    apply(ctx) {
      const target = ctx.action.target_dept;
      if (!target) {
        return { action_result: { executed: false, message: "目标部门无效。" } };
      }
      const profession = professionByName(ctx, target);
      if (!profession) {
        return { action_result: { executed: false, message: "找不到目标部门。" } };
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
          return { action_result: { executed: false, message: "关于 " + target + "，你的部门已经知道一切。" } };
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
          action_result: { executed: true, message: "这次对 " + target + " 的探查一无所获（概率 " + Math.round(p * 100) + "%）。该部门戒备上升。" },
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
            ? "经 " + channel.id + " 把材料 " + fragment + " 对照 " + target + " 的记录复核了一遍。你的确信加深。"
            : "经 " + channel.id + " 从 " + target + " 观察到新材料 " + fragment + "（概率 " + Math.round(p * 100) + "%）。",
        },
      };
    },
  },
  {
    id: "action_share_info",
    action_type: "SHARE_INFO",
    label: "Share Info",
    description:
      "把事实交给能独立查账的人——从「你知道」走向「他们知道」的第一步。接收方吸收后，事实进入其信念层，开始不依赖你而存在，也开始随时间淡忘：流通跑赢淡忘才守得住。B < 0.30 的材料仍是私人猜测，不能作为事实分享；传输强度随你与目标部门的关系上升。",
    target_type: "DEPT",
    // AP 经济修复 (17-information-economy §3, TODO(balance)): 20 → 3。
    // 分享是流通的主动词：成本 ~ 月收入，才能维持跑赢淡忘的节奏。
    ap_cost: 3,
    duration_months: 0,
    apply(ctx) {
      const target = ctx.action.target_dept;
      const fragmentIds = ctx.action.fragment_ids || [];
      if (!target || fragmentIds.length === 0) {
        return { action_result: { executed: false, message: "目标无效，或未选择任何碎片。" } };
      }
      const profession = professionByName(ctx, target);
      if (!profession) {
        return { action_result: { executed: false, message: "找不到目标部门。" } };
      }
      // 20 分册 T2-3 分享目标收紧：不再是"拒绝"，而是"更贵"——戒备 ≥ 0.50
      // 的部门交接成本翻倍（服务端 resolveActionCost 计费），效果不变。
      // 项目硬约束：任何反制只抬成本，不永久锁死路径 (escape_response.go
      // 文件头纪律)；戒备由月度冷却回落 (DecayProfessionAlertness)。
      for (const fragment of fragmentIds) {
        // §3.6 行动资格：B < 0.30 的材料只是私人笔记，不能作为事实分享。
        if (beliefB(ctx, fragment) < INFO_CFG.qual_rumor) {
          return { action_result: { executed: false, message: fragment + " 仍是私人猜测（确信 B 低于 " + INFO_CFG.qual_rumor + "），不能作为事实分享。" } };
        }
      }
      // §3.6 分享接收：强度 = B_player·transmit_c·receptivity_g。接收方得到的是
      // 信念强度而不是曝光——私下确信永远不直接进 E/Q (§3.3)。
      // TODO(balance): transmit/receptivity 为 sim 起点。
      const connectionValue = actorConnectionTo(ctx, target);
      const transmit = 0.5 + 0.5 * connectionValue;
      const receptivity = 0.5 + 0.5 * ((profession.ideologies || {}).pro_foreign || 0);
      const mutations = [{ type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) }];
      // 已持有计数：接收部门信念层已不低于本次传入强度的事实，本次分享不再
      // 抬升它——反馈要说清楚，否则玩家看到的是"选了 3 份只吸收 2 份"。
      let alreadyHeld = 0;
      for (const fragment of fragmentIds) {
        const absorbed = beliefB(ctx, fragment) * transmit * receptivity;
        const existing = (profession.beliefs || []).find(function (b) { return b.fact_id === fragment; });
        if (existing && (existing.b || 0) >= absorbed) alreadyHeld++;
        mutations.push({
          type: "profession_belief_absorb",
          profession: profession.name,
          fragment: fragment,
          source: "player",
          value: absorbed,
        });
        // 开放痕迹 (《无人裁决》可达性)：被吸收的可核验事实让接收部门更开放。
        // 开放思潮与《共证》传播是同一条路：流通既沉淀信念，也改变思潮。
        // TODO(balance): 0.12 的推导——单次吸收量 ≈ 0.22–0.34，即每次分享给接收
        // 部门 cohort 亲外 +0.026–0.041；~12 次有效分享可抬一个部门 ~0.4。
        // 亲外以 0.35/年 向低目标回落，只有持续流通才守得住——不分享就回落。
        // 取 0.12（而非更低）是为了让"认真持续流通"在覆灭前夕把人口加权亲外
        // 推到 ~0.65，越过忠诚侵蚀后的门槛；零星分享达不到该水位。
        mutations.push({
          type: "cohort_ideology_delta_all",
          profession: profession.name,
          ideology: "pro_foreign",
          value: absorbed * 0.12,
        });
      }
      const heldNote = alreadyHeld > 0
        ? "其中 " + alreadyHeld + " 份对方已以不低于本次强度的水平持有，未再抬升。"
        : "";
      return {
        mutations,
        action_result: {
          executed: true,
          message:
            profession.name + " 把 " + fragmentIds.length + " 份事实吸收进信念层（传输 " + transmit.toFixed(2) + "，接收 " + receptivity.toFixed(2) + "）。" + heldNote + "信念随时间淡忘——持续流通才守得住。",
        },
      };
    },
  },
  {
    id: "action_build_connection",
    action_type: "BUILD_CONNECTION",
    label: "Build Network",
    description:
      "与一个部门建立信任。信任是流通的管道：关系越深，分享时的传输强度越高，好材料才接得住。没有信任的部门即使拿到事实，吸收也会大打折扣。",
    target_type: "DEPT",
    // AP 经济修复 (17-information-economy §3, TODO(balance)): 15 → 4。
    ap_cost: 4,
    duration_months: 1,
    apply(ctx) {
      const target = ctx.action.target_dept;
      if (!target) {
        return { action_result: { executed: false, message: "目标部门无效。" } };
      }
      const profession = professionByName(ctx, target);
      if (!profession) {
        return { action_result: { executed: false, message: "找不到目标部门。" } };
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
        action_result: { executed: true, message: "与 " + profession.name + " 的关系建立成功（+" + increase.toFixed(2) + "）。" },
      };
    },
  },
  {
    id: "action_incite_rebellion",
    action_type: "INCITE_REBELLION",
    label: "Incite Rebellion",
    description:
      "鼓动平民部门，向现有秩序施压。叛乱能打开窗口，也能把宿主掀翻、喂大侦察与整肃链——它不是《共证》的主路，只在需要倾斜走向时使用。",
    target_type: "NONE",
    ap_cost: 30,
    duration_months: 2,
    apply(ctx) {
      const mutations = [{ type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) }];
      const commoners = (ctx.silo.professions || []).filter(function (p) {
        return p.class_type === "COMMONER";
      });
      const baseEffect = 0.05 + (ctx.actor.political_prestige || 0) * 0.002;
      const propagandaMultiplier = 1 + (ctx.actor.propaganda_level || 0) * 0.2;
      // 效应量落地：煽动对叛乱有可观测、可复现的即时增量 (round 到千分位)，
      // 宣传等级以乘数作用于该增量。此前只加恐慌/思潮、叛乱不动而文案照报
      // 成功——欺骗性反馈不允许存在：没起作用就说没起作用。
      const rebellionShare = 1.0; // TODO(balance) 煽动转化为叛乱的份额
      const wanted = Math.round(baseEffect * propagandaMultiplier * rebellionShare * 1000) / 1000;
      const currentRebellion = (ctx.silo.metrics || {}).rebellion || 0;
      const headroom = Math.round((1.0 - currentRebellion) * 1000) / 1000;
      // 叛乱增量只归属玩家 (代表) 的煽动：反应链衡量的是玩家对秩序的推动；
      // NPC 的煽动只保留恐慌效应 (他们的 incite 是局部动荡 = panic)，否则任何
      // 高恐慌玩法都会被 NPC 月度煽动泵起全局叛乱，链就失去归因意义。
      const applied = commoners.length > 0 && ctx.actor.kind === "PLAYER"
        ? Math.min(wanted, headroom) : 0;
      if (applied > 0) {
        mutations.push({ type: "silo_metric_delta", metric: "rebellion", value: applied });
      }
      for (const profession of commoners) {
        const connectionValue = actorConnectionTo(ctx, profession.name);
        const finalEffect = baseEffect * (1 + connectionValue) * propagandaMultiplier;
        mutations.push({ type: "profession_metric_delta", profession: profession.name, field: "panic_value", value: finalEffect });
        mutations.push({ type: "profession_ideology_delta", profession: profession.name, ideology: "pro_foreign", value: finalEffect * 0.5 });
      }
      mutations.push(suspicionDelta(ctx, 0.05));
      let message;
      if (ctx.actor.kind !== "PLAYER") {
        // NPC 的煽动只作用于恐慌/思潮 (局部动荡)，全局叛乱增量只归属玩家。
        message = "Incited unrest among all commoner departments.";
      } else if (applied <= 0) {
        message = commoners.length === 0
          ? "这一点没有起作用：驻地没有可被煽动的平民部门。"
          : "这一点没有起作用：叛乱已经顶到上限，煽动只留下痕迹。";
      } else {
        message = "Incited unrest among all commoner departments. Rebellion +" +
          applied.toFixed(3) + " (propaganda multiplier x" + propagandaMultiplier.toFixed(1) + ").";
      }
      return {
        mutations,
        action_result: { executed: true, message },
      };
    },
  },
  {
    id: "action_conduct_propaganda",
    action_type: "CONDUCT_PROPAGANDA",
    label: "Propaganda",
    description:
      "提高宣传力度，扩大你声音的触达。注意：声量不等于可核验性——宣传为事实开路，事实本身仍要交给能查账的人才算数。",
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
        action_result: { executed: true, message: "宣传已展开。宣传力度 +1.0。" },
      };
    },
  },
  {
    id: "action_publicize_faction",
    action_type: "PUBLICIZE_FACTION",
    label: "Publicize Faction",
    description:
      "公开你的派系，用隐匿换取威望与公开位置。这是一条单行道：公开后暴露与终止链风险随之上升，只有在公开位置能换来别处换不到的东西时才值得。",
    target_type: "NONE",
    ap_cost: 25,
    duration_months: 0,
    apply(ctx) {
      const faction = ctx.actor.faction;
      if (!ctx.actor.is_representative || !faction) {
        return { action_result: { executed: false, message: "只有派系代表才能公开自己的派系。" } };
      }
      if (faction.is_public) {
        return { action_result: { executed: false, message: "派系已经公开。" } };
      }
      return {
        mutations: [
          { type: "faction_is_public_actor", bool_value: true },
          { type: "faction_metric_delta_actor", field: "prestige", value: 15.0 },
          { type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) },
        ],
        action_result: { executed: true, message: faction.name + " 已公开。地堡现在知道了你的存在。" },
      };
    },
  },
]);

// suspicionDelta / detectionClass live in the mechanic prelude; detection is
// structural (UNSAMPLED / SAMPLED / DIRECT_PROTOCOL), not a profession factor.

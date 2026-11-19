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
      return applyGatherProbe(ctx, false);
    },
  },
  {
    // 无声探查 (19-opening-variability §4.2)：隐秘行事特质解锁的探查变体。
    // 效果与 Gather Info 相同，但检测分类强制 UNSAMPLED——不进抽样、无怀疑
    // 增量，AP 成本由服务端按普通探查的 2 倍结算 (resolveActionCost)。
    // 边界：不豁免暴露水位 E，失败探查对目标部门的戒备抬升照旧——"你不被
    // 怀疑，但你的动作仍然有痕迹"。
    id: "action_silent_probe",
    action_type: "SILENT_PROBE",
    label: "Silent Probe",
    description:
      "隐秘行事者的无声探查：与普通探查同效，但不进入监控抽样、不产生怀疑增量——代价是 2 倍 AP。你不被怀疑，但动作仍有痕迹：失败仍会抬升目标戒备，暴露水位照旧累积。",
    target_type: "DEPT",
    // 基准 = 普通探查 ×2；服务端 resolveActionCost 按 2×(含规范冲突上浮) 权威结算。
    ap_cost: 4,
    duration_months: 0,
    apply(ctx) {
      // 动作门控已按特质解锁，这里是防绕过的纵深守卫 (服务端同样校验)。
      if (((ctx.actor && ctx.actor.traits) || []).indexOf("隐秘行事") === -1) {
        return { action_result: { executed: false, message: "无声探查需要特质：隐秘行事。" } };
      }
      return applyGatherProbe(ctx, true);
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
      for (const fragment of fragmentIds) {
        const absorbed = beliefB(ctx, fragment) * transmit * receptivity;
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
        // 20 分册 §3.2：玩家的一次流通会在反制方 (IT/Judicial) 心里留下
        // "有人在传这个事实"的观察。走既有部门信念吸收与同源折减，只淡忘、
        // 不计入成熟事实；T1-3/T3-2 读的正是这条泄漏确信。
        if (ctx.actor.kind === "PLAYER") {
          for (const watcher of ["IT", "Judicial"]) {
            if (!professionByName(ctx, watcher)) {
              continue;
            }
            mutations.push({
              type: "profession_belief_absorb",
              profession: watcher,
              fragment: "leak:" + fragment,
              source: "player",
              value: absorbed,
            });
          }
        }
      }
      // C8 搭载：流通场合人多耳杂，分享成功可能顺带传出派系传闻
      // (Go 权威侧掷骰；玩家确定性搭载不消耗 RNG)。
      mutations.push({ type: "faction_rumor_piggyback" });
      return {
        mutations,
        action_result: {
          executed: true,
          message:
            "已向 " + profession.name + " 转交 " + fragmentIds.length + " 份材料（传输 " + transmit.toFixed(2) + "，接收 " + receptivity.toFixed(2) + "）。同源转述的增益会打折，部门信念也会随时间淡忘。",
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
      // 20 分册 T1-3：部门对泄漏事实的确信越线后，关系推进减半 (12 月)。
      // 冷却由月度反制账本维护；回执只描述"对方比上次谨慎"，不报数值。
      const cooling = (profession.relation_cooldown || 0) > 0;
      if (cooling) {
        increase *= 0.5;
      }
      const current = actorConnectionTo(ctx, target);
      const applied = clampUnit(current + increase) - current;
      const message = applied <= 0
        ? "与 " + profession.name + " 的人脉已达上限。"
        : cooling
          ? "与 " + profession.name + " 的人脉加深" + (applied >= 0.005 ? "（+" + applied.toFixed(2) + "）。" : "。") + "对方比上次谨慎，谈话一直停在例行的寒暄上。"
          : "与 " + profession.name + " 的人脉加深" + (applied >= 0.005 ? "（+" + applied.toFixed(2) + "）。" : "。");
      return {
        mutations: [
          { type: "actor_connection_delta", connection_dept: target, value: increase },
          { type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) },
          suspicionDelta(ctx, 0.01),
        ],
        action_result: { executed: true, message },
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
          : "煽动没有起作用于全局叛乱度，仍可能加剧平民部门的局部动荡。";
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
      "提高宣传力度，让下一次煽动更有声势。可核验的材料仍需单独交给能查账的人。",
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
      "让驻地知道你的派系。公开后派系威望增加 15，派系不再隐匿。",
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
        action_result: { executed: true, message: faction.name + " 已公开。派系威望 +15。" },
      };
    },
  },
  {
    id: "action_found_faction",
    action_type: "FOUND_FACTION",
    label: "Found Faction",
    description:
      "发起派系聚类事件，创立自己的派系。需要政治威望 ≥ 50、意识形态强度 ≥ 0.25，消耗 20 AP。零成员派系合法，筹备期内凑不够人数则消亡。",
    target_type: "NONE",
    ap_cost: 20,
    duration_months: 0,
    apply(ctx) {
      // 预检与 Go 侧 foundFactionEligible 同口径，只为给出失败文案；
      // 权威判定与 AP 扣除在 faction_found_actor 处理器中原子完成。
      const prestige = ctx.actor.political_prestige || 0;
      if (prestige < 50) {
        return { action_result: { executed: false, message: "政治威望不足 50，还没有人愿意追随你。" } };
      }
      const pf = (ctx.actor.ideologies || {}).pro_foreign || 0;
      if (Math.abs(pf - 0.5) < 0.25) {
        return { action_result: { executed: false, message: "意识形态立场不够鲜明，无法凝聚追随者 (强度需 ≥ 0.25)。" } };
      }
      const faction = ctx.actor.faction;
      if (faction && faction.signature !== "special:unaffiliated") {
        return { action_result: { executed: false, message: "你已身处政治派系，不能再发起新派系。" } };
      }
      if ((ctx.actor.action_points || 0) < 20) {
        return { action_result: { executed: false, message: "AP 不足，发起派系聚类事件需要 20 AP。" } };
      }
      return {
        mutations: [{ type: "faction_found_actor" }],
        action_result: {
          executed: true,
          message: "你发起了派系聚类事件，「" + ctx.actor.label + "的派系」成立 (零成员筹备期)。",
        },
      };
    },
  },
  {
    id: "action_join_faction",
    action_type: "JOIN_FACTION",
    label: "Join Faction",
    description:
      "直选加入一个已感知的派系 (玩家不掷骰)。需无政治派系归属，消耗 10 AP。目标派系由 target_faction_id 指定。",
    target_type: "NONE",
    ap_cost: 10,
    duration_months: 0,
    apply(ctx) {
      // 预检与 Go 侧 joinFactionEligible 同口径，只为给出失败文案；
      // 权威判定 (含目标存在/无阵营桶排除) 与 AP 扣除在 faction_join_actor
      // 处理器中原子完成。玩家直选不受感知地板/关系闸约束 (ADR-0010)。
      const targetID = ctx.action.target_faction_id || 0;
      if (!targetID) {
        return { action_result: { executed: false, message: "未指定要加入的派系。" } };
      }
      const factions = ctx.silo.factions || [];
      const target = factions.find((f) => f.id === targetID);
      if (!target) {
        return { action_result: { executed: false, message: "目标派系不存在。" } };
      }
      const faction = ctx.actor.faction;
      if (faction && faction.signature !== "special:unaffiliated") {
        return { action_result: { executed: false, message: "你已身处政治派系，不能再加入其他派系。" } };
      }
      if ((ctx.actor.action_points || 0) < 10) {
        return { action_result: { executed: false, message: "AP 不足，加入派系需要 10 AP。" } };
      }
      return {
        mutations: [{ type: "faction_join_actor", int_value: targetID }],
        action_result: { executed: true, message: "你加入了「" + target.name + "」。" },
      };
    },
  },
  {
    id: "action_merge_faction",
    action_type: "MERGE_FACTION",
    label: "Merge Faction",
    description:
      "把另一个派系并入你领导的派系。需双方领袖个人关系 ≥ 0.60、思潮距离在亲缘半径内、AP ≥ 15。存续方保留 ID，被并方成员转移、派系除名。零成员派系也可发起/被合并。",
    target_type: "NONE",
    ap_cost: 15,
    duration_months: 0,
    apply(ctx) {
      // 预检与 Go 侧 mergeFactionEligible 同口径，只为给出失败文案；
      // 权威判定 (领袖身份/关系/思潮/AP) 与成员转移在 faction_merge_actor
      // 处理器中原子完成。合并是两位领袖的合意，关系闸对玩家同样生效。
      const targetID = ctx.action.target_faction_id || 0;
      if (!targetID) {
        return { action_result: { executed: false, message: "未指定要合并的派系。" } };
      }
      const factions = ctx.silo.factions || [];
      const target = factions.find((f) => f.id === targetID);
      if (!target) {
        return { action_result: { executed: false, message: "目标派系不存在。" } };
      }
      const faction = ctx.actor.faction;
      if (!ctx.actor.is_representative || !faction || faction.signature === "special:unaffiliated") {
        return { action_result: { executed: false, message: "只有派系领袖才能发起合并。" } };
      }
      if (faction.id === targetID) {
        return { action_result: { executed: false, message: "不能与自身合并。" } };
      }
      if ((ctx.actor.action_points || 0) < 15) {
        return { action_result: { executed: false, message: "AP 不足，合并派系需要 15 AP。" } };
      }
      return {
        mutations: [{ type: "faction_merge_actor", int_value: targetID }],
        action_result: { executed: true, message: "你把「" + target.name + "」并入「" + faction.name + "」。" },
      };
    },
  },
]);

// applyGatherProbe 是 GATHER_INFO 与 SILENT_PROBE 的共享结算体 (19 §4.2)：
// silent=false 与普通探查逐位一致 (同一 RNG 流、同一抽样顺序)；silent=true
// 只改检测分类——强制 UNSAMPLED，不产生怀疑增量，其余 (碎片抽样、获取概率、
// 复核路径、失败戒备抬升、AP 结算) 全部照旧。"你不被怀疑，但动作仍有痕迹"。
function applyGatherProbe(ctx, silent) {
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
  // 无声探查强制 UNSAMPLED (19 §4.2)：不进入抽样、无怀疑增量。
  const detection = silent ? "UNSAMPLED" : detectionClass(ctx, target, false);
  const mutations = [{ type: "actor_metric_delta", field: "action_points", value: -(ctx.action.cost || 0) }];
  const sampled = detection === "SAMPLED";
  const probeName = silent ? "无声探查" : "探查";
  if (randomFloat() >= p) {
    // 复核失败的戒备抬升减半 (TODO(balance))：重读已见过的材料没有新痕迹。
    // 无声探查不豁免戒备：动作痕迹照旧 (19 §4.2 边界)。
    mutations.push({ type: "profession_alertness_delta", profession: target, value: INFO_CFG.alert_on_failed_probe * (recheck ? 0.5 : 1) });
    if (sampled) {
      const fit = ownDept ? INFO_CFG.probe_fit_own : INFO_CFG.probe_fit_cross;
      mutations.push({
        type: "actor_metric_delta",
        field: "suspicion_level",
        value: sampledSuspicion(INFO_CFG.probe_base_suspicion, INFO_CFG.probe_visibility, fit, INFO_CFG.probe_witness, INFO_CFG.probe_record, mismatch) * archetypeDetectionFactor(ctx),
      });
    }
    return {
      mutations,
      action_result: { executed: true, message: "这次对 " + target + " 的" + probeName + "一无所获（概率 " + Math.round(p * 100) + "%）。该部门戒备上升。" },
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
      value: sampledSuspicion(INFO_CFG.probe_base_suspicion, INFO_CFG.probe_visibility, fit, INFO_CFG.probe_witness, INFO_CFG.probe_record, mismatch) * archetypeDetectionFactor(ctx),
    });
  }
  // C8 搭载：探查成功 (含复核) 可能顺带听到派系传闻；掷骰与吸收在
  // Go 权威侧 (faction_rumor_piggyback)，无政治派系时零 RNG。
  mutations.push({ type: "faction_rumor_piggyback" });
  return {
    mutations,
    action_result: {
      executed: true,
      message: recheck
        ? "经 " + channel.id + " 把材料 " + fragment + " 对照 " + target + " 的记录复核了一遍。你的确信加深。"
        : "经 " + channel.id + " 从 " + target + " 观察到新材料 " + fragment + "（概率 " + Math.round(p * 100) + "%）。",
    },
  };
}

// suspicionDelta / detectionClass live in the mechanic prelude; detection is
// structural (UNSAMPLED / SAMPLED / DIRECT_PROTOCOL), not a profession factor.

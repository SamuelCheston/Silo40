import { AgentActionType, ACTION_COSTS, AgentAction, Silo, ALL_FRAGMENTS } from './models';
import { ActorView } from './actor';
import { getProfessionActions } from './professionActions';

// ============ NPC 决策层 (文档 M2：效用函数先行) ============
/**
 * NpcBrain 只负责"决策"：根据 ActorView (统一状态视图) 与地堡世界状态，
 * 在 AP 预算内挑选一个动作。决策结果通过 submitAction 进入统一执行管线，
 * 与玩家动作完全同规则 (成本 / 怀疑度 / 职业修正)。
 */

export interface NpcDecision {
  action: AgentAction;
  /** 动作完成后的日志文案 (由调用方拼上 actor label) */
  message: string;
  /** 日志可见概率 (保持"流言"氛围，不暴露机制) */
  logChance: number;
}

interface WeightedDecision extends NpcDecision {
  weight: number;
}

export class NpcBrain {
  /** 每年尝试行动的平均次数 (按 deltaYears 缩放) */
  private static readonly FREQUENCY_PER_YEAR = 1.2;

  /**
   * 决策入口：返回待提交动作；无可行动作或概率未命中时返回 null。
   * @param deltaYears 时间片长度 (决定本次行动尝试概率)
   */
  public static decide(view: ActorView, silo: Silo, deltaYears: number): NpcDecision | null {
    const attemptChance = Math.min(1.0, this.FREQUENCY_PER_YEAR * deltaYears);
    if (Math.random() > attemptChance) return null;

    const candidates = this.scoreCandidates(view, silo);
    if (candidates.length === 0) return null;

    // 按效用权重加权随机
    const total = candidates.reduce((s, c) => s + c.weight, 0);
    let roll = Math.random() * total;
    for (const c of candidates) {
      roll -= c.weight;
      if (roll <= 0) return { action: c.action, message: c.message, logChance: c.logChance };
    }
    return candidates[candidates.length - 1];
  }

  private static scoreCandidates(view: ActorView, silo: Silo): WeightedDecision[] {
    const candidates: WeightedDecision[] = [];
    const ap = view.actionPoints;
    const ownId = view.ref.professionId;
    const depts = silo.professions ?? [];

    const add = (type: AgentActionType, weight: number, message: string, logChance = 0.5) => {
      const cost = ACTION_COSTS[type];
      if (cost > ap) return; // 决策前预算检查：只提交可执行动作
      candidates.push({
        action: { type, cost },
        weight: Math.max(0, weight),
        message,
        logChance,
      });
    };

    // --- GATHER_INFO：还有未知目标碎片时搜集 ---
    const gatherable = depts.filter(d => {
      const targetFrags = ALL_FRAGMENTS.filter(f => f.startsWith(d.name + '_'));
      return targetFrags.some(f => !view.knownFragments.includes(f));
    });
    if (gatherable.length > 0) {
      const target = gatherable[Math.floor(Math.random() * gatherable.length)];
      const knowRatio = view.knownFragments.length / ALL_FRAGMENTS.length;
      add('GATHER_INFO', 1.5 + (1 - knowRatio) * 1.5, `is secretly gathering intel on ${target.name}.`, 0.4);
    }

    // --- SHARE_INFO：有高信任盟友且持有碎片时共享 ---
    const allies = depts.filter(d => d.id !== ownId && view.getConnection(d.id) >= 0.8);
    if (view.knownFragments.length > 0 && allies.length > 0) {
      const target = allies[Math.floor(Math.random() * allies.length)];
      const frag = view.knownFragments[Math.floor(Math.random() * view.knownFragments.length)];
      add('SHARE_INFO', 1.2, `confidentially shared ${frag} secrets with ${target.name}.`, 0.8);
    }

    // --- BUILD_CONNECTION：人脉薄弱时结交 ---
    const weakTargets = depts.filter(d => d.id !== ownId && view.getConnection(d.id) < 0.8);
    if (weakTargets.length > 0) {
      // 挑人脉最薄弱的部门，效用随缺口增大
      const weakest = weakTargets.reduce((a, b) =>
        view.getConnection(a.id) <= view.getConnection(b.id) ? a : b
      );
      const deficit = 0.8 - view.getConnection(weakest.id);
      add('BUILD_CONNECTION', 1.0 + deficit * 2.0, `strengthened ties with ${weakest.name}.`, 0.5);
    }

    // --- CONDUCT_PROPAGANDA：宣传力度不足时宣传 ---
    if (view.propagandaLevel < 3.0) {
      add('CONDUCT_PROPAGANDA', 0.8, `conducted propaganda to boost influence.`, 0.4);
    }

    // --- INCITE_REBELLION：平民恐慌严重且自身怀疑度低时煽动 ---
    const avgPanic =
      depts.length > 0 ? depts.reduce((s, d) => s + d.panic_value, 0) / depts.length : 0;
    const hasCommoners = depts.some(d => d.class_type === 'COMMONER');
    if (hasCommoners && avgPanic > 0.4 && view.suspicionLevel < 0.5) {
      add('INCITE_REBELLION', 0.8, `incited unrest among the commoners.`, 0.3);
    }

    // --- 职业专属行动：按职业注册表挑选，走统一执行管线 (与玩家同规则) ---
    const RESOURCE_TYPES = ['Energy', 'Materials', 'Supplies'];
    for (const def of getProfessionActions(view.profession)) {
      if (def.apCost > ap) continue;
      let target_dept: string | undefined;
      let resource_target: string | undefined;
      if (def.targetType === 'DEPT') {
        const others = depts.filter(d => d.id !== ownId);
        if (others.length === 0) continue;
        target_dept = others[Math.floor(Math.random() * others.length)].name;
      } else if (def.targetType === 'RESOURCE') {
        resource_target = RESOURCE_TYPES[Math.floor(Math.random() * RESOURCE_TYPES.length)];
      }
      candidates.push({
        action: { type: 'PROFESSION_ACTION', profession_action: def.id, target_dept, resource_target, cost: def.apCost },
        weight: 0.8,
        message: `performed ${def.label}.`,
        logChance: 0.35,
      });
    }

    return candidates;
  }
}

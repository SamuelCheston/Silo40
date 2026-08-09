import { Agent, Profession, Silo } from './models';

// ============ Actor 抽象 (文档 M0：状态归一) ============
/**
 * 统一 Actor 模型：
 * - PLAYER：特工 Agent (由 UI 控制)
 * - NPC：部门 Profession (由 NpcBrain 决策层控制)
 *
 * 执行引擎只认识 ActorView (统一状态视图)，不区分来源 ——
 * 玩家与 NPC 通过同一入口 submitAction 提交动作，共用同一套 AP/怀疑度/成本体系。
 */

export type ActorKind = 'PLAYER' | 'NPC';

export interface ActorRef {
  kind: ActorKind;
  /** PLAYER：特工 id */
  agentId?: number;
  /** 被控制的部门 id (NPC 必填；PLAYER 由 profession 推导) */
  professionId?: number;
}

export function createActorRefForAgent(agent: Agent, silo: Silo): ActorRef {
  const prof = silo.professions?.find(p => p.name === agent.profession);
  return { kind: 'PLAYER', agentId: agent.id, professionId: prof?.id };
}

export function createActorRefForProfession(prof: Profession): ActorRef {
  return { kind: 'NPC', professionId: prof.id };
}

export function createActorView(ref: ActorRef, silo: Silo, agent?: Agent): ActorView {
  if (ref.kind === 'PLAYER') {
    if (!agent) throw new Error('PLAYER actor requires agent reference.');
    return new ActorView(ref, agent, null, silo);
  }
  const prof = silo.professions?.find(p => p.id === ref.professionId);
  if (!prof) throw new Error(`NPC actor profession not found: ${ref.professionId}`);
  return new ActorView(ref, null, prof, silo);
}

/**
 * 统一状态视图：对 Agent / Profession 提供同构读写接口。
 * 执行管线 (engine) 与决策层 (NpcBrain) 均只依赖该视图。
 */
export class ActorView {
  constructor(
    public readonly ref: ActorRef,
    private readonly agent: Agent | null,
    private readonly prof: Profession | null,
    private readonly silo: Silo,
  ) {}

  get isPlayer(): boolean {
    return this.ref.kind === 'PLAYER';
  }

  /** 所属部门名称 (Agent.profession / Profession.name) */
  get profession(): string {
    return this.agent ? this.agent.profession : this.prof!.name;
  }

  /** 日志称谓：玩家显示特工名，NPC 显示部门名 */
  get label(): string {
    if (this.agent) return this.agent.name || this.agent.profession;
    return this.prof!.name;
  }

  // ============ 统一经济字段 (读写穿透到 Agent / Profession) ============

  get actionPoints(): number {
    return this.agent ? this.agent.action_points : (this.prof!.action_points ?? 0);
  }
  set actionPoints(v: number) {
    if (this.agent) this.agent.action_points = v;
    else this.prof!.action_points = v;
  }

  get suspicionLevel(): number {
    return this.agent ? this.agent.suspicion_level : (this.prof!.suspicion_level ?? 0);
  }
  set suspicionLevel(v: number) {
    if (this.agent) this.agent.suspicion_level = v;
    else this.prof!.suspicion_level = v;
  }

  get politicalPrestige(): number {
    return this.agent ? this.agent.political_prestige : (this.prof!.political_prestige ?? 0);
  }
  set politicalPrestige(v: number) {
    if (this.agent) this.agent.political_prestige = v;
    else this.prof!.political_prestige = v;
  }

  get propagandaLevel(): number {
    return this.agent ? this.agent.propaganda_level : (this.prof!.propaganda_level ?? 0);
  }
  set propagandaLevel(v: number) {
    if (this.agent) this.agent.propaganda_level = v;
    else this.prof!.propaganda_level = v;
  }

  get organizationFactor(): number {
    return this.agent ? this.agent.organization_factor : (this.prof!.organization_factor ?? 1.0);
  }
  set organizationFactor(v: number) {
    if (this.agent) this.agent.organization_factor = v;
    else this.prof!.organization_factor = v;
  }

  /** 政治点数：仅玩家特工拥有 (NPC 恒为 0) */
  get politicalPoints(): number {
    return this.agent ? this.agent.political_points : 0;
  }
  set politicalPoints(v: number) {
    if (this.agent) this.agent.political_points = v;
  }

  get traits(): string[] {
    if (this.agent) {
      if (!this.agent.traits) this.agent.traits = [];
      return this.agent.traits;
    }
    if (!this.prof!.traits) this.prof!.traits = [];
    return this.prof!.traits;
  }

  get knownFragments(): string[] {
    if (this.agent) {
      if (!this.agent.known_fragments) this.agent.known_fragments = [];
      return this.agent.known_fragments;
    }
    if (!this.prof!.known_fragments) this.prof!.known_fragments = [];
    return this.prof!.known_fragments;
  }

  // ============ 人脉 (统一按部门 id 访问) ============

  /** 该 Actor 对各地堡部门的人脉值数组 (用于平均威望计算) */
  connectionValues(): number[] {
    if (this.agent) {
      return (this.agent.connections ?? []).map(c => c.value);
    }
    return Object.values(this.prof!.relations ?? {});
  }

  getConnection(professionId: number): number {
    if (this.agent) {
      const c = this.agent.connections?.find(x => x.profession_id === professionId);
      return c ? c.value : 0;
    }
    const p = this.silo.professions?.find(x => x.id === professionId);
    return p ? (this.prof!.relations?.[p.name] ?? 0) : 0;
  }

  setConnection(professionId: number, value: number): void {
    if (this.agent) {
      if (!this.agent.connections) this.agent.connections = [];
      let c = this.agent.connections.find(x => x.profession_id === professionId);
      if (!c) {
        c = { id: Date.now(), agent_id: this.agent.id, profession_id: professionId, value: 0 };
        this.agent.connections.push(c);
      }
      c.value = value;
      return;
    }
    const p = this.silo.professions?.find(x => x.id === professionId);
    if (!p) return;
    if (!this.prof!.relations) this.prof!.relations = {};
    this.prof!.relations[p.name] = value;
  }

  addConnection(professionId: number, delta: number): void {
    this.setConnection(professionId, Math.min(1.0, Math.max(0, this.getConnection(professionId) + delta)));
  }
}

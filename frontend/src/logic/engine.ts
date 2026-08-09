import {
  Silo, Agent,
  StoryEvent, AgentAction, ALL_FRAGMENTS, ActionResult,
} from './models';
import { EventEngine, STORY_EVENT_TYPE } from './events';
import {
  EventBus, Scheduler, ConditionEngine, ScriptEngine, RuleEngine,
  TriggerEngine, EventContext, GameTime, GameState, GameEvent, createEvent, advanceTime,
} from './eventbus';
import { GameRule } from './eventbus/RuleEngine';
import { Trigger } from './eventbus/TriggerEngine';

// ============ 统一事件类型 (文档第一层：一切行为都是事件) ============
export const EVENT_TYPES = {
  TIME_TICK: 'TIME_TICK',           // 时间推进基准
  AGENT_UPDATE: 'AGENT_UPDATE',     // 特工状态更新
  RESOURCE_UPDATE: 'RESOURCE_UPDATE', // 资源结算
  METRICS_UPDATE: 'METRICS_UPDATE', // 地堡指标更新
  IDEOLOGY_UPDATE: 'IDEOLOGY_UPDATE', // 思潮演化
  NPC_ACTIONS: 'NPC_ACTIONS',       // NPC 自主行为
  VICTORY_CHECK: 'VICTORY_CHECK',   // 胜利判定
  STORY_EVENT: STORY_EVENT_TYPE,    // 剧情随机事件
  PLAYER_ACTION: 'PLAYER_ACTION',   // 玩家动作
  GAME_OVER: 'GAME_OVER',           // 游戏结束
} as const;

/**
 * 事件驱动版 GameEngine (文档第八层：推荐架构)
 *
 * 所有系统 (特工/资源/思潮/指标/NPC/胜利/剧情) 全部注册为 EventBus 的订阅者，
 * 由 TIME_TICK 编排触发；延时事件进入 Scheduler (MinHeap)；
 * 游戏规则由 RuleEngine 数据驱动；剧情触发由 TriggerEngine 条件驱动；
 * 防事件风暴由 EventContext (fired set + maxDepth) 保证。
 */
export class GameEngine {
  // ============ 事件驱动核心组件 (文档第八层) ============
  public readonly bus: EventBus;
  public readonly scheduler: Scheduler;
  public readonly conditionEngine: ConditionEngine;
  public readonly scriptEngine: ScriptEngine;
  public readonly ruleEngine: RuleEngine;
  public readonly triggerEngine: TriggerEngine;

  private readonly eventEngine: EventEngine = new EventEngine();

  // ============ 内部结果收集 (事件驱动下的同步返回) ============
  private tickStories: StoryEvent[] = [];
  private actionResult: ActionResult | null = null;
  private idCounter = 0;

  constructor() {
    this.bus = new EventBus();
    this.scheduler = new Scheduler();
    this.conditionEngine = new ConditionEngine();
    this.scriptEngine = new ScriptEngine();
    this.ruleEngine = new RuleEngine(this.conditionEngine, this.scriptEngine, this.bus);
    this.triggerEngine = new TriggerEngine();

    // 规则引擎的延时效果 → 进入 Scheduler
    this.ruleEngine.onSchedule = (event, delayMonths, source) => {
      const data = source.data as Record<string, unknown> & { silo?: Silo };
      if (data.silo) {
        const now: GameTime = { year: data.silo.current_year, month: data.silo.current_month };
        this.scheduler.schedule(event, advanceTime(now, delayMonths));
      }
    };

    this.registerSystems();
    this.registerConditions();
    this.registerScripts();
    this.registerRules();
    this.registerTriggers();
  }

  private nextId(): string {
    return String(++this.idCounter);
  }

  // ============ 系统注册：一切通过事件总线 (文档第二/八层) ============
  private registerSystems(): void {
    // --- 时间推进编排：依次触发各子系统 ---
    this.bus.subscribe(EVENT_TYPES.TIME_TICK, (event, ctx) => {
      const { silo, agent, deltaYears, addLog } = event.data as {
        silo: Silo; agent?: Agent; deltaYears: number;
        addLog?: (msg: string) => void;
      };

      this.bus.emit(createEvent(`agent_update#${this.nextId()}`, EVENT_TYPES.AGENT_UPDATE, {
        silo, agent, deltaYears, addLog,
      }), ctx);

      this.bus.emit(createEvent(`resource_update#${this.nextId()}`, EVENT_TYPES.RESOURCE_UPDATE, {
        silo, deltaYears,
      }), ctx);

      this.bus.emit(createEvent(`metrics_update#${this.nextId()}`, EVENT_TYPES.METRICS_UPDATE, {
        silo, deltaYears,
      }), ctx);

      this.bus.emit(createEvent(`ideology_update#${this.nextId()}`, EVENT_TYPES.IDEOLOGY_UPDATE, {
        silo, deltaYears,
      }), ctx);

      this.bus.emit(createEvent(`npc_actions#${this.nextId()}`, EVENT_TYPES.NPC_ACTIONS, {
        silo, agent, deltaYears, addLog,
      }), ctx);

      this.bus.emit(createEvent(`victory_check#${this.nextId()}`, EVENT_TYPES.VICTORY_CHECK, {
        silo, agent,
      }), ctx);
    });

    // --- 特工状态更新 ---
    this.bus.subscribe(EVENT_TYPES.AGENT_UPDATE, (event) => {
      const { silo, agent, deltaYears, addLog } = event.data as {
        silo?: Silo; agent: Agent; deltaYears: number;
        addLog?: (msg: string) => void;
      };
      this.updateAgentState(agent, deltaYears, silo, addLog);
    });

    // --- 资源结算 (含运作条件校验) ---
    this.bus.subscribe(EVENT_TYPES.RESOURCE_UPDATE, (event) => {
      const { silo, deltaYears } = event.data as { silo: Silo; deltaYears: number };
      this.checkOperationalConditions(silo, deltaYears);
      this.updateResources(silo, deltaYears);
    });

    // --- 地堡指标更新 (倒计时/叛乱/人口) ---
    this.bus.subscribe(EVENT_TYPES.METRICS_UPDATE, (event) => {
      const { silo, deltaYears } = event.data as { silo: Silo; deltaYears: number };
      this.updateSiloMetrics(silo, deltaYears);
    });

    // --- 思潮演化 ---
    this.bus.subscribe(EVENT_TYPES.IDEOLOGY_UPDATE, (event) => {
      const { silo, deltaYears } = event.data as { silo: Silo; deltaYears: number };
      this.updateIdeology(silo, deltaYears);
    });

    // --- NPC 自主行为 ---
    this.bus.subscribe(EVENT_TYPES.NPC_ACTIONS, (event) => {
      const { silo, agent, deltaYears, addLog } = event.data as {
        silo: Silo; agent?: Agent; deltaYears: number;
        addLog?: (msg: string) => void;
      };
      if (agent) this.triggerNPCActions(silo, agent, deltaYears, addLog);
    });

    // --- 胜利判定 + 分数结算 ---
    this.bus.subscribe(EVENT_TYPES.VICTORY_CHECK, (event) => {
      const { silo, agent } = event.data as { silo: Silo; agent?: Agent };
      this.checkVictoryConditions(silo, agent);

      // 游戏结束 → 计算最终评分并发布 GAME_OVER (新因果链)
      if (silo.victory_status?.is_won !== undefined && !silo.victory_status.score) {
        silo.victory_status.score = this.calculateScore(silo);
        this.bus.emit(createEvent(`game_over#${this.nextId()}`, EVENT_TYPES.GAME_OVER, { silo }), new EventContext());
      }
    });

    // --- 玩家动作执行 ---
    this.bus.subscribe(EVENT_TYPES.PLAYER_ACTION, (event) => {
      const { silo, agent, action } = event.data as { silo: Silo; agent: Agent; action: AgentAction };
      this.actionResult = this.executeActionInternal(silo, agent, action);
    });

    // --- 剧情随机事件效果应用 ---
    this.bus.subscribe(EVENT_TYPES.STORY_EVENT, (event) => {
      const { silo, story } = event.data as { silo: Silo; story: StoryEvent };
      story.effects(silo);
    });
  }

  // ============ 规则引擎：条件 / 脚本 / 规则 (文档第五层) ============
  private registerConditions(): void {
    this.conditionEngine.registerMany({
      water_low: (state) =>
        (state.silo.resources.find(r => r.type === 'Water')?.amount ?? Infinity) < 200,
      panic_high: (state) =>
        state.silo.professions.some(p => p.panic_value > 0.7),
      rebellion_high: (state) =>
        state.silo.rebellion > 0.6,
    });
  }

  private registerScripts(): void {
    this.scriptEngine.registerMany({
      // 恐慌蔓延 → 亲外度小幅上升 (恐慌→亲外转化)
      panic_to_ideology: (event, state) => {
        state.silo.professions.forEach(p => {
          p.ideology_value = Math.min(1.0, p.ideology_value + 0.02);
        });
        state.logs?.push('[Rule] 高恐慌情绪转化为对外部世界的好奇。');
      },
      // 高叛乱 → 增加基础死亡率之外的额外死亡
      rebellion_deaths: (event, state) => {
        const extra = Math.floor(state.silo.total_population * 0.005);
        state.silo.total_population = Math.max(0, state.silo.total_population - extra);
        state.logs?.push(`[Rule] 叛乱冲突造成约 ${extra} 人伤亡。`);
      },
      // 缺水 → 凝聚力缓慢下降
      water_shortage_cohesion: (event, state) => {
        state.silo.cohesion = Math.max(0, state.silo.cohesion - 0.02);
        state.logs?.push('[Rule] 供水短缺削弱了地堡的凝聚力。');
      },
    });
  }

  private registerRules(): void {
    const rules: GameRule[] = [
      {
        id: 'water_shortage_rule',
        trigger: { eventType: EVENT_TYPES.METRICS_UPDATE, condition: 'water_low' },
        effects: [
          { type: 'script', script: 'water_shortage_cohesion' },
          { type: 'script', script: 'panic_to_ideology' },
        ],
      },
      {
        id: 'rebellion_escalation_rule',
        trigger: { eventType: EVENT_TYPES.VICTORY_CHECK, condition: 'rebellion_high' },
        effects: [
          { type: 'script', script: 'rebellion_deaths' },
        ],
      },
      {
        // 延时事件示例：恐慌过高时，3 个月后触发 "暴乱爆发" 延时事件
        id: 'panic_storm_delay_rule',
        trigger: { eventType: EVENT_TYPES.VICTORY_CHECK, condition: 'panic_high' },
        effects: [
          {
            type: 'schedule_event',
            delayMonths: 3,
            event: createEvent('delayed_riot', 'DELAYED_RIOT', {
              // silo/agent 由规则引擎注入
            }),
          },
        ],
      },
    ];
    this.ruleEngine.registerMany(rules);

    // 延时事件订阅：暴乱爆发 → 恐慌加剧
    this.bus.subscribe('DELAYED_RIOT', (event) => {
      const state = event.data as Record<string, unknown> & { silo?: Silo };
      if (!state.silo) return;
      state.silo.professions.forEach(p => {
        p.panic_value = Math.min(1.0, p.panic_value + 0.15);
      });
      state.silo.legitimacy = Math.max(0, state.silo.legitimacy - 0.1);
    });
  }

  // ============ 剧情触发器 (文档第四层) ============
  private registerTriggers(): void {
    const triggers: Trigger[] = [
      {
        id: 'silo1_fallout',
        description: '一号地堡失联后，凝聚力下降、恐慌上升 (剧情链条起点)',
        condition: (state) => state.silo.silo1_destroyed && state.silo.cohesion > 0.3,
        effect: (bus, ctx, state) => {
          state.silo.cohesion = Math.max(0, state.silo.cohesion - 0.05);
          state.silo.professions.forEach(p => {
            p.panic_value = Math.min(1.0, p.panic_value + 0.05);
          });
          state.logs?.push('[Trigger] 一号地堡失联的传闻在居民中蔓延。');
        },
      },
      {
        id: 'pro_foreign_awakening',
        description: '多数部门亲外度超过 50% 时，社会进入觉醒阶段',
        condition: (state) =>
          state.silo.professions.filter(p => p.ideology_value > 0.5).length >= 4,
        effect: (bus, ctx, state) => {
          state.silo.legitimacy = Math.max(0, state.silo.legitimacy - 0.05);
          state.logs?.push('[Trigger] 社会思潮觉醒，旧秩序受到挑战。');
        },
      },
    ];
    this.triggerEngine.registerMany(triggers);
  }

  // ============ 对外接口 (兼容原 API，内部事件驱动) ============

  /** 推进一个游戏时间片：发布 TIME_TICK 并结算延时/剧情/随机事件 */
  public updateSiloState(silo: Silo, deltaYears: number, agent?: Agent, addLog?: (msg: string) => void): StoryEvent[] {
    this.tickStories = [];
    const ctx = new EventContext();

    // 刷新规则引擎状态视图 (规则/脚本将日志写入 state.logs)
    const state: GameState = { silo, agent, logs: [] };
    this.ruleEngine.lastState = state;

    // 1. 时间推进 (各系统订阅响应)
    this.bus.emit(createEvent(`tick#${this.nextId()}`, EVENT_TYPES.TIME_TICK, {
      silo, agent, deltaYears, addLog,
    }), ctx);

    // 2. 调度器：触发到期延时事件
    this.scheduler.tick({ year: silo.current_year, month: silo.current_month }, this.bus);

    // 3. 剧情触发器：条件检查
    this.triggerEngine.evaluate(this.bus, ctx, { silo, agent, logs: state.logs });

    // 4. 随机剧情事件
    const story = this.eventEngine.triggerRandomEvent(silo, this.bus, ctx);
    if (story) this.tickStories.push(story);

    // 将规则/触发器产生的日志回调给 UI
    if (addLog && state.logs) {
      for (const msg of state.logs) addLog(msg);
    }

    return this.tickStories;
  }

  /** 玩家执行动作 (发布 PLAYER_ACTION 事件) */
  public executeAgentAction(silo: Silo, agent: Agent, action: AgentAction): ActionResult {
    this.actionResult = null;
    const ctx = new EventContext();
    this.ruleEngine.lastState = { silo, agent };

    this.bus.emit(createEvent(`action#${this.nextId()}`, EVENT_TYPES.PLAYER_ACTION, {
      silo, agent, action,
    }), ctx);

    return this.actionResult ?? { executed: false, message: 'Unknown error.' };
  }

  /** 注册延时事件到调度器 */
  public scheduleEvent(event: import('./eventbus').GameEvent, at: GameTime): void {
    this.scheduler.schedule(event, at);
  }

  /** 特工状态更新 (保留原公开 API，内部逻辑不变) */
  public updateAgentState(agent: Agent, deltaYears: number, silo?: Silo, addLog?: (msg: string) => void): void {
    if (deltaYears <= 0) return;

    // Medical profession passive trait: randomly gain information fragments over time
    if (agent.profession === 'Medical' && silo && addLog) {
      if (Math.random() < 0.2) { // 20% chance per year
        const availableFragments = ALL_FRAGMENTS.filter(f => !agent.known_fragments.includes(f));
        if (availableFragments.length > 0) {
          const randomFragment = availableFragments[Math.floor(Math.random() * availableFragments.length)];
          agent.known_fragments.push(randomFragment);
          addLog(`[Medical Passive] Your medical duties allowed you to overhear rumors, gaining information about ${randomFragment}.`);
        }
      }
    }

    // 1. 计算平均人脉值 (0.0 - 1.0)
    let totalConnection = 0;
    const count = agent.connections?.length || 0;
    if (count > 0) {
      agent.connections.forEach((conn) => {
        totalConnection += conn.value;
      });
      totalConnection /= count;
    }

    // 2. 计算职业修正系数
    const profFactor = this.professionFactors[agent.profession] || 0;

    // 3. 计算特质修正系数
    let traitFactor = 0;
    agent.traits?.forEach((trait) => {
      traitFactor += this.traitFactors[trait] || 0;
    });

    // 4. 计算政治威望
    agent.political_prestige = totalConnection * 100 * (1 + profFactor) * (1 + traitFactor);
    if (agent.political_prestige < 0) agent.political_prestige = 0;

    // 5. 给予政治点数和行动点数 (AP)
    const pointGainRate = 0.1;
    agent.political_points += agent.political_prestige * pointGainRate * deltaYears;

    // 行动点数恢复：基础恢复 10 点/年，受威望和组织度加成
    const apGainRate = 10 + (agent.political_prestige * 0.05) + (agent.organization_factor * 2);
    agent.action_points += apGainRate * deltaYears;
    // 设置 AP 上限
    const maxAp = 100 + (agent.organization_factor * 10);
    if (agent.action_points > maxAp) {
      agent.action_points = maxAp;
    }

    // 6. 怀疑度随时间衰减
    const suspicionDecayRate = 0.05; // 每年降低5%
    if (agent.suspicion_level > 0) {
      agent.suspicion_level -= suspicionDecayRate * deltaYears;
      if (agent.suspicion_level < 0) agent.suspicion_level = 0;
    }
  }

  /** 特工执行动作内部实现 (由 PLAYER_ACTION 订阅者调用) */
  private executeActionInternal(silo: Silo, agent: Agent, action: AgentAction): ActionResult {
    if (agent.action_points < action.cost) {
      return { executed: false, message: "Not enough Action Points (AP)." };
    }

    const preSuspicion = agent.suspicion_level || 0;
    let result: ActionResult = { executed: false, message: "" };

    switch (action.type) {
      case 'GATHER_INFO':
        result = this.gatherInformation(silo, agent, action);
        break;
      case 'SHARE_INFO':
        result = this.shareInformation(silo, agent, action);
        break;
      case 'BUILD_CONNECTION':
        result = this.buildConnection(silo, agent, action);
        break;
      case 'INCITE_REBELLION':
        result = this.inciteRebellion(silo, agent, action);
        break;
      case 'CONDUCT_PROPAGANDA':
        result = this.conductPropaganda(silo, agent, action);
        break;
    }

    if (result.executed) {
      let gained = (agent.suspicion_level || 0) - preSuspicion;

      // 基础行为怀疑度惩罚 (兜底产生)
      if (action.type === 'INCITE_REBELLION') gained += 0.05;
      else if (action.type === 'SHARE_INFO') gained += 0.01;
      else if (action.type === 'BUILD_CONNECTION') gained += 0.01;
      else if (action.type === 'GATHER_INFO') gained += 0.005;
      else if (action.type === 'CONDUCT_PROPAGANDA') gained += 0.02;

      // 职业修正
      if (agent.profession === 'Mayor') {
        gained *= 3.0;
      } else if (agent.profession === 'IT') {
        gained = 0; // IT部门行动不增加怀疑度
      } else if (agent.profession === 'Police') {
        const discount = 0.5 + Math.random() * 0.4;
        gained *= discount;
      } else if (agent.profession === 'Mines') {
        gained *= 0.05;
      }

      // 特质修正
      if (agent.traits?.includes('隐秘行事')) {
        gained *= 0.8;
      }

      agent.suspicion_level = preSuspicion + gained;

      // IT 专属机制：恶化 safeguard 风险系数
      if (agent.profession === 'IT') {
        silo.safeguard_risk = (silo.safeguard_risk || 0) + (action.cost * 0.002);
      }
    }

    return result;
  }

  // 特工建立或强化与目标部门的人脉
  private buildConnection(silo: Silo, agent: Agent, action: AgentAction): ActionResult {
    if (!action.target_dept) return { executed: false, message: "Invalid target department." };

    const targetProf = silo.professions?.find(p => p.name === action.target_dept);
    if (!targetProf) return { executed: false, message: "Target department not found." };

    if (!agent.connections) agent.connections = [];

    let connection = agent.connections.find(c => c.profession_id === targetProf.id);
    if (!connection) {
      connection = { id: Date.now(), agent_id: agent.id, profession_id: targetProf.id, value: 0 };
      agent.connections.push(connection);
    }

    let increaseValue = 0.05 + (agent.political_prestige * 0.005);
    if (agent.traits?.includes('魅力非凡')) {
      increaseValue *= 1.5;
    }
    connection.value += increaseValue;
    if (connection.value > 1.0) connection.value = 1.0;

    agent.action_points -= action.cost;
    return { executed: true, message: `Successfully built connections with ${targetProf.name}.` };
  }

  // 特工煽动底层叛乱（全局增加所有平民阶层的恐慌和排外/亲外极端化）
  private inciteRebellion(silo: Silo, agent: Agent, action: AgentAction): ActionResult {
    const commoners = silo.professions?.filter(p => p.class_type === 'COMMONER') || [];
    if (commoners.length === 0) return { executed: false, message: "No commoner departments found to incite." };

    commoners.forEach(prof => {
      const connection = agent.connections?.find(c => c.profession_id === prof.id);
      const connectionValue = connection ? connection.value : 0;

      const baseEffect = 0.05 + (agent.political_prestige * 0.002);
      const propagandaMultiplier = 1 + (agent.propaganda_level || 0) * 0.2;
      const multiplier = (1 + connectionValue) * propagandaMultiplier;
      const finalEffect = baseEffect * multiplier;

      prof.panic_value += finalEffect;
      prof.ideology_value += finalEffect * 0.5;
    });

    agent.action_points -= action.cost;
    return { executed: true, message: `Incited unrest among all commoner departments.` };
  }

  // 特工主动进行宣传，提升宣传力度
  private conductPropaganda(silo: Silo, agent: Agent, action: AgentAction): ActionResult {
    agent.propaganda_level = (agent.propaganda_level || 0) + 1.0;
    agent.action_points -= action.cost;
    return { executed: true, message: `Conducted propaganda. Propaganda Level increased by 1.0.` };
  }

  // 特工搜集其他部门的信息碎片
  private gatherInformation(silo: Silo, agent: Agent, action: AgentAction): ActionResult {
    if (!action.target_dept) return { executed: false, message: "Invalid target department." };

    if (!agent.known_fragments) agent.known_fragments = [];

    const targetFragments = ALL_FRAGMENTS.filter(f => f.startsWith(action.target_dept! + '_'));
    const unknownTargetFragments = targetFragments.filter(f => !agent.known_fragments?.includes(f));

    if (unknownTargetFragments.length > 0) {
      const fragmentToGather = unknownTargetFragments[Math.floor(Math.random() * unknownTargetFragments.length)];
      agent.known_fragments.push(fragmentToGather);
      agent.action_points -= action.cost;
      return { executed: true, message: `Gathered intel on ${fragmentToGather}.` };
    }

    return { executed: false, message: `Your department already knows everything about ${action.target_dept}.` };
  }

  // 特工将自己掌握的信息碎片分享给目标部门
  private shareInformation(silo: Silo, agent: Agent, action: AgentAction): ActionResult {
    if (!action.target_dept || !action.fragment_ids || action.fragment_ids.length === 0) {
      return { executed: false, message: "Invalid target or no fragments selected." };
    }

    const targetProf = silo.professions?.find(p => p.name === action.target_dept);
    if (!targetProf) return { executed: false, message: "Target department not found." };

    const connection = agent.connections?.find(c => c.profession_id === targetProf.id);
    const connectionValue = connection ? connection.value : 0;

    // AP 即使被拒绝也会消耗
    agent.action_points -= action.cost;

    const unexplainedFragments = action.fragment_ids.filter(id => !agent.known_fragments.includes(id));
    const unexplainedCount = unexplainedFragments.length;

    if (unexplainedCount > 0) {
      const suspicionPenalty = (unexplainedCount * 0.1) + (Math.pow(unexplainedCount, 1.5) * 0.05);
      agent.suspicion_level = (agent.suspicion_level || 0) + suspicionPenalty;
    }

    let acceptanceRate = 0.1 + targetProf.ideology_value + connectionValue;
    acceptanceRate -= (unexplainedCount * 0.1);
    if (acceptanceRate < 0.05) acceptanceRate = 0.05;
    if (acceptanceRate > 1.0) acceptanceRate = 1.0;

    const roll = Math.random();
    if (roll > acceptanceRate) {
      return {
        executed: true,
        message: `Attempted to share info with ${targetProf.name}, but they rejected it! (Acceptance rate was ${(acceptanceRate * 100).toFixed(0)}%)`
      };
    }

    if (!targetProf.known_fragments) targetProf.known_fragments = [];
    for (const f of action.fragment_ids) {
      if (!targetProf.known_fragments.includes(f)) {
        targetProf.known_fragments.push(f);
      }
    }

    const panicImpact = 0.05 + unexplainedCount * 0.05;
    targetProf.panic_value = Math.min(1.0, targetProf.panic_value + panicImpact);

    if (connectionValue >= 0.3) {
      const ideologyImpact = 0.02 + unexplainedCount * 0.02;
      targetProf.ideology_value = Math.min(1.0, targetProf.ideology_value + ideologyImpact);
    }

    return {
      executed: true,
      message: `Successfully shared ${action.fragment_ids.length} fragments with ${targetProf.name}. (Included ${unexplainedCount} pieces of unexplained knowledge)`
    };
  }

  public getOrganizedPopulation(silo: Silo, agent: Agent): number {
    let organizedPopulation = 0;
    if (agent.connections && agent.connections.length > 0) {
      agent.connections.forEach(conn => {
        let orgFactor = agent.organization_factor || 1.0;
        if (agent.traits?.includes('魅力非凡')) {
          orgFactor *= 1.2;
        }

        const targetProf = silo.professions?.find(p => p.id === conn.profession_id);
        if (targetProf) {
          const isAgentCommoner = ['Supply', 'Mechanical', 'Mines', 'Agricultural'].includes(agent.profession);

          if (isAgentCommoner && targetProf.class_type === 'COMMONER') {
            if (agent.profession === 'Mechanical') {
              orgFactor *= 2.0;
            } else {
              orgFactor *= 1.5;
            }
          }

          let appeal = 0.1;

          if (agent.profession === 'Mechanical' && targetProf.name === 'Mechanical') {
            appeal += 0.4;
          }

          if (agent.traits?.includes('魅力非凡')) {
            appeal += 0.2;
          }

          const propagandaMultiplier = agent.propaganda_level || 0;

          const appealEffect = appeal * propagandaMultiplier;
          const conversionRate = (appealEffect * 0.4 + conn.value * 0.6) * orgFactor * targetProf.ideology_value;

          const maxConvertible = targetProf.population * 0.20;
          let deptOrganized = targetProf.population * conversionRate;

          if (deptOrganized > maxConvertible) {
            deptOrganized = maxConvertible;
          }

          organizedPopulation += deptOrganized;
        }
      });
    }
    return Math.floor(organizedPopulation);
  }

  // 模拟 NPC 部门的自主行为
  public triggerNPCActions(silo: Silo, agent: Agent, deltaYears: number, addLog?: (msg: string) => void): void {
    if (!silo.professions) return;

    silo.professions.forEach(prof => {
      if (prof.name === agent.profession) return;

      if (prof.name === 'Medical') {
        if (Math.random() < 0.2) {
          const unknownFragments = ALL_FRAGMENTS.filter(f => !prof.known_fragments?.includes(f));
          if (unknownFragments.length > 0) {
            const newFrag = unknownFragments[Math.floor(Math.random() * unknownFragments.length)];
            if (!prof.known_fragments) prof.known_fragments = [];
            prof.known_fragments.push(newFrag);
            if (Math.random() < 0.5 && addLog) {
              addLog(`Medical (NPC) overheard rumors and gained intel on ${newFrag}`);
            }
          }
        }
      }

      const ideology = prof.ideology_value;
      const willToAct = ideology > 0.4 ? ideology : 0.1;

      const actionChance = willToAct * (0.1 + prof.power_level * 0.05) * deltaYears;

      if (Math.random() < actionChance) {
        const actionType = Math.random();

        if (actionType < 0.4 && ideology > 0.4) {
          const unknownFragments = ALL_FRAGMENTS.filter(f => !prof.known_fragments?.includes(f));
          if (unknownFragments.length > 0) {
            const newFrag = unknownFragments[Math.floor(Math.random() * unknownFragments.length)];
            if (!prof.known_fragments) prof.known_fragments = [];
            prof.known_fragments.push(newFrag);
            if (Math.random() < 0.4 && addLog) {
              addLog(`${prof.name} is secretly gathering intel on ${newFrag}`);
            }
          }
        } else if (actionType < 0.8 && ideology > 0.4) {
          if (prof.known_fragments && prof.known_fragments.length > 0) {
            const fragmentToShare = prof.known_fragments[Math.floor(Math.random() * prof.known_fragments.length)];

            const targetCandidates = silo.professions!.filter(p =>
              p.id !== prof.id &&
              prof.relations && prof.relations[p.name] >= 0.8
            );

            if (targetCandidates.length > 0) {
              const targetProf = targetCandidates[Math.floor(Math.random() * targetCandidates.length)];

              if (!targetProf.known_fragments) targetProf.known_fragments = [];
              if (!targetProf.known_fragments.includes(fragmentToShare)) {
                targetProf.known_fragments.push(fragmentToShare);

                targetProf.panic_value = Math.min(1.0, targetProf.panic_value + 0.05);
                targetProf.ideology_value = Math.min(1.0, targetProf.ideology_value + 0.02);

                if (Math.random() < 0.8 && addLog) {
                  addLog(`${prof.name} confidentially shared ${fragmentToShare} secrets with ${targetProf.name}`);
                }
              }
            }
          }
        } else {
          if (prof.panic_value > 0.1) {
            prof.panic_value = Math.max(0, prof.panic_value - 0.15);
            if (Math.random() < 0.4 && addLog) {
              addLog(`${prof.name} suppressed their internal panic`);
            }
          } else {
            prof.productivity = Math.min(1.0, prof.productivity + 0.05);
          }
        }
      }
    });
  }

  private calculateScore(silo: Silo): any {
    const survival_points = silo.total_population * 1;

    const diversity_points = (silo.professions?.filter(p => p.productivity > 0.5).length || 0) * 100;

    const heritage_points = Math.floor((1.0 - silo.history_burden) * 500);

    let avgIdeology = 0;
    silo.professions?.forEach(p => avgIdeology += p.ideology_value);
    avgIdeology /= (silo.professions?.length || 1);
    const ideology_points = Math.floor(avgIdeology * 200);

    let multiplier = 1.0;
    switch (silo.victory_status?.type) {
      case 'INFORMATION': multiplier = 2.0; break;
      case 'TIME': multiplier = 1.5; break;
      case 'REBELLION': multiplier = 1.2; break;
      case 'EXCLUSIONIST': multiplier = 0.5; break;
      case 'DEATH': multiplier = 0; break;
      case 'AGENT_COMPROMISED': multiplier = 0; break;
    }

    const total = Math.floor((survival_points + diversity_points + heritage_points + ideology_points) * multiplier);

    return {
      total,
      survival_points,
      diversity_points,
      heritage_points,
      ideology_points,
      multiplier
    };
  }

  public getEndingNarrative(silo: Silo): string {
    if (!silo.victory_status) return "地堡的故事仍在继续...";

    let narrative = silo.victory_status.description + "\n\n";

    const proForeignRatio = (silo.professions?.filter(p => p.ideology_value > 0.5).length || 0) / (silo.professions?.length || 1);

    if (proForeignRatio > 0.7) {
      narrative += "地堡社会展现出了前所未有的开放性，人们渴望与外界建立联系。";
    } else if (proForeignRatio < 0.2) {
      narrative += "地堡社会深陷排外情绪，人们对任何来自外部的事物都充满敌意。";
    } else {
      narrative += "地堡社会在保守与开放之间艰难地维持着平衡。";
    }

    if (silo.history_burden > 0.5) {
      narrative += " 沉重的历史包袱如阴影般笼罩着每一个人，文明的进步举步维艰。";
    } else {
      narrative += " 过去的一页已被翻开，新的一代正以轻松的姿态面对未来。";
    }

    return narrative;
  }

  private checkVictoryConditions(silo: Silo, agent?: Agent): void {
    if (silo.victory_status?.is_won) return;

    // 0. Safeguard 危机 (IT专属)
    if (silo.safeguard_risk !== undefined && silo.safeguard_risk >= 1.0) {
      silo.victory_status = {
        is_won: false,
        type: 'DEATH',
        description: 'Safeguard 协议被激活。IT部门的过度干预触发了底层核心逻辑，清理程序启动，40号地堡被彻底清洗。',
      };
      return;
    }

    // 1. 信息胜利：每个部门至少获得5个其他部门的信息碎片
    let allDeptsHaveFragments = true;
    if (silo.professions && silo.professions.length > 0) {
      for (const prof of silo.professions) {
        const uniqueFragments = new Set(prof.known_fragments || []);
        if (uniqueFragments.size < 5) {
          allDeptsHaveFragments = false;
          break;
        }
      }
    } else {
      allDeptsHaveFragments = false;
    }

    if (allDeptsHaveFragments) {
      silo.victory_status = {
        is_won: true,
        type: 'INFORMATION',
        description: '你成功让真相在所有部门间流传。全知视角的拼图终于拼凑完整，地堡的居民迎来了觉醒的黎明。',
      };
      return;
    }

    // 2. 时间胜利判定：由"1号地堡覆灭"事件触发后结算 (后续详细实现)
    if (silo.silo1_destroyed) {
      silo.victory_status = {
        is_won: true,
        type: 'TIME',
        description: '一号地堡已经覆灭，控制网络断开。40号地堡迎来了属于自己的时间。',
      };
      return;
    }

    // 3. 叛乱胜利
    if (agent && silo.total_population > 0) {
      const SUSPICION_THRESHOLD = 1.0;
      if (agent.suspicion_level >= SUSPICION_THRESHOLD) {
        silo.victory_status = {
          is_won: false,
          type: 'AGENT_COMPROMISED',
          description: '由于传播过多掺杂了个人意图的虚假信息，你的特工身份彻底暴露。司法部已经下达了逮捕令。',
        };
        return;
      }

      let organizedPopulation = this.getOrganizedPopulation(silo, agent);

      if (organizedPopulation >= silo.total_population * 0.03) {
        const hasEnoughSurvivors = silo.total_population >= 10000 * 0.03;

        let escapingDeptsCount = 0;
        silo.professions?.forEach(p => {
          const escapingPeople = p.population * p.ideology_value;
          if (escapingPeople > 10) {
            escapingDeptsCount++;
          }
        });
        const hasLaborEscape = escapingDeptsCount >= 3;

        if (hasEnoughSurvivors || hasLaborEscape) {
          silo.victory_status = {
            is_won: true,
            type: 'REBELLION',
            description: '你成功组织了反抗力量并发动了叛乱。旧的统治被推翻，幸存者们冲破了封闭的牢笼。',
          };
          return;
        }
      }
    }

    // 4. 失败判定 (人口灭绝)
    if (silo.total_population <= 0) {
      silo.victory_status = {
        is_won: false,
        type: 'DEATH',
        description: '地堡内已无生命迹象。人类最后的堡垒沦为了一座寂静的坟墓。',
      };
      return;
    }
  }

  private checkOperationalConditions(silo: Silo, deltaYears: number): void {
    const proForeignDepts = silo.professions?.filter(p => p.ideology_value >= 0.1).length || 0;

    if (proForeignDepts < 3) {
      silo.history_burden += 0.05 * deltaYears;

      silo.professions?.forEach(p => {
        p.productivity -= 0.02 * deltaYears;
        if (p.productivity < 0.1) p.productivity = 0.1;
      });
    } else {
      silo.history_burden -= 0.01 * deltaYears;
      if (silo.history_burden < 0) silo.history_burden = 0;

      silo.professions?.forEach(p => {
        p.productivity += 0.01 * deltaYears;
        if (p.productivity > 1.0) p.productivity = 1.0;
      });
    }
  }

  private updateIdeology(silo: Silo, deltaYears: number): void {
    silo.professions?.forEach((p) => {
      const stability = silo.cohesion;

      if (p.panic_value > 0.3 && stability < 0.5) {
        const drift = p.panic_value * (1.0 - stability) * deltaYears * 0.01;
        p.ideology_value += drift;
      }

      if (p.panic_value > 0) {
        const conversionRate = 0.10;
        const convertedAmount = p.panic_value * conversionRate * deltaYears;

        p.panic_value -= convertedAmount;
        if (p.panic_value < 0) p.panic_value = 0;

        p.ideology_value += convertedAmount;
      }

      if (p.ideology_value > 1.0) p.ideology_value = 1.0;
      else if (p.ideology_value < 0) p.ideology_value = 0;
    });

    if (silo.traits?.includes('psychoactive_meds')) {
      const itDept = silo.professions?.find(p => p.name === 'IT');
      if (itDept) {
        const targetIdeology = itDept.ideology_value;
        silo.professions?.forEach(p => {
          if (p.name !== 'IT') {
            const diff = targetIdeology - p.ideology_value;
            p.ideology_value += diff * 0.05 * deltaYears;
          }
        });
      }
    }
  }

  private updateResources(silo: Silo, deltaYears: number): void {
    const populationFactor = silo.total_population / 10000.0;
    const isRebelling = silo.rebellion > 0.7;

    silo.resources?.forEach((r) => {
      const consumption = (this.perCapitaConsumption[r.type] || 0) * populationFactor;

      let production = 0;
      const producers = this.resourceProducers[r.type] || [];

      producers.forEach(profName => {
        const prof = silo.professions?.find(p => p.name === profName);
        if (prof) {
          const efficiency = (1.0 - prof.panic_value) * prof.productivity;
          const baseProd = (this.perCapitaConsumption[r.type] || 0) * 1.2 / producers.length;
          production += baseProd * efficiency;
        }
      });

      if (isRebelling) {
        production *= 0.3;
      }

      r.net_balance = production - consumption;
      r.amount += r.net_balance * deltaYears;

      if (r.amount < 0) r.amount = 0;
    });
  }

  private updateSiloMetrics(silo: Silo, deltaYears: number): void {
    silo.countdown -= deltaYears;
    if (silo.countdown < 0) silo.countdown = 0;

    silo.event_trigger += (1.0 - silo.cohesion) * deltaYears * 0.1;

    let avgPanic = 0;
    const profCount = silo.professions?.length || 0;
    if (profCount > 0) {
      silo.professions.forEach((p) => {
        avgPanic += p.panic_value;
      });
      avgPanic /= profCount;
    }

    const threshold = 0.1;
    const stressFactor = (1.0 - silo.legitimacy) * avgPanic;
    if (stressFactor > threshold) {
      silo.rebellion += (stressFactor - threshold) * deltaYears * 0.05;
    } else {
      silo.rebellion -= 0.01 * deltaYears;
    }

    if (silo.rebellion > 1.0) silo.rebellion = 1.0;
    else if (silo.rebellion < 0) silo.rebellion = 0;

    this.updatePopulation(silo, deltaYears);
  }

  private updatePopulation(silo: Silo, deltaYears: number): void {
    let deathRate = 0.001;

    silo.resources?.forEach(r => {
      if (r.amount <= 0) {
        deathRate += 0.05;
      }
    });

    if (silo.rebellion > 0.8) {
      deathRate += (silo.rebellion - 0.8) * 0.2;
    }

    const deaths = silo.total_population * deathRate * deltaYears;
    silo.total_population -= Math.floor(deaths);
    if (silo.total_population < 0) silo.total_population = 0;
  }

  // ============ 配置常量 ============

  // 每万人每年消耗基数
  private readonly perCapitaConsumption: Record<string, number> = {
    Food: 50.0,
    Energy: 100.0,
    Water: 80.0,
    Materials: 20.0,
  };

  // 职业对资源的产出贡献
  private readonly resourceProducers: Record<string, string[]> = {
    Food: ['Agricultural'],
    Energy: ['Mechanical', 'IT'],
    Water: ['Mechanical', 'Supply'],
    Materials: ['Mines', 'Mechanical'],
  };

  // 职业威望加成系数
  private readonly professionFactors: Record<string, number> = {
    Mayor: 0.5,
    Judicial: 0.4,
    IT: 0.3,
    Police: 0.3,
    Mechanical: 0.2,
    Medical: 0.2,
  };

  // 特质威望加成系数
  private readonly traitFactors: Record<string, number> = {
    地堡土著: 0.1,
    一号地堡密使: 0.5,
    煽动者: 0.2,
    守旧派: -0.1,
  };
}

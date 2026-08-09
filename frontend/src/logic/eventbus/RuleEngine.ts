import { GameEvent, GameState, EventContext, createEvent } from './types';
import { EventBus } from './EventBus';
import { ConditionEngine } from './ConditionEngine';
import { ScriptEngine } from './ScriptEngine';

/** 规则效果：数据驱动的描述，运行时由 RuleEngine 解释执行 */
export interface RuleEffect {
  type: 'script' | 'fire_event' | 'schedule_event';
  /** script 类型：注册在 ScriptEngine 中的脚本名 */
  script?: string;
  /** fire_event / schedule_event 类型：要触发的事件模板 */
  event?: GameEvent;
  /** schedule_event 类型：相对当前时刻延后的月数 */
  delayMonths?: number;
}

/**
 * 游戏规则 (文档第五层：规则系统)
 *
 * 数据驱动：`event + trigger + effects`
 *
 * ```yaml
 * event:
 *   id: civil_war
 * trigger:
 *   eventType: X      # 响应的事件类型
 *   condition: ...    # 附加条件 (ConditionEngine)
 * effects:
 *   - type: script
 *     script: add_unrest
 *   - type: fire_event
 *     event: { id: refugee_wave, type: REFUGEE_WAVE }
 * ```
 */
export interface GameRule {
  id: string;
  trigger: {
    /** 监听的源事件类型；缺省/ANY 表示响应所有事件 */
    eventType?: string;
    /** ConditionEngine 中的条件 ID */
    condition?: string;
  };
  effects: RuleEffect[];
}

/**
 * 规则引擎：订阅总线所有事件，逐条匹配规则并执行效果，
 * 效果可触发新事件 (bus.emit) 形成链条。
 */
export class RuleEngine {
  private rules: GameRule[] = [];

  /**
   * 延时效果回调：由 GameEngine 注入，将延时事件送入 Scheduler
   * (event 为待调度的模板事件，delayMonths 为延后月数，source 为源事件)
   */
  public onSchedule?: (event: GameEvent, delayMonths: number, source: GameEvent) => void;

  constructor(
    private conditionEngine: ConditionEngine,
    private scriptEngine: ScriptEngine,
    private bus: EventBus
  ) {
    this.bus.subscribeAny((event, ctx) => this.onEvent(event, ctx));
  }

  public register(rule: GameRule): void {
    this.rules.push(rule);
  }

  public registerMany(rules: GameRule[]): void {
    for (const r of rules) this.register(r);
  }

  public onEvent(event: GameEvent, ctx: EventContext): void {
    const state: GameState = this.resolveState(event);
    if (!state.silo) return; // 无状态上下文 (纯信号事件)，跳过规则
    for (const rule of this.rules) {
      if (rule.id === event.id) continue; // 防止规则自我触发
      const t = rule.trigger;
      if (t.eventType && t.eventType !== event.type && t.eventType !== EventBus.ANY) {
        continue;
      }
      if (t.condition && !this.conditionEngine.check(t.condition, state, event)) {
        continue;
      }
      for (const effect of rule.effects) {
        this.applyEffect(effect, event, state, ctx);
      }
    }
  }

  private applyEffect(
    effect: RuleEffect,
    source: GameEvent,
    state: GameState,
    ctx: EventContext
  ): void {
    switch (effect.type) {
      case 'script': {
        if (effect.script) {
          this.scriptEngine.run(effect.script, source, state, this.bus, ctx);
        }
        break;
      }
      case 'fire_event': {
        if (!effect.event) break;
        const data = { ...state, ...(effect.event.data || {}) };
        const child = createEvent(
          effect.event.id,
          effect.event.type,
          data,
          {
            source: effect.event.source ?? source.source,
            target: effect.event.target ?? source.target,
          }
        );
        this.bus.emit(child, ctx);
        break;
      }
      case 'schedule_event': {
        // 延时效果：注入状态后交给 GameEngine 的 onSchedule，进入 Scheduler 排队
        if (effect.event && effect.delayMonths && this.onSchedule) {
          const data = { ...state, ...(effect.event.data || {}) };
          const child = createEvent(
            effect.event.id,
            effect.event.type,
            data,
            {
              source: effect.event.source ?? source.source,
              target: effect.event.target ?? source.target,
            }
          );
          this.onSchedule(child, effect.delayMonths, source);
        }
        break;
      }
    }
  }

  /** 从事件 data 中恢复 GameState 视图 (无则返回空视图) */
  private resolveState(event: GameEvent): GameState {
    const data = event.data as Record<string, unknown> & {
      silo?: GameState['silo'];
      agent?: GameState['agent'];
      logs?: string[];
    };
    return {
      silo: data.silo ?? this.lastState?.silo ?? (undefined as unknown as GameState['silo']),
      agent: data.agent ?? this.lastState?.agent,
      logs: data.logs ?? this.lastState?.logs,
    };
  }

  /** 兜底状态引用：由 GameEngine 在每次 tick 前刷新 */
  public lastState: GameState | null = null;
}

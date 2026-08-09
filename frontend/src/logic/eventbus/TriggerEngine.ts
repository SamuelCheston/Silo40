import { GameState, EventContext } from './types';
import { EventBus } from './EventBus';

export interface Trigger {
  id: string;
  description?: string;
  /** 触发条件：满足即触发 */
  condition: (state: GameState) => boolean;
  /** 触发效果：可 fire_event / 执行脚本 */
  effect: (bus: EventBus, ctx: EventContext, state: GameState) => void;
}

/**
 * 剧情触发器 (文档第四层)
 *
 * 游戏持续检查 trigger 条件，满足即 fire：
 *
 * ```text
 * 皇帝死亡 → 内战 → 难民潮 → 粮价上涨   (形成链条)
 * ```
 */
export class TriggerEngine {
  private triggers: Trigger[] = [];

  public register(trigger: Trigger): void {
    this.triggers.push(trigger);
  }

  public registerMany(triggers: Trigger[]): void {
    for (const t of triggers) this.register(t);
  }

  /** 每次 Tick 后评估所有触发器 */
  public evaluate(bus: EventBus, ctx: EventContext, state: GameState): void {
    for (const t of this.triggers) {
      if (t.condition(state)) {
        t.effect(bus, ctx, state);
      }
    }
  }
}

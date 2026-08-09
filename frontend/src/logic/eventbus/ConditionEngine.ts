import { GameEvent, GameState } from './types';

export type Condition = (state: GameState, event?: GameEvent) => boolean;

/**
 * 条件引擎 (文档第五/八层)
 *
 * 集中注册、复用命名条件，供 RuleEngine / TriggerEngine 数据驱动引用。
 */
export class ConditionEngine {
  private conditions = new Map<string, Condition>();

  public register(id: string, fn: Condition): void {
    this.conditions.set(id, fn);
  }

  public registerMany(defs: Record<string, Condition>): void {
    for (const [id, fn] of Object.entries(defs)) {
      this.register(id, fn);
    }
  }

  public check(id: string, state: GameState, event?: GameEvent): boolean {
    const fn = this.conditions.get(id);
    return fn ? fn(state, event) : false;
  }

  public has(id: string): boolean {
    return this.conditions.has(id);
  }
}

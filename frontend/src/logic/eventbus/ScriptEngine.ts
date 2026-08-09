import { GameEvent, GameState, EventContext } from './types';
import { EventBus } from './EventBus';

export type Script = (
  event: GameEvent,
  state: GameState,
  bus: EventBus,
  ctx: EventContext
) => void;

/**
 * 脚本引擎 (文档第八层)
 *
 * 把 "效果" 注册为命名脚本，供 RuleEngine 的 effects 按名调用，
 * 实现数据驱动 (90% 配置，1% 代码)。
 */
export class ScriptEngine {
  private scripts = new Map<string, Script>();

  public register(name: string, script: Script): void {
    this.scripts.set(name, script);
  }

  public registerMany(defs: Record<string, Script>): void {
    for (const [name, fn] of Object.entries(defs)) {
      this.register(name, fn);
    }
  }

  public run(
    name: string,
    event: GameEvent,
    state: GameState,
    bus: EventBus,
    ctx: EventContext
  ): void {
    const script = this.scripts.get(name);
    if (script) script(event, state, bus, ctx);
  }

  public has(name: string): boolean {
    return this.scripts.has(name);
  }
}

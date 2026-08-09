import { GameEvent, EventHandler, EventContext } from './types';

/**
 * 事件总线 (文档第二层)
 *
 * 所有事件都进入总线，订阅者按事件类型响应：
 *
 * Player Click / AI / Story / Rule / Scheduler
 *      │
 *      ▼
 *   EventBus
 *      │
 * ┌────┼────┐
 * ▼    ▼    ▼
 * AI  Rule  UI ...
 *
 * 两种发布方式：
 * - publish(): 新因果链 (创建全新 EventContext)
 * - emit():    在既有因果链内继续发布 (防事件风暴生效，用于 A→B→C 链)
 */
export class EventBus {
  /** 通配类型：订阅后响应所有事件 (供 RuleEngine 等使用) */
  public static readonly ANY = '*';

  private handlers = new Map<string, Set<EventHandler>>();

  /** 订阅事件，返回取消订阅函数 */
  public subscribe(type: string, handler: EventHandler): () => void {
    if (!this.handlers.has(type)) {
      this.handlers.set(type, new Set());
    }
    this.handlers.get(type)!.add(handler);
    return () => {
      this.handlers.get(type)?.delete(handler);
    };
  }

  /** 订阅所有事件 */
  public subscribeAny(handler: EventHandler): () => void {
    return this.subscribe(EventBus.ANY, handler);
  }

  /** 发布事件 (创建新因果链) */
  public publish(event: GameEvent): void {
    this.dispatch(event, new EventContext());
  }

  /** 在既有因果链内发布事件 (防风暴生效) */
  public emit(event: GameEvent, ctx: EventContext): void {
    this.dispatch(event, ctx);
  }

  private dispatch(event: GameEvent, ctx: EventContext): void {
    if (!ctx.canFire(event.id)) return;

    ctx.markFired(event.id);
    ctx.depth++;
    try {
      const typeHandlers = this.handlers.get(event.type);
      if (typeHandlers) {
        for (const h of typeHandlers) h(event, ctx);
      }
      const anyHandlers = this.handlers.get(EventBus.ANY);
      if (anyHandlers) {
        for (const h of anyHandlers) h(event, ctx);
      }
    } finally {
      ctx.depth--;
    }
  }
}

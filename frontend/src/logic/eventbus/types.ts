import { Agent, Silo } from '../models';

// ============ 统一游戏时间 (文档第三层：时间调度) ============
export interface GameTime {
  year: number;
  month: number; // 1-12
}

/** 比较两个游戏时间：返回负/零/正 */
export function compareTime(a: GameTime, b: GameTime): number {
  if (a.year !== b.year) return a.year - b.year;
  return a.month - b.month;
}

/** 时间推移 months 个月 */
export function advanceTime(t: GameTime, months: number): GameTime {
  const total = t.year * 12 + (t.month - 1) + months;
  const year = Math.floor(total / 12);
  const month = (total % 12) + 1;
  return { year, month };
}

// ============ 统一事件对象 (文档第一层) ============
/**
 * 游戏内一切行为统一抽象为事件：
 * - 玩家点击按钮 → 事件
 * - AI/NPC 行为 → 事件
 * - 剧情触发 → 事件
 * - 时间到达 → 事件
 * - 玩家间互相影响 → 事件
 */
export interface GameEvent {
  id: string;
  type: string;
  source?: string;
  target?: string;
  /** 设置后为延时事件，由 Scheduler 到期发布 */
  trigger_time?: GameTime;
  data: Record<string, unknown>;
}

// ============ 全局游戏状态视图 ============
/** 事件处理器读取/写入的全局状态 */
export interface GameState {
  silo: Silo;
  agent?: Agent;
  logs?: string[];
}

// ============ 事件上下文 (文档第七层：避免事件风暴) ============
/**
 * 事件因果链上下文：
 * - fired: 本链内已触发的事件 id 集合，防止 A→B→C→A 无限循环
 * - depth: 链深度，超过 maxDepth 直接停止
 */
export class EventContext {
  public readonly fired: Set<string> = new Set();
  public depth = 0;
  public readonly maxDepth = 20;

  public canFire(id: string): boolean {
    return this.depth < this.maxDepth && !this.fired.has(id);
  }

  public markFired(id: string): void {
    this.fired.add(id);
  }
}

// ============ 事件处理函数 ============
export type EventHandler = (event: GameEvent, ctx: EventContext) => void;

// ============ 事件工厂 ============
export function createEvent(
  id: string,
  type: string,
  data: Record<string, unknown> = {},
  extra?: Partial<GameEvent>
): GameEvent {
  return { id, type, data, ...extra };
}

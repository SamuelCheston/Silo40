import { GameEvent, GameTime, compareTime } from './types';
import { EventBus } from './EventBus';

interface ScheduledEntry {
  /** 唯一键：同一事件可被调度多次，用自增计数区分 */
  key: string;
  time: GameTime;
  event: GameEvent;
}

/**
 * 时间调度器 (文档第三层)
 *
 * 区分即时与延时事件：
 * - 即时：bus.publish() 立刻执行
 * - 延时：scheduler.add(now + N, event) 进入 MinHeap
 *
 * 每个 Tick：
 *   while (heap.top.time <= now) { bus.publish(heap.pop()); }
 */
export class Scheduler {
  private heap: ScheduledEntry[] = [];
  private counter = 0;

  /** 注册延时事件：at 时刻到期后自动发布到总线 */
  public schedule(event: GameEvent, at: GameTime): void {
    const entry: ScheduledEntry = {
      key: `${event.id}#${++this.counter}`,
      time: at,
      event: { ...event, trigger_time: at },
    };
    this.heap.push(entry);
    this.bubbleUp(this.heap.length - 1);
  }

  /**
   * 每个游戏 Tick 调用：将所有到期事件发布到总线
   * @returns 本次触发的到期事件列表
   */
  public tick(now: GameTime, bus: EventBus): GameEvent[] {
    const due: GameEvent[] = [];
    while (this.heap.length > 0 && compareTime(this.heap[0].time, now) <= 0) {
      const entry = this.popMin()!;
      bus.publish(entry.event);
      due.push(entry.event);
    }
    return due;
  }

  /** 下一次到期时间 (无则 undefined) */
  public peek(): GameTime | undefined {
    return this.heap.length > 0 ? this.heap[0].time : undefined;
  }

  /** 清除所有延时事件 (新游戏初始化) */
  public clear(): void {
    this.heap = [];
    this.counter = 0;
  }

  public get size(): number {
    return this.heap.length;
  }

  // ============ MinHeap 实现 ============

  private popMin(): ScheduledEntry | undefined {
    if (this.heap.length === 0) return undefined;
    const top = this.heap[0];
    const last = this.heap.pop()!;
    if (this.heap.length > 0) {
      this.heap[0] = last;
      this.siftDown(0);
    }
    return top;
  }

  private bubbleUp(i: number): void {
    while (i > 0) {
      const parent = Math.floor((i - 1) / 2);
      if (this.less(i, parent)) {
        this.swap(i, parent);
        i = parent;
      } else {
        break;
      }
    }
  }

  private siftDown(i: number): void {
    const n = this.heap.length;
    for (;;) {
      const left = 2 * i + 1;
      const right = 2 * i + 2;
      let smallest = i;
      if (left < n && this.less(left, smallest)) smallest = left;
      if (right < n && this.less(right, smallest)) smallest = right;
      if (smallest === i) break;
      this.swap(i, smallest);
      i = smallest;
    }
  }

  private less(a: number, b: number): boolean {
    return compareTime(this.heap[a].time, this.heap[b].time) < 0;
  }

  private swap(a: number, b: number): void {
    const tmp = this.heap[a];
    this.heap[a] = this.heap[b];
    this.heap[b] = tmp;
  }
}

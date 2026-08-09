import { Silo, StoryEvent } from './models';
import { EventBus, EventContext, createEvent } from './eventbus';

/** 剧情随机事件类型 (StoryEvent) */
export const STORY_EVENT_TYPE = 'STORY_EVENT';

/**
 * 剧情随机事件引擎
 *
 * 按文档架构改造：不再直接调用 effects，而是将剧情包装为统一 GameEvent
 * 发布到 EventBus，由 STORY_EVENT 订阅者 (GameEngine) 应用效果。
 */
export class EventEngine {
  private events: StoryEvent[] = [
    {
      id: 'water_leak',
      title: '供水管线泄漏',
      description: '底层机械部报告发生严重水管泄漏，部分楼层供水中断。',
      type: 'TECHNICAL',
      effects: (silo: Silo) => {
        const supplies = silo.resources.find(r => r.type === 'Supplies');
        if (supplies) supplies.amount -= 500;
        silo.cohesion -= 0.05;
      }
    },
    {
      id: 'food_poisoning',
      title: '群体性食物中毒',
      description: '水培区的一批农产品受到污染，引发大规模恐慌。',
      type: 'SOCIAL',
      effects: (silo: Silo) => {
        silo.professions.forEach(p => p.panic_value += 0.1);
        silo.legitimacy -= 0.05;
      }
    },
    {
      id: 'outside_signal',
      title: '接收到外部信号',
      description: 'IT部门截获了一段模糊的无线电信号，似乎来自地表或其他地堡。',
      type: 'EXTERNAL',
      effects: (silo: Silo) => {
        silo.professions.forEach(p => p.ideology_value += 0.05);
        // 这里只是示例，实际信息流动由玩家特工行动控制
      }
    },
    {
      id: 'silo1_destroyed_signal',
      title: '一号地堡失去联系',
      description: '所有与一号地堡的通信协议均已超时，服务器不再响应。',
      type: 'EXTERNAL',
      effects: (silo: Silo) => {
        silo.silo1_destroyed = true;
      }
    }
  ];

  /**
   * 触发随机剧情事件 (在 EventContext 内发布，防事件风暴生效)
   * @returns 被选中的剧情事件，未触发返回 null
   */
  public triggerRandomEvent(silo: Silo, bus: EventBus, ctx: EventContext): StoryEvent | null {
    if (silo.event_trigger < 1.0) return null;

    // 重置触发器
    silo.event_trigger = 0;

    const randomIndex = Math.floor(Math.random() * this.events.length);
    const story = this.events[randomIndex];

    // 发布统一事件，效果由总线订阅者应用
    bus.emit(createEvent(`story_${story.id}@${Date.now()}`, STORY_EVENT_TYPE, { silo, story }), ctx);
    return story;
  }
}

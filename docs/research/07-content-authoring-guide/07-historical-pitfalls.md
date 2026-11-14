# Silo40 内容作者手册 · 历史坑速查

> 拆分自 [Silo40 内容作者手册](../07-content-authoring-guide.md)；目录、章节映射与出处见该索引。

## 12. 历史坑速查

| # | 坑 | 修复/守卫 |
| --- | --- | --- |
| 1 | 事件缺 trigger/脚本 → 校验失败 → **全部内容事件静默不加载** | `bd18be1`；防呆闸拦截 |
| 2 | `events/rules/` 被白名单排除，4 个规则事件成死内容 | `8eec461`；`TestContentGuardNoDeadGroups` 拦截 |
| 3 | `delayed_riot` 无触发条件不过校验（纯调度事件） | `8eec461` 起用 `canTrigger: () => false` 约定（§3） |
| 4 | 对部门思潮做持久设计——每 tick 被 cohort 覆写 | 判定为设计（`2255de4`），见 §8 |
| 5 | 订阅延迟队列派发的事件（无 `story_sink`）→ 空指针崩溃 | `51d312f` 已防护；`grief_release_ritual` 已恢复订阅 `blackout_cascade`，`TestDelayedDispatchFollowupDoesNotCrash` 锁定 |
| 6 | `EventTrigger` 空池 `rng.Intn(0)` | `bd04aac` 空池不触发；内容别加 `event_trigger` delta（§4） |
| 7 | 内容故事不进 `TickResult.Stories`（`engine.Drain` 丢 `step_stories`；step 流丢 `runOneMonth` 返回值且操作卡死） | `fd03bf7` + `326241d`；测试 `TestPassTimeReturnsContentStories` / `TestProcessNextEventSurfacesMonthStories` |
| 8 | 事件越过 Y130 窗口（`silo1_destroyed` → TIME）永不触发 | 时序约束见 §6；探针断言触发时点 |
| 9 | 对派生指标（凝聚力/合法性/部门紧张/阶层裂痕）做机制设计 | 每 tick 重设，见 §4 持久性表 |
| 10 | 叛乱门槛事件的门槛值对结局分布**极端敏感** | 门槛必须落在无人值守叛乱轨迹的双峰之间才有分水岭效果；调参务必跑 sim 对照，别凭手感 |
| 11 | 楼层稳定度消费层在稳定度 1.0 时必须**完全中性**（产出因子=1、死亡不加项），否则无声打乱基线 | `core_systems.js` 用 `0.5+0.5×稳定度` 与 `<0.8 才加死亡` 保证中性；`TestFloorStability*` 锁定 |
| 12 | 少量碎片直发不足以保证无人值守的 INFORMATION | 该结局主路径仍是玩家/NPC `SHARE_INFO`；`TestFragmentBridgeMakesInformationEndingReachable` 只锁结构可达 |


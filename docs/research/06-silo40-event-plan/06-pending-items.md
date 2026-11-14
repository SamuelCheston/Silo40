# Silo40 事件内容清单与实现计划 · 待决项与缺陷处理记录

> 拆分自 [Silo40 事件内容清单与实现计划](../06-silo40-event-plan.md)；目录、章节映射与出处见该索引。

## 6. 待决项与缺陷处理记录

状态标记：【已修复】/【判定：设计如此】/【方案待决】/【留待作者】。

1. 【已修复 `8eec461`】**`events/rules/` 不加载**：`contentEventDirectories` 白名单缺 `rules`，
   4 个规则事件是死内容。修复=白名单加入 `rules`；`delayed_riot` 补 `canTrigger: () => false`
   过校验（纯调度约定）。新增测试：规则组加载 + `panic_storm → delayed_riot` 时延链路。
   **sim 对照（20 局，种子 7–26，IT）**：修复前 TIME 100%、avg pop 4486 → 修复后 TIME 100%、
   avg pop 4480——分布无变化（delayed_riot 的 panic+0.15 在基线已饱和，legitimacy 为派生值）。
2. 【判定：设计如此，代码不动】**部门思潮每 tick 被 cohort 覆写**：git 考据确认这是有意设计——
   `2255de4`（2026-08-11，"refactor(ideology): use cohorts for ideology tracking"）提交说明
   明确："track ideologies on population Cohorts as **single source of truth**"、"auto-sync of
   department ideology aggregates from cohort data"、"Update all profession actions to modify
   cohort ideologies instead of direct silo/profession values"。即 `Profession.Ideologies` 自
   该重构起就是 cohort 的人口加权派生视图，`sync_profession_ideologies_from_cohorts` 每 tick
   覆写是**特性而非 bug**。影响：内容事件的 `profession_ideology_delta_all` 写入派生视图，活不过
   下一 tick（`outside_signal` 的 pro_foreign +0.05 无持久机制作用，仅当次快照可见）；内容事件
   想持久影响思潮，现有唯一杠杆是 `panic_value`（驱动 cohort 漂移）。若未来要内容事件直达
   cohort 思潮，需新增内容效果类型（见第 6 条同类扩展），由作者立项。
3. 【已解决（作者拍板 adjust，`bfcbada`/`18e3b47`）】**REBELLION 无人值守可达**：
   实现=门槛式滚雪球事件 `underground_coordination`（rebellion ≥0.62 且 Y126 起，cd6，
   每次 +0.10）。基础叛乱预算（无人值守终局均值 ~0.78、双峰分布 0.69–0.96）决定越线时机，
   越早越线越能在 Y130 TIME 钟之前滚到 1.0。**实测（20 局种子 7–26）：REBELLION 30% /
   TIME 70%；40 局复测 35%/65%**。调参敏感度实测：门槛 0.65→95%、0.67→5%、0.62→30%
   （暴乱注入 ±0.01 同级别敏感）——门槛位于轨迹双峰之间才有分水岭效果。
   另注意判定优先级 TIME 先于 REBELLION：Y130 同刻叛乱到 1.0 也算 TIME（部分局实测如此）。
4. 【已修复 `bd04aac`】**`EventTrigger` 空池采样**：`TriggerRandomEvent` 在池为空时改为直接不触发
   （预算保留），消除 `rng.Intn(0)` 崩溃。测试：`TestTriggerRandomEventEmptyPoolNoTrigger`。
5. 【已修复 `fd03bf7` + `326241d` + `d70dca6`】**内容事件故事不进 `TickResult.Stories`**：
   根因有两层——(a) `engine.Drain` 只回收 `tickStories`、丢弃事件自带的
   `step_stories`/`step_logs`（service 层 `drainQueuedEvents` 一直在收，引擎侧没收），
   `fd03bf7` 对齐回收；(b) step 流（`BeginPassTime → ProcessNextEvent`）里
   `runOneMonth` 的返回值被整段丢弃、且月度 tick 排空队列后 `Step` 报
   "no pending event"、操作卡死在 `phaseQueued`，`326241d` 改为回传该月故事/日志并推进
   操作状态。呈现端：`App.tsx` 结果行原本只显示 `step.stories[0]`，`d70dca6` 改为按既有
   句式连显全部故事（无新交互）。日志端本就消费（`recordStories` → `eventLogs` → 落库 →
   `GetEventHistory`/`/debug/game/events`），无需改动。sim 分布无变化（纯反馈通道）。
6. 【已实现（作者立项，`9923dce`/`393a385`/`18e3b47`）】**楼层/碎片对内容事件的可达性**：

   **碎片**：内容效果封闭集新增 `profession_fragment_add` / `profession_fragment_remove`
   （`Fragment` 字段；add 的 profession 空=补给碎片最少部门、fragment 空=种子随机抽未知碎片，
   均确定性可复现）。机制层变异本就有同名实现，桥只是接入内容面。
   **可达性实测**：引擎测试 `TestFragmentBridgeMakesInformationEndingReachable` 证明连发
   laggard 补给可把全部门抬到 ≥5 碎片并触发 INFORMATION；**但无人值守 sim 中 INFORMATION
   仍为 0%**——内容侧碎片事件（`network_data_cache` 2 枚 + `archive_salvage_operation`
   约 7 枚）不足以把 9 个部门全部补到 ≥5（缺口约 19 枚），该结局仍以玩家/NPC 的
   SHARE_INFO 为主路径。这是实测现状，不是缺陷。

   **楼层**：消费层已立（`core_systems.js`）——(a) 产出：各部门效率 ×(0.5+0.5×本区平均
   稳定度)，稳定度 1.0 时因子=1 完全中性；(b) 死亡：全堡平均稳定度 <0.8 时每低 1.0 加
   +0.25/年 死亡率（上限 +0.20/年）。内容入口 `floor_stability_delta`（zone 过滤，0–1 截断）。
   衰减来源=互调后的 `water_leak`/`blackout_cascade`（Lower −0.04）与 `lower_deck_collapse`
   （Lower −0.08），修复渠道=`structural_reinforcement`（耗 600 Materials，Lower +0.10）。
   实测基线保持：无衰减时分布逐位不变；接入衰减后 avg pop 4480→3197（见 §7 对照）。
   前端无楼层展示面，按作者指示不新造 UI。
7. 【已修复 `51d312f`】**延迟队列事件缺 `story_sink` 的空指针崩溃**：`emitContentEvent` 对
   nil `stories` 计数解引用加防护（`handleContentEvent` 本就 nil 检查 sinks、故事经
   content_source 路径照常记录）。回归测试按延迟派发原样构造无 sink 事件 + `event_triggered`
   订阅者复现原崩溃路径；真实内容路径由 `TestDelayedDispatchFollowupDoesNotCrash` 锁定。
   `grief_release_ritual` 对 `blackout_cascade` 的订阅已在互调中恢复（`bfcbada`）。


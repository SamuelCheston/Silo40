# Silo40 事件内容清单与实现计划

> 用途：`events/` 新事件的选题、校准与分批实现依据。本篇是内容计划，不是设定考据；
> 设定依据引自本目录 01/02/05 篇，机制依据引自仓库代码与只读探针实测。
>
> 硬约束（来自任务）：只写**第 40 号地堡**自己的故事——借世界观/结构/部门/禁忌/思潮/技术设定，
> 不搬 Silo 18 剧情线（不写原著具体人物命运）；只用现有 `events/` DSL 与封闭效果集，
> 不新增效果/变异/事件类型或管线阶段；不改 `internal/engine/`；与现有 8 个内容事件不重复。

本文档已按主题拆分为多个小文件，一份只讲一件事；本文件只做索引，原有信息一条不丢，只改变了分布方式。拆分文件位于 [06-silo40-event-plan/](./06-silo40-event-plan/) 目录。

## 子文件目录

| 文件 | 内容 |
| --- | --- |
| [01-time-window-and-game-calendar.md](./06-silo40-event-plan/01-time-window-and-game-calendar.md) | §1 任务时间窗、游戏历与 Silo 40 的历史定位 |
| [02-mechanism-baseline.md](./06-silo40-event-plan/02-mechanism-baseline.md) | §2 机制基线（只读探针实测：基线轨迹、持久杠杆、加载面事实） |
| [03-quantitative-calibration.md](./06-silo40-event-plan/03-quantitative-calibration.md) | §3 量化校准（粒度换算、密度预算、延迟档位） |
| [04-event-list.md](./06-silo40-event-plan/04-event-list.md) | §4 事件清单（批次 1–3）与叛乱预算核对 |
| [05-validation-and-regression.md](./06-silo40-event-plan/05-validation-and-regression.md) | §5 验证与回归 |
| [06-pending-items.md](./06-silo40-event-plan/06-pending-items.md) | §6 待决项与缺陷处理记录 |
| [07-global-mutual-tuning.md](./06-silo40-event-plan/07-global-mutual-tuning.md) | §7 全局互调轮（量级标尺、因果网、sim 前后对照） |

## 原章节 → 子文件映射

| 原章节 | 所在文件 |
| --- | --- |
| §1（1.1–1.3） | 01-time-window-and-game-calendar.md |
| §2（2.1–2.3） | 02-mechanism-baseline.md |
| §3（3.1–3.3） | 03-quantitative-calibration.md |
| §4 | 04-event-list.md |
| §5 | 05-validation-and-regression.md |
| §6 | 06-pending-items.md |
| §7（7.1–7.3） | 07-global-mutual-tuning.md |

## Sources

- 机制事实：`internal/engine/content_events.go`（触发/效果/校验）、`internal/engine/mechanics.go`
  （变异应用）、`events/mechanics/core_systems.js`（每 tick 重设逻辑、victory_check）、
  `internal/service/game_service.go`（加载白名单、调度与订阅）、`events/README.md`（DSL 约定）。
- 量化基准：RimWorld/Frostpunk 数值研究（本地归档，未随仓库发布；26 来源，[事实]/[社区反推]/[推断] 分级）。
- 设定依据：本目录 01（结构/禁忌/处置机制）、02（小说纪年与改编侧阶段）、05（桥段映射）篇；
  Silo 40 史实（2212 断联战争）见 01 篇 §6.2【原著明确】。
- 基线数据：本轮只读探针实测（种子 7、IT 特工、无玩家动作、97 个月）。

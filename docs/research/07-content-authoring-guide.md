# Silo40 内容作者手册

> 一页速查：给 Silo40 加事件/改机制内容前先看这里，踩过的坑都在 §12。
> 配套资料：`events/README.md`（DSL 字段）、`references/parity-v7-three-silo.json`（当前确定性基线）。
> 本手册只描述现状与约定，不含待落地的设计。

本文档已按主题拆分为多个小文件，一份只讲一件事；本文件只做索引，原有信息一条不丢，只改变了分布方式。拆分文件位于 [07-content-authoring-guide/](./07-content-authoring-guide/) 目录。

## 子文件目录

| 文件 | 内容 |
| --- | --- |
| [01-quickref-and-loading.md](./07-content-authoring-guide/01-quickref-and-loading.md) | §0 决策速查、§1 目录分组与加载链路（含防呆闸） |
| [02-format-triggers-effects.md](./07-content-authoring-guide/02-format-triggers-effects.md) | §2 事件文件格式、§3 触发器封闭集、§4 内容事件效果封闭集 |
| [03-mechanics-mutations.md](./07-content-authoring-guide/03-mechanics-mutations.md) | §5 机制脚本（`events/mechanics/`）变异封闭集 |
| [04-cooldown-delay-id.md](./07-content-authoring-guide/04-cooldown-delay-id.md) | §6 冷却与延迟、§7 事件 ID 与 `schedule_event` 解析 |
| [05-ideology-and-unreachable.md](./07-content-authoring-guide/05-ideology-and-unreachable.md) | §8 思潮：cohort 是唯一真源、§9 内容事件目前拿不到的系统 |
| [06-engine-boundary-and-validation.md](./07-content-authoring-guide/06-engine-boundary-and-validation.md) | §10 内容改不动、必须动引擎的清单，§11 本地验证手册 |
| [07-historical-pitfalls.md](./07-content-authoring-guide/07-historical-pitfalls.md) | §12 历史坑速查 |

## 原章节 → 子文件映射

| 原章节 | 所在文件 |
| --- | --- |
| §0 + §1 | 01-quickref-and-loading.md |
| §2 + §3 + §4 | 02-format-triggers-effects.md |
| §5 | 03-mechanics-mutations.md |
| §6 + §7 | 04-cooldown-delay-id.md |
| §8 + §9 | 05-ideology-and-unreachable.md |
| §10 + §11 | 06-engine-boundary-and-validation.md |
| §12 | 07-historical-pitfalls.md |

## Sources

- 代码事实：`internal/engine/content_events.go`（触发/效果/校验）、
  `internal/engine/mechanics.go`（变异集）、`internal/engine/story_events.go`、
  `internal/service/game_service.go`（白名单/调度/订阅/step 流）、
  `internal/engine/engine.go`（管线与 Drain）、`events/README.md`（DSL）。
- 当前确定性基线：`references/parity-v7-three-silo.json`。
- 历史决议：`2255de4`（cohort 思潮真源）、`bd18be1`/`8eec461`/`51d312f`/`bd04aac`/
  `fd03bf7`/`326241d`/`d69cdcb`（缺陷修复与防呆闸）。

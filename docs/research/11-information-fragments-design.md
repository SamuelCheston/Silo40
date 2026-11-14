# Silo40 信息碎片：共享视角模板、证据组合与有限 LLM 变体

> 日期：2026-11-14
> 性质：内容与机制设计，不包含代码改动。
> 范围：常规地堡共享的信息碎片；不为每座地堡单写一套，不把 固定时间线场景 正典人物命运改造成随机碎片。
> 目标：让玩家获得碎片时立刻读到有来源、有立场、有后续用途的内容，并使同一内容骨架能在不同局中产生安全的表层变化。

本文档已按主题拆分为多个小文件，一份只讲一件事；本文件只做索引，原有信息一条不丢，只改变了分布方式。拆分文件位于 [11-information-fragments-design/](./11-information-fragments-design/) 目录。

## 子文件目录

| 文件 | 内容 |
| --- | --- |
| [01-conclusion-and-current-state.md](./11-information-fragments-design/01-conclusion-and-current-state.md) | §0 阅读约定与最终结论、§1 现状边界 |
| [02-content-model.md](./11-information-fragments-design/02-content-model.md) | §2 内容模型：Fact / Version / Observation / Belief 与字段设计 |
| [03-shared-template-scale.md](./11-information-fragments-design/03-shared-template-scale.md) | §3 纯预写方案：162 份共享模板的规模、分批与分发 |
| [04-presentation-design.md](./11-information-fragments-design/04-presentation-design.md) | §4 呈现设计：可读档案卡、证据组合、互斥版本、NPC 质询、确信度批注 |
| [05-llm-surface-variants.md](./11-information-fragments-design/05-llm-surface-variants.md) | §5 LLM 方案：受限槽位、质量判据、生成时机与存档、成本、退化路径 |
| [06-mechanism-linkage.md](./11-information-fragments-design/06-mechanism-linkage.md) | §6 与 B/π、部门分层、Covenant/Protocol/Directive、E/Q、确定性的衔接 |
| [07-roadmap-acceptance-non-goals.md](./11-information-fragments-design/07-roadmap-acceptance-non-goals.md) | §7 Roadmap（R1–R6）、§8 验收标准、§9 明确不做 |

## 原章节 → 子文件映射

| 原章节 | 所在文件 |
| --- | --- |
| §0 + §1 | 01-conclusion-and-current-state.md |
| §2（2.1–2.2） | 02-content-model.md |
| §3（3.1–3.4） | 03-shared-template-scale.md |
| §4（4.1–4.6） | 04-presentation-design.md |
| §5（5.1–5.6） | 05-llm-surface-variants.md |
| §6（6.1–6.5） | 06-mechanism-linkage.md |
| §7 + §8 + §9 | 07-roadmap-acceptance-non-goals.md |

## 参考资料

- 当前碎片模型与行动：[models.go](../../internal/model/models.go)、[common_actions.js](../../events/mechanics/common_actions.js)、[core_systems.js](../../events/mechanics/core_systems.js)。
- 当前界面与信息架构：[HeaderStats.tsx](../../frontend/src/components/HeaderStats.tsx)、[09-ui-information-architecture.md](./09-ui-information-architecture.md)。
- 机制母题：项目内机制文档与当前事件实现；直接剧情和设定参考资料不进入公开库。
- 连续确信、版本、曝光与公开证据目标：[10-it-control-credibility.md](./10-it-control-credibility.md)。
- 既有 LLM 边界与存档回放经验：[llm-npc.md](../../references/llm-npc.md)。

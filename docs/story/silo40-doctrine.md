# Silo 40 的两份教义文件（二创设定）

> 用途：定义 Silo 40 在覆灭前留给两类人的内部文件——给流散者的 the Guest（《客人》），给被冻特工的 the Errand（《差事》）。
> 性质：基于 Hugh Howey 小说三部曲（Wool / Shift / Dust）的**二创设定**。40 号拥有自己的对应物属外推；
> 条款与节点属项目原创。判定标准仍是"与原著冲突才是否决"，原著留白处可发挥。
> 边界：只使用原书内容；不使用原书没有的人名；不出现剧集独有的人名、情节、工艺与场景。
> 两份文件不是宗教文本：它们是行政文件，没有仪式、没有称谓、没有传教对象。

本文档已按主题拆分为多个小文件，一份只讲一件事；本文件只做索引，原有信息一条不丢，只改变了分布方式。拆分文件位于 [silo40-doctrine/](./silo40-doctrine/) 目录。

## 子文件目录

| 文件 | 内容 |
| --- | --- |
| [01-annotation-and-naming.md](./silo40-doctrine/01-annotation-and-naming.md) | 三层标注；the Guest / the Errand 的命名规则、候选与关系 |
| [02-structure-and-order-form.md](./silo40-doctrine/02-structure-and-order-form.md) | 借 the Order 的条款形态；两份文件的条款骨架 |
| [03-guest-clauses-and-examples.md](./silo40-doctrine/03-guest-clauses-and-examples.md) | the Guest 14 条条款、唯一例外、示例条文、冲突裁决顺序 |
| [04-guest-edge-cases.md](./silo40-doctrine/04-guest-edge-cases.md) | the Guest 条款失效迹象、后代得知真相、失忆与清档、与持有者的关系 |
| [05-errand-policy-tree.md](./silo40-doctrine/05-errand-policy-tree.md) | the Errand 国策树：5 阶段 / 20 节点 |
| [06-errand-example-clauses.md](./silo40-doctrine/06-errand-example-clauses.md) | the Errand 示例条文 |
| [07-errand-self-sufficiency-and-priority.md](./silo40-doctrine/07-errand-self-sufficiency-and-priority.md) | 死手式自足（只写目标、世界与底线）与目标冲突排序 |
| [08-errand-target-expiry-and-invalidation.md](./silo40-doctrine/08-errand-target-expiry-and-invalidation.md) | 目标地堡不存在时；任务失效条件 |
| [09-game-interface-and-mechanism-gap.md](./silo40-doctrine/09-game-interface-and-mechanism-gap.md) | 与游戏的接口（只描述映射）；机制缺口（只列缺口，不改代码） |
| [10-redlines.md](./silo40-doctrine/10-redlines.md) | 写作边界（红线） |

## 原章节 → 子文件映射

| 原章节 | 所在文件 |
| --- | --- |
| §0 三层标注、§1 命名 | 01-annotation-and-naming.md |
| §2（2.1–2.3） | 02-structure-and-order-form.md |
| §3 + §3.1 + §3.2 | 03-guest-clauses-and-examples.md |
| §3.3–3.6 | 04-guest-edge-cases.md |
| §4.1–4.5 | 05-errand-policy-tree.md |
| §4.6 | 06-errand-example-clauses.md |
| §4.7–4.8 | 07-errand-self-sufficiency-and-priority.md |
| §4.9–4.10 | 08-errand-target-expiry-and-invalidation.md |
| §5 + §6 | 09-game-interface-and-mechanism-gap.md |
| §7 | 10-redlines.md |

## Sources

- the Order 的条款形态与 IT 主管的执行方式：《Shift》Second Shift；Wikipedia "Silo (series)" Shift 段。
- Pact（隐藏终局与 E-Day）：《Shift》；Wikipedia "Silo (series)"。
- 40 号断联、令远程关闭失效、干扰终止信号、被物理摧毁（2212）：《Shift》Second Shift；Astromech "Every Known Silo" §Silo 40。
- 40 号部分幸存者流入其他地堡（流散线的作者层面出口，单源）：Astromech "Every Known Silo" §Silo 40 转述 Howey 表态。
- 40 号前史、小队编制、逐舱唤醒与本篇引用的边界：仓库 `docs/story/silo40-backstory.md`。
- 行动门控与阶段推进的代码事实：`events/action_gates.json`、`internal/service/action_gates.go`（只读引用）。

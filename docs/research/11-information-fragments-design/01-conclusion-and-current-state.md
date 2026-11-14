# Silo40 信息碎片：共享视角模板、证据组合与有限 LLM 变体 · 阅读约定、最终结论与现状边界

> 拆分自 [Silo40 信息碎片：共享视角模板、证据组合与有限 LLM 变体](../11-information-fragments-design.md)；目录、章节映射与出处见该索引。

## 0. 阅读约定与最终结论

- **【现状事实】**：已在当前仓库核实的行为。
- **【设计提案】**：本文新增方案，不声称已经实现。
- **【文案示例】**：用于说明写法的原创内容，不是对外部剧情事实的追加断言。

核心结论：

1. 碎片不是“真相 ID”，而是一次有来源的 **Observation**。底层分为 `Fact`（事实骨架）、`Version`（对事实的解释）和 `Observation`（某观察者经某渠道得到的材料）。拥有材料不等于事实为真，事实确信也不等于某一种归因成立。
2. 共享预写库按 **9 个常规部门 × 3 个信息源层级 × 6 个主题 = 162 份视角模板**估算，不乘地堡数量。当前 28 个 ID 先做可读化改造，再扩到 54 份垂直切片，最后才需要补满 162 份。
3. 视角差完全由预写承担。LLM 只把同一份模板改写成不同局中“相似但不同”的表述，不能生成事实、人名、时间、因果、法律条款、证据强度或碎片关系。
4. LLM 在**每局首次获得某块碎片时生成一次，最终文本随该 Observation 入库存档**；之后查看只读存档，绝不每次重算。失败或质量不达标直接使用预写原文，获取与 gameplay 结算不受影响。
5. 可组合证据、互斥立场、NPC 质询、随确信度增加的验证信息，**全部可以纯预写实现**。必须靠 AI 的只有“玩家自由提问并理解任意措辞”或“自由组合任意文本后推断新事实”；这两条均不建议进入近期范围。
6. 最优先不是上 LLM，而是把当前 28 个 ID 变成“标题 + 正文 + 来源 + 指向”的可读卡片。LLM 改善的是重复开局的新鲜度，不解决第一局“拿到了但不知道是什么”的问题。

## 1. 现状边界

| 【现状事实】 | 设计影响 |
| --- | --- |
| 当前共有 28 个字符串 ID：Mayor/Judicial/IT 各 5 个，Police/Medical/Mechanical/Supply/Mines 各 2 个，Agricultural 1 个，见 [models.go](../../../internal/model/models.go#L776-L787)。 | 第一批应覆盖现有 28 个 ID，不能先造一套玩家看不到的新库。 |
| Agent、Profession、PopulationCohort 都只保存 `KnownFragments []string`，见 [models.go](../../../internal/model/models.go#L57-L77)、[Profession](../../../internal/model/models.go#L348-L370)。 | 当前不足以保存来源、版本、确信度、生成文本和观察去重信息；这些应是增量数据，不把长文本塞回 ID 数组。 |
| `GATHER_INFO` 从目标部门前缀下随机取未知 ID；`SHARE_INFO` 接收 ID 列表，未知 ID 还能以 unexplained knowledge 形式分享，见 [common_actions.js](../../../events/mechanics/common_actions.js#L1-L99)。 | 第一阶段只改呈现，不改获取概率、AP、RNG、分享接受率与伪造语义。 |
| 玩家开局最多保留本职业前 2 个碎片，见 [station.go](../../../internal/service/station.go#L21-L40)。 | 每局实际获得量远小于完整内容库，LLM 成本按实际获得量计算。 |
| 前端只在身份 Tooltip 中列出 ID Badge，见 [HeaderStats.tsx](../../../frontend/src/components/HeaderStats.tsx#L172-L199)。 | 增加获取结果卡与档案详情即可产生最大体验收益，不必先做推理棋盘。 |
| INFORMATION 结局仍要求所有部门各持有至少 5 个不同 ID，见 [core_systems.js](../../../events/mechanics/core_systems.js#L374-L395)。 | 第一阶段保留计数契约；以后 `B/π` 是额外层次，不能悄悄把“知道 ID”改成“完全确信”。 |
| 连续 `B`、版本 `π`、阴谋曝光 `E`、公开证据 `Q` 目前来自 [10-it-control-credibility.md](../10-it-control-credibility.md)，尚不是现有实现。 | 本文规定内容如何供给这些未来机制，不在 UI 预告已能公开举证或计算曝光。 |
| 当前驻地与 上层控制方 已有 operation `Exposure`，见 [models.go](../../../internal/model/models.go#L135-L174)。 | 必须把未来的阴谋曝光命名为 `conspiracy_exposure (E)`，与现有 `operation_exposure` 分开，不能复用同一字段。 |
| 当前没有 NPC 碎片质询或 AI 叙事文本生成；已有 LLM 管线只辅助 NPC 决策，见 [llm-npc.md](../../../references/llm-npc.md)。 | NPC 质询与文本变体都是新能力，不能声称复用现有对话系统。 |
| 上层控制方 当前有特殊 Archives 部门，固定时间线场景 跑独立正典链。 | 162 份是常规驻地共享库；Archives 专属材料与 固定时间线场景 正典叙述另行处理，不借此把常规库乘以堡数。 |


# Silo40 内容作者手册 · 思潮真源与内容事件拿不到的系统

> 拆分自 [Silo40 内容作者手册](../07-content-authoring-guide.md)；目录、章节映射与出处见该索引。

## 8. 思潮：cohort 是唯一真源

**结论**：`Profession.Ideologies` 是 cohort 的人口加权派生视图，每 tick 被
`sync_profession_ideologies_from_cohorts` 覆写（设计如此，`2255de4` 起）。内容事件的
`profession_ideology_delta_all` 活不过下一 tick。

- **反例**：`outside_signal` 的 `pro_foreign +0.05` 无持久机制作用，只有触发当月快照可见。
- 内容事件想持久推动思潮，唯一现有杠杆：`panic_value`（恐慌 >0 时驱动 cohort `pro_foreign`
  漂移 `panic×0.10/年`，恐慌 >0.3 且凝聚力 <0.5 时附加漂移）。
- 读侧注意：`profession_ideology_gte_count` 触发器读的是派生值——它反映的是 cohort 现状，
  可放心用作条件。
- 需要内容直达 cohort 思潮 = 新增内容效果类型 = 引擎改动（§10）。

## 9. 内容事件目前拿不到的系统

| 系统 | 现状 | 后果 |
| --- | --- | --- |
| `safeguard_risk` | 内容 metric 集不含它（机制变异层含） | 内容事件碰不到 Safeguard 死局 |
| 新 flag | 读写只有 `silo1_destroyed` | 无法自建状态链，用 metric 阈值 + `event_triggered` 代替 |
| 楼层按 Function 过滤 | `floor_stability_delta` 只按 Zone 过滤 | 想只打"机械区"而非整个 Lower 需引擎扩展 |
| 碎片分发策略 | `profession_fragment_add` 只有"最缺部门 + 随机未知"一种策略 | 指定部门/指定碎片之外的策略需引擎扩展 |

（碎片与楼层本身已可达，见 §4。）


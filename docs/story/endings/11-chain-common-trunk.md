# 结局事件链 · 共同开局主干

> [返回总索引](../silo40-endings.md) · [机制底座](./10-chain-foundation.md) · [通用前提与共同开局](../good-endings/00-common.md)

十条线共用同一条开局主干。主干全部由**既有门控与既有事件**构成，不新增节点；分岔从 `archive_awakened` 之后开始。

## 1. 主干节点

| 节点 | 机制载体 | 触发/推进条件 | 现状 |
| --- | --- | --- | --- |
| 落脚 | 阶段 `arrival` | 开局即是 | 既有 |
| 首份本地材料 | 任一基础职务行动，材料作为伴随 Observation | 职务主结果成功 | 行动设计 15 口径；`first_signal` 来源迁移见缺口 G10 |
| 第一道裂缝 | 阶段 `first_signal` | 现状消费 `player_action:action_gather_info`；目标口径消费首份本地 Observation 键 | 既有阶段，来源待迁移（G10） |
| 材料离手 | 阶段 `archive_awakened` | 完成一次 Share Info（`player_action:action_share_info`） | 既有 |
| 世界锚点一 | `archive_truth_broadcast`（histories） | Y122M1 | 既有 |
| 世界锚点二 | `neighbor_silo_silence` → `network_data_cache` | Y124M1；缓存节点经 `event_triggered` 跟随 | 既有 |
| 世界锚点三 | `silo1_ultimatum` → `ultimatum_aftermath`（+3 月） | Y126M1 | 既有 |
| 清洗拉锯 | `archive_purge_order` → `purge_backlash`（+4 月） | `history_burden ≥ 0.12` | 既有 |
| 终局时钟 | `silo1_destroyed_signal` | Y130M1，置 `silo1_destroyed` flag | 既有 |

## 2. 主干上的第一次分岔

`archive_awakened` 之后，玩家的第一批选择决定本局进入哪条线的节点群。分岔不靠新机制，靠**完成键**：

- 继续向可核验者分享 → 进入《共证》感染节点群或《公议》规则断点节点群；
- 与工程/供给/医疗建立路线关系 → 进入《越岭》路线调查节点群；
- 控制发布节奏、维持低暴露 → 进入《无人裁决》维持节点群；
- 向 Mayor 圈层喂证 → 进入《人治》节点群；
- 什么都不推进 → 世界锚点照常推进，《深闭》惯性节点开始计数。

同一条主干也可以被失败线拦断：`archive_purge_order` 链是《遗忘》缓慢绞杀路径的既有起点；`silo1_ultimatum` 链是《寂堡》终止征兆的既有起点；玩家行动的痕迹累积是《除名》的既有压力源（victory_check 怀疑封顶）。

## 3. 主干给十条线的共同输入

| 输入 | 来源 | 消费者 |
| --- | --- | --- |
| 历史包袱基线 | `archive_truth_broadcast` +0.08 | 《遗忘》清洗阈值、《深闭》常态化读数 |
| 跨堡受限材料 | `network_data_cache` 两份碎片 | 《共证》感染材料、《无人裁决》外部视野 |
| 终止压力 | `silo1_ultimatum` 链 | 《寂堡》《遗忘》《越岭》窗口压缩 |
| 终局事实 | `silo1_destroyed_signal` | 《无人裁决》资格、《深闭》死后复制分岔 |
| 叛乱预算 | 既有事件合计注入 ≈ +0.49（事件计划核对） | 无人值守终局 rebellion ≈ 0.5 < 1.0：《越岭》与《断层》的高叛乱前提仍需玩家动作推高 |

## 4. 主干纪律

1. 空转月份不推进阶段：主干只认成功行动与明确事件结果。
2. 世界锚点不被玩家改写：`neighbor_silo_silence` 与 `silo1_destroyed_signal` 是权威事实，玩家行动只能改变驻地对这些事实的处置，不能提前或阻止它们。
3. 主干不产生新结局主体：锚点事件的效果全部落在既有持久杠杆上。

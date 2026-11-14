# 结局事件链 · `INFORMATION`《共证》

> [返回总索引](../silo40-endings.md) · [机制底座](./10-chain-foundation.md) · [剧情台阶](../good-endings/01-information.md)

链条形态：**感染 → 复制 → 外溢 → 外部独立复制**。宿主必须活到外溢完成。

## 1. 节点表

| 节点 | 分组 | 触发 | 分岔与抉择 | 效果（封闭集） | 后续 |
| --- | --- | --- | --- | --- | --- |
| 阶段主干 | 门控 | 见[共同主干](./11-chain-common-trunk.md) | — | — | `archive_awakened` 后进入本线 |
| `first_receiver_verifies` | special，ONCE | `event_triggered("player_action","action_share_info")` | 抉择 I-B 的机制投影：分享对象由 Share Info 的目标参数承担 | `professionFragmentAdd()`（接收者留下第一份本地副本） | 感染成立 |
| `replication_progress` | special，REPEATABLE cd3 | `event_triggered("player_action","action_share_info")` | `apply` 按 `trigger_count` 分支（第 N 次质变范式）：偶数次为一次成功复制 | 每次 `professionFragmentAdd()`、`history_burden +0.02`（sim 校准起点） | 复制计数 |
| `host_concealment_pressure` | special，ONCE | `all(timestampAtOrAfter(125,1), siloMetricGte("rebellion", 0.4))` | 高叛乱开始威胁宿主隐匿；玩家抉择 I-D（公开化换速度与否）在此结算 | `panic_value +0.04` 全部门 | 提示暴露窗口收窄 |
| `spillover_attempt` | player_actions，ONCE | 门控解锁：`stage_at_least archive_awakened` + `metric known_fragments min 6` + `event_completed replication_progress`（达到复制里程碑） | 抉择 I-F：受限渠道交接（本节点）对立于广泛广播（未建模，按暴露后果结算） | 完成键 `player_action:spillover_attempt` | **缺口 G7**：外部接收与独立复制的结算需要跨堡出站投递 |
| 宿主暴露与处置 | 机制层 | 暴露输入（`E`、operation Exposure）抬升介入风险 | 玩家不能主动献祭宿主 | `safeguard_risk` 属机制层，内容不可写（**缺口 G6**） | 处置成立时走 victory_check `DEATH` 硬护栏 |

## 2. 分支与改道

| 完成键／事实 | 去向 |
| --- | --- |
| `replication_progress` 长期停滞且玩家转向制度内喂证 | 转《公议》节点群（`charter_*`） |
| 宿主在外部复制确认前被处置 | 按实际结果进入《遗忘》（`reset_completed` 键）或《寂堡》 |
| 核心节点随撤离离开 | 转《越岭》节点群；外部复制未独立时《共证》不成立 |
| 宿主活到 `silo1_destroyed_signal` 且开放快照合格 | 与《无人裁决》共同进入资格检查（快照结算见缺口 G4） |

## 3. 超时与失败结算

- 复制链没有全局超时；压力来自世界锚点（清洗链、通牒链）与宿主暴露。
- 宿主暴露后不存在"读档式"救援节点：处置一旦成立，按硬护栏顺序结算，不被信息路线高分覆写。

## 4. 与结局判定的对接

- 现状：`victory_check` 的 INFORMATION 占位条件（全部门持有碎片）只是结构桥；主路径仍是玩家/NPC 分享（作者手册 §12.12）。本线节点以碎片分发与完成键把"传播链"投影为可读状态。
- 目标资格（活链外溢＋外部独立复制）中，**外部一半无法在现有机制内结算**（G7）；本地一半由 `first_receiver_verifies` / `replication_progress` 与碎片分布承担。
- 候选本身已在仲裁池（`INFORMATION`），无 G1 类缺口。

## 5. 能力缺口

G6（内容→介入风险桥接）、G7（跨堡出站与外部复制结算）、G8（惨胜尾声展示）。缺口未补齐前，本线事件链落地到 `spillover_attempt` 为止，其后节点以注释形式预留，不写成看似可运行的内容。

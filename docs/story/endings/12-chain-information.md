# 结局事件链 · `INFORMATION`《共证》

> [返回总索引](../silo40-endings.md) · [机制底座](./10-chain-foundation.md) · [剧情台阶](../good-endings/01-information.md)

链条形态：**感染 → 复制 → 本地活链成立，宿主活到结算**。【裁决更新】**外溢不进胜负判定**：外溢为零同样可以判胜，外溢数量只影响结算分数；外溢准备与投递已落地（Layer 1/3），相关内容已以 `escape_spillover_*` 落地（见 37 分册）。

## 1. 节点表

| 节点 | 分组 | 触发 | 分岔与抉择 | 效果（封闭集） | 后续 |
| --- | --- | --- | --- | --- | --- |
| 阶段主干 | 门控 | 见[共同主干](./11-chain-common-trunk.md) | — | — | `archive_awakened` 后进入本线 |
| `first_receiver_verifies` | special，ONCE | `event_triggered("player_action","action_share_info")` | 抉择 I-B 的机制投影：分享对象由 Share Info 的目标参数承担 | `professionFragmentAdd()`（接收者留下第一份本地副本） | 感染成立 |
| `replication_progress` | special，REPEATABLE cd3 | `event_triggered("player_action","action_share_info")` | `apply` 按 `trigger_count` 分支（第 N 次质变范式）：偶数次为一次成功复制 | 每次 `professionFragmentAdd()`、`history_burden +0.02`（sim 校准起点） | 复制计数 |
| `host_concealment_pressure` | special，ONCE | `all(timestampAtOrAfter(3,1), siloMetricGte("rebellion", 0.4))`（相对 Y3 起） | 高叛乱开始威胁宿主隐匿；玩家抉择 I-D（公开化换速度与否）在此结算 | `panic_value +0.04` 全部门 | 提示暴露窗口收窄 |
| `public_fact_cross_dept` | special，ONCE | `canTrigger` 读 `ctx.states["special:replication_progress"].trigger_count` 达里程碑（配置次数） | 公共事实跨过部门边界的叙事时刻；资格本身由仲裁读碎片覆盖（见 §4） | `history_burden +0.03`（sim 校准起点） | 资格叙事投影 |
| 外溢簇（已落地） | 玩家动作 | `escape_spillover_*` 信道／包裹／投递动作（见 37 分册） | 受限渠道、外部接收与独立复制结算；只进评分 | 出站信号日志 | 见 §5 |
| 宿主暴露与处置 | 机制层 | 暴露输入（`E`、operation Exposure）抬升介入风险 | 玩家不能主动献祭宿主 | `safeguard_risk` 已可由内容效果写入（机制底座） | 处置成立时走 `DEATH` 硬护栏 |

## 2. 分支与改道

| 完成键／事实 | 去向 |
| --- | --- |
| `replication_progress` 长期停滞且玩家转向制度内喂证 | 转《公议》节点群（`charter_*`） |
| 宿主在结算前被处置 | 按实际结果进入《遗忘》（`reset_completed` 键）或《寂堡》；活链未立，资格不成立 |
| 核心节点随撤离离开 | 转《越岭》节点群；本地活链被掏空，《共证》资格随之流失 |
| 宿主活到覆灭结算且开放快照合格 | 与《无人裁决》共同进入资格检查，仲裁只选一个主结局 |

## 3. 超时与失败结算

- 复制链没有全局超时；压力来自世界锚点（清洗链、通牒链）与宿主暴露。
- 宿主暴露后不存在"读档式"救援节点：处置一旦成立，按硬护栏顺序结算，不被信息路线高分覆写。

## 4. 与结局判定的对接

- 仲裁资格（已实现）：`INFORMATION` 的 P1 资格以全部门碎片覆盖为结构桥（作者手册 §12.12：主路径仍是玩家/NPC 分享）；本线节点以碎片分发与完成键把"传播链"投影为可读状态。
- **外溢是评分项（已实现）**：仲裁按出站信号到达的不同地堡数为《共证》加权（`PlayerPerSpilloverSilo`），不进资格。外溢为零不影响判胜。
- 候选已在仲裁池，同分顺序 `INFORMATION` 居首。

## 5. 能力缺口

- 外溢准备与投递已落地（Layer 1/3）：受限渠道、外部接收与独立复制结算见 37 分册（`spillover.go`）；内容已以 `escape_spillover_*` 动作落地。实体交接×逃出群体耦合与 Attention 承载未实装。
- G8（尾声展示：外溢评分对应的尾声画面）。原 G6（内容→`safeguard_risk`）已随机制底座落地。

# 结局事件链 · `AGENT_COMPROMISED`《除名》

> [返回总索引](../silo40-endings.md) · [机制底座](./10-chain-foundation.md) · [剧情台阶](./06-agent-compromised.md)

链条形态：**痕迹累积（计数）→ 封顶 → 抹除演出**。怀疑度是 actor 状态，内容读不到；本线用**秘密行动的完成键计数**作为痕迹累积的可读代理。

## 1. 节点表

| 节点 | 分组 | 触发 | 分岔与含义 | 效果（封闭集） | 后续 |
| --- | --- | --- | --- | --- | --- |
| `identity_trace_accumulates` | special，REPEATABLE cd2 | `any(event_triggered("player_action","station_infiltrate"), event_triggered("player_action","station_deceive"), event_triggered("player_action","silo1_deceive"))`（可追加 `escape_spillover_send_*` 投递完成键） | `apply` 按 `trigger_count` 分支（第 N 次质变范式）：第 3 次=本地执法级叙事，第 5 次=本地 IT 级叙事 | 每次 `panic_value +0.02`（sim 校准起点） | 暴露层级递进 |
| `identity_query_local` | special，ONCE | `script.canTrigger` 读 `ctx.runtime`：`identity_trace_accumulates` 的 `trigger_count ≥ 3`（经 `ctx.states` 读取） | 抉择 E-A 的机制投影：来历被本地执法复核 | `rebellion +0.02` | 烧关系或放慢 |
| `archive_review_opened` | special，ONCE | `ctx.states` 中 `identity_trace_accumulates` 的 `trigger_count ≥ 5` | 本地 IT 开始复查档案：身份线与证据线同时燃烧；抉择 E-B：最后一击还是保护性拆解 | `history_burden +0.03` | 封顶前置 |
| 封顶结算 | 机制层 | `victory_check`：`suspicion_level ≥ 1.0` | 既有硬护栏，内容不介入判定 | `AGENT_COMPROMISED` | 抹除演出 |
| `agent_erasure_staged` | special，ONCE | `canTrigger` 读 `ctx.actor.suspicion_level` 逼近封顶（机制底座已暴露 actor 状态） | 演出约束：抹除以本地名义包装（来历存疑、刑事罪名、无声失踪），不得公开审讯"他堡特工" | `history_burden +0.02` | 尾声 |

## 2. 分支与改道

| 事实 | 去向 |
| --- | --- |
| 封顶前终局事实已成立 | 按该终局判定，《除名》不成立：痕迹累积只是过程压力 |
| 保护性拆解（玩家主动停止） | 驻地仍可能走向《公议》《人治》甚至《共证》——但主结局已是《除名》，那些只是尾声（结局体系 §8） |
| 驻地同时被重置 | `reset_completed` 先于《除名》（P0.2 先于 P0.4）：社会已被格式化，玩家任务无从谈起 |
| 驻地同时维存崩溃 | `fracture_confirmed`／《寂堡》先于《除名》 |

## 3. 超时与失败结算

- 本线无超时：怀疑只升不降是设计口径（身份安全与任务进度方向相反）；玩家放慢只能延缓封顶，不能回退痕迹。
- `identity_trace_accumulates` 的 cd2 防止同一月多次秘密行动重复计数，不改变累积方向。

## 4. 与结局判定的对接

- `AGENT_COMPROMISED` 已是既有硬失败（`victory_check` 怀疑封顶），无 G1 缺口。
- 内容层承担的是**过程叙事**：三级暴露的台阶、抉择时刻、抹除演出。判定本身不经过内容事件。
- 连点级暴露（Silo 1 连上身份预置这个点）正常玩法不可达，不进事件链（剧情分册 §2 已留白）。

## 5. 能力缺口

G8（两个终局变体的展示）。原 G3（内容读不到怀疑度）已随机制底座落地：`ctx.actor` 暴露怀疑/威望/宣传，封顶前演出节点可以直接读。

# 《共证》事件链 · 字段级事件规格

> [返回总索引](../silo40-endings.md) · [节点设计](./23-information-nodes.md) · [作者手册](../../research/07-content-authoring-guide.md)
> 性质：**实装规格**。只写规格，不动 `events/` 下任何实际文件与引擎代码。目标读者是后续实装者：照着填即可，不必回读设计稿。
> 纪律：数值一律 `TODO(校准)`；依赖未落地机制的节点只留注释，不写跑不起来的版本；人名用原书口径。

## 0. 通用约定

- **目录与命名**：条件/里程碑节点入 `events/special/`，升级与执行入 `events/crisis/`，接触卡与玩家动作入 `events/player_actions/`（管线就绪、目录待建）。文件名＝事件 `id`，snake_case。
- **完成键**：事件节点为 `special:<id>`／`crisis:<id>`；玩家动作为 `player_action:<action_id>`（`player_action_complete` 效果或动作自身完成写入）。一个键只记事实，不预写路线归属。
- **fire_mode**：里程碑 ONCE；计数 REPEATABLE＋`cooldown_months`。
- **守卫**：纯调度事件写 `canTrigger: () => false`；守卫逻辑放 `apply` 内读 `ctx.states`；需要"失败事实"时由守卫未过的一方 `emit` 独立事实事件，订阅者只订阅事实事件。
- **作用域**：效果 `scope` 缺省为 self；跨堡投递用 `cross_signal`（`schedule_event` 不接受跨堡 scope）。
- **现有 DSL 盘点**（实装时以此为准）：触发器含 `timestamp_at_or_after / silo_metric_gte|lte / silo_resource_gte|lte / silo_population_gte|lte / silo_flag_true|false / world_flag_true|false / signal_received / event_triggered / player_action_is / profession_metric_gte_count / profession_ideology_gte_count / all / any / canTrigger`；效果含既有持久杠杆、`profession_fragment_add|remove`、`schedule_event`、`cross_signal`、`world_flag_set`、`cohort_ideology_delta`、`authority_evidence_delta / authority_public_fact / authority_pact_qualify`、`escape_group_depart|status`、`player_action_complete`、`belief_observe`、`public_evidence_register`。
- **ctx 可读**：`public_evidence`（已登记证据 ID 表）、`actor.beliefs`、`authority.public_evidence_q`、silo 指标、派系、事件状态。**不可读**：部门信念层、暴露高水位 E、登记 coverage 数值（只有 ID 表）。

## 1. 节点规格

### I-3　first_receiver_verifies（感染成就）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/special/first_receiver_verifies.js`（首选版）；回退版为 `events/player_actions/receiver_verify_accompany.js` |
| 触发条件 | 首选：`all(event_triggered("player_action","action_share_info"), <接收部门信念越过核验阈值>)`——**信念阈值无现有触发器**，首选版等机制（§3）。回退（可写）：玩家动作"陪同回查"，门控 `event_completed` 首次分享＋关系阈值；动作完成即写入 |
| 完成键 | 首选 `special:first_receiver_verifies`；回退 `player_action:receiver_verify_accompany` |
| DSL 能力 | 回退版全部现有：`player_actions` 文件（`scope: "common"`、`target_type: "DEPT"`、`unavailable_behavior: "hide"`、`action_type` 必填）＋门控 `event_completed`／`relation` |
| 依赖 | 首选版依赖"部门信念层可被内容触发器读取"（当前缺失）；回退版无依赖 |
| 反馈 | 接收者指出一处错误或补一条旁证（文案层） |
| 树成就 | 感染（同时推进主干 T2） |

### R-1　replication_progress（复制计数）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/special/replication_progress.js` |
| 触发条件 | `event_triggered("player_action","action_share_info")`；`fire_mode: "REPEATABLE"`，`cooldown_months: 3` |
| 效果 | `profession_fragment_add()`；`silo_metric_delta("history_burden", +0.02)`（`TODO(校准)`）；`apply` 按 `trigger_count` 分支（偶数次＝一次成功复制，质变档） |
| 完成键 | `special:replication_progress`（计数键；质变档由 `trigger_count` 表达，不另立键） |
| DSL 能力 | 全部现有 |
| 依赖 | 无 |
| 树成就 | 复制（与隐匿保持并列为多成就要件） |

### R-2　登记跳传（公共证据登记）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | 不单独建文件：作为 `replication_progress.js` 的附加效果 |
| 触发条件 | 随 R-1 |
| 效果 | `public_evidence_register(fragment, provenance, coverage)`：`fragment` 为该事实键，`provenance`／`coverage` 为 `TODO(校准)` 正数；登记按事实去重，重复只抬字段 |
| 完成键 | 无新键；登记状态读 `ctx.public_evidence` |
| DSL 能力 | 现有（`public_evidence_register` 已实现） |
| 依赖 | 无 |

### L-1　independent_receivers_met（独立接收者达标）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/special/independent_receivers_met.js` |
| 触发条件 | `canTrigger`：`ctx.public_evidence.length >= TODO(校准)` 且来源类型计数 ≥2（按登记 fragment 前缀分类）。**注意**：这是"登记覆盖"的代理口径；定义层三条（独立说明来源／无授意保存／不认权威）不可机械校验，属叙事纪律，实装时在文案与证人选择上体现 |
| 完成键 | `special:independent_receivers_met` |
| DSL 能力 | 现有（`canTrigger` 读 `ctx.public_evidence`） |
| 依赖 | 无（代理口径）；若未来要读真实"接收者状态"，等接收者模型 |

### L-2　living_chain_confirmed（活链终局成就）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/special/living_chain_confirmed.js` |
| 触发条件 | `all(event_triggered("special","independent_receivers_met"), <replication_progress 质变档>, <隐匿保持>)`。质变档：`canTrigger` 读 `ctx.states["special:replication_progress"].trigger_count >= TODO(校准)`。**隐匿保持**：现用代理 `silo_metric_lte("rebellion", 0.8)`（不喂大侦察/整肃）＋`!ctx.states["crisis:silo1_purge_execution"].triggered`；E 阈值版等机制（§3 C-1） |
| 完成键 | `special:living_chain_confirmed` |
| DSL 能力 | 全部现有 |
| 依赖 | 代理版无依赖；E 版等 C-1 |
| 熄灭语义 | 活链可因维持失败熄灭：当前 DSL 无"撤销完成键"能力——**缺口**：熄灭只能表达为"后续节点不再读它"＋L-3 维持条件失效；若要点亮后再熄灭，需机制批次提供状态可逆能力（§3） |

### L-3　再流通（维持公共事实，并行）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/rules/info_recirculation_pressure.js`（常驻规则）＋`events/special/info_fact_fading.js`（淡忘预警） |
| 触发条件 | 需要读"覆盖跌破淡忘地板"：依赖**碎片淡忘机制**（已分享碎片随时间衰减、流通抵衰减，机制批次实现中）。当前 DSL 无衰减触发器、无可读覆盖数值 |
| 效果 | 预警节点：`panic_value` 微增或纯文案；再流通手段＝玩家再 Share Info／`public_evidence_register` 抬字段 |
| 完成键 | 无常驻键；`info_fact_fading` 为可重复预警键 |
| DSL 能力 | **不足**：缺衰减速率、淡忘地板、覆盖读数 |
| 依赖 | **必须等**：碎片淡忘机制（§3） |

### P-1　public_fact_cross_dept（公共事实投影）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/special/public_fact_cross_dept.js` |
| 触发条件 | `canTrigger`：`ctx.public_evidence.length >= TODO(校准)`（跨部门里程碑档，高于 L-1 档）且 `ctx.authority.public_evidence_q > 0` |
| 效果 | `silo_metric_delta("history_burden", +0.03)`（`TODO(校准)`） |
| 完成键 | `special:public_fact_cross_dept` |
| DSL 能力 | 全部现有 |
| 依赖 | 无（《公议》资格供给若引用本节点，走 `authority_evidence_delta`，另行接 charter 链） |

### C-1　concealment_maintained（暴露管理，常驻条件）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | 不建事件文件——这是常驻条件，不是节点。实装为 L-2／后续节点触发器中的一个条件项 |
| 触发条件 | 目标口径：E < 注意力阈值 `TODO(校准)`。**E 不可被内容读取**（ctx 无 E、无部门信念层）——缺口 |
| 现行代理 | `silo_metric_lte("rebellion", 0.8)` ＋ 未触发整肃/侦察键（`!ctx.states["special:strike_reconnaissance"].triggered`） |
| 依赖 | **必须等**：E／部门信念层的内容可读性（§3） |

### C-2　反情报动作

| 字段 | 规格 |
| --- | --- |
| 事件文件 | 无新文件：沿用既有 `station_deceive`／`silo1_deceive` 动作 |
| 备注 | 暴露增量按既有 Exposure 轨道；高水位不可逆由 E 语义保证，无需新节点 |

### S-1　外溢簇（预留评分）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | 预留：`events/player_actions/spillover_channel_it_line.js`／`..._cross_signal.js`／`..._physical_handoff.js`（三种信道接触卡）＋`events/player_actions/spillover_package_<能力>.js`×5 |
| 触发条件 | 接触卡门控：`stage_at_least archive_awakened`＋`event_completed living_chain_confirmed`＋对应关系 |
| 效果 | 需要 `spillover_channel_add`（16 分册 §6.2 新增效果，**未实现**）与 `preparation_delta`（账本，**未实现**）；投递本身可用现有 `cross_signal(event_id, delay_months, scope)`，但接收结算属外溢模型主体 |
| 完成键 | `player_action:spillover_channel_*`、`player_action:spillover_package_*` |
| 依赖 | **必须等**：16 分册 Layer 1（账本）与 Layer 3（投递/独立接收）；此前只留注释 |
| 胜负 | 明确不进资格；仲裁只按出站信号到达数计 Information 分 |

## 2. 清单1：可直接写的节点

不依赖新机制，机制批次一到即可进仓库（实装时按本规格填值）：

| 节点 | 文件 | 备注 |
| --- | --- | --- |
| R-1 replication_progress | `events/special/replication_progress.js` | 全部现有能力 |
| R-2 登记跳传 | 并入 R-1 | `public_evidence_register` 已实现 |
| I-3 回退版 receiver_verify_accompany | `events/player_actions/receiver_verify_accompany.js` | 玩家动作管线就绪 |
| L-1 independent_receivers_met | `events/special/independent_receivers_met.js` | 代理口径（登记计数） |
| L-2 living_chain_confirmed | `events/special/living_chain_confirmed.js` | 隐匿用代理条件 |
| P-1 public_fact_cross_dept | `events/special/public_fact_cross_dept.js` | 全部现有能力 |

## 3. 清单2：必须等机制的节点

| 节点 | 等的机制 | 未到位前的处理 |
| --- | --- | --- |
| I-3 首选版（接收者自发核验） | 部门信念层可被内容触发器读取（或机制层在核验发生时发事件） | 用回退版（陪同回查动作） |
| L-3 再流通／淡忘预警 | **碎片淡忘机制**（衰减速率、淡忘地板、覆盖读数；机制批次实现中） | 只留注释；禁止硬写一个读不到衰减的版本 |
| C-1 暴露管理（E 口径） | E／部门信念层的内容可读性（ctx 扩展或机制桥接） | 以叛乱阈值＋整肃/侦察键代理 |
| L-2 的"熄灭"语义 | 完成键可逆／状态撤销能力 | 熄灭表达为后续节点不再读取＋L-3 失效 |
| S-1 外溢簇 | 16 分册 Layer 1 账本（`spillover_channel_add`、`preparation_delta`）＋Layer 3 投递与独立接收结算 | 只留注释；接触卡骨架也不提前写，避免门控引用不存在的账本 |

## 4. 缺口汇总（给机制批次）

1. 部门信念层／核验状态的内容触发器（I-3 首选）。
2. E（归因加权信念高水位）与部门信念的 ctx 可读性（C-1、L-2 精确版）。
3. 碎片淡忘：衰减速率、淡忘地板、覆盖读数触发器（L-3）。
4. 完成键可逆性（L-2 熄灭）。
5. 外溢账本效果 `spillover_channel_add`／`preparation_delta`（S-1，16 分册 §6.2 已列）。

# 《人治》事件链 · 字段级事件规格

> [返回总索引](../silo40-endings.md) · [节点设计](./30-personal-rule-nodes.md) · [作者手册](../../research/07-content-authoring-guide.md)
> 性质：**实装规格**。沿用 [27《公议》事件规格](./27-rule-of-law-event-spec.md) 的格式与纪律：只写规格，数值一律 `TODO(balance)`；依赖未落地机制的节点只留注释；功能位、无人名。

## 0. 通用约定

- **目录与命名**：条件/里程碑节点入 `events/special/`，接触卡与选择入 `events/player_actions/`。文件名＝事件 `id`，snake_case，本线前缀 `legitimacy_*`／`mayor_*`／`police_*`／`succession_*`。
- **完成键**：事件节点为 `special:<id>`；玩家动作为 `player_actions:<事件 id>`（内容完成键按 `组:事件 ID` 命名；门控 `event_completed` 引用须带组前缀）。一个键只记事实。
- **布线事实**（与 27 号规格相同）：完成键写入即发总线事件；带 `event_triggered` 的定义参与月度持续评估；订阅既有动作（Share Info）的节点必须放 `special/`，不得放 `player_actions/`（否则宿主动作结算被吞）。
- **机制事实（本线特有）**：`legitimacy` 是派生指标——`metrics_update` 每 tick 重设为派系加权凝聚力，内容 delta 活不过下一 tick；但权威推进在重设前读取正统性，警察服从按引擎响应率（0.35，TODO(balance)）每月向其收敛一次。因此本线的正统性 delta 是**服从响应的月度杠杆**，不是持久写入。资格读取（`personalRuleEligible`：现任 Mayor、W ≤ 30、正统性 ≥ 0.35、服从 ≥ 0.60、Regime ≠ IT）全在引擎侧，内容不写。
- **ctx 可读**：`authority`（`w / police_obedience / regime / pact_locked`）、`silo_metrics.legitimacy`（即 `MayorLegitimacy()`）、`states`、silo 指标。
- **门控**：条件节点支持 `event_completed / stage_at_least / metric / relation` 与 `all/any` 递归；无否定谓词（见 M-2 缺口）。

## 1. 节点规格

### M-1 legitimacy_survey（正统性调查接触卡）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/player_actions/legitimacy_survey.js` |
| 动作规格 | `id: "LEGITIMACY_SURVEY"`、`action_type: "PLAYER_EVENT"`、`scope: "common"`、`target_type: "NONE"`、`unavailable_behavior: "hide"`；`ap_cost: 1`、`duration_months: 0`（`TODO(balance)`） |
| 触发条件 | `playerActionIs("LEGITIMACY_SURVEY")`；`fire_mode: "ONCE"` |
| 效果 | 无机制效果：`script.apply` 返回 `{ effects: [], emit: [] }` |
| 完成键 | `player_actions:legitimacy_survey` |
| 门控 | `unlock: all(stage_at_least "first_signal", relation Mayor min 0.15)`（`TODO(balance)`）；`available: {}` |
| DSL 能力 | 全部现有 |
| 依赖 | 无 |

### M-2 mayor_ally_accept / mayor_ally_bypass（二选一）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/player_actions/mayor_ally_accept.js`／`mayor_ally_bypass.js` |
| 动作规格 | 各为 `PLAYER_EVENT`、`target_type: "NONE"`、`unavailable_behavior: "hide"`；`ap_cost: 0`、`duration_months: 0`（`TODO(balance)`） |
| 触发条件 | 各自 `playerActionIs(<动作 ID>)`；`fire_mode: "ONCE"` |
| 效果 | 均无机制效果（`apply` 返回空）；绕过键注释指依赖：线关闭后局面滑向《深闭》（`normalcy_hardens` 已实现） |
| 完成键 | `player_action:mayor_ally_accept`／`player_action:mayor_ally_bypass` |
| 门控 | 两键同解：`unlock: event_completed "player_actions:legitimacy_survey"` |
| DSL 能力 | 全部现有 |
| 依赖 | **缺口**：门控无否定谓词，选定后兄弟选择不能隐藏；下游只读接受键 |

### F-1 mayor_evidence_step（喂证心跳）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/special/mayor_evidence_step.js`（**必须放 special/**：订阅完成键、不拦截 Share Info 结算） |
| 触发条件 | `all(eventTriggered("player_action","mayor_ally_accept"), eventTriggered("player_action","action_share_info"))`；`fire_mode: "REPEATABLE"`，`cooldown_months: 3`（`TODO(balance)`） |
| 效果 | `siloMetricDelta("legitimacy", +0.02)`（`TODO(balance)`）：派生指标，次月权威推进折算为一次服从响应（§0 机制事实） |
| 完成键 | `special:mayor_evidence_step`（计数键） |
| DSL 能力 | 全部现有 |
| 依赖 | 无。心跳语义与 `charter_draft_step` 同型 |

### S-1 mayor_crisis_stage（危机舞台）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/special/mayor_crisis_stage.js` |
| 触发条件 | `any(eventTriggered("crisis","food_poisoning"), eventTriggered("crisis","water_leak"), eventTriggered("special","energy_rolling_blackout"), siloMetricGte("rebellion", 0.3))`；`fire_mode: "REPEATABLE"`，`cooldown_months: 9`（`TODO(balance)`） |
| 效果 | `professionMetricDeltaAll("panic_value", +0.03)`（`TODO(balance)`） |
| 完成键 | `special:mayor_crisis_stage` |
| DSL 能力 | 全部现有（`any` 混 `event_triggered` 与数值分支已被硬伤 #2 修复保护） |
| 依赖 | 无。危机同时被其他线消费（《断层》《遗忘》），不设互斥闸 |

### S-2 mayor_public_stand（公开担当时刻）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/special/mayor_public_stand.js` |
| 触发条件 | `all(eventTriggered("special","mayor_crisis_stage"), siloMetricGte("legitimacy", 0.5))`；`fire_mode: "REPEATABLE"`，`cooldown_months: 9`（`TODO(balance)`）。【裁决】允许重复 |
| 效果 | `siloMetricDelta("rebellion", -0.05)`；`professionMetricDeltaAll("panic_value", -0.05)`；`siloMetricDelta("legitimacy", +0.05)`（均 `TODO(balance)`） |
| 完成键 | `special:mayor_public_stand`（计数键） |
| DSL 能力 | 全部现有 |
| 依赖 | 无。`legitimacy < 0.5` 时永不点火是本线的失败形态（30 号 §7），不是机制缺口 |

### S-3 police_obedience_shift（警察转向）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/special/police_obedience_shift.js` |
| 触发条件 | `eventTriggered("special","mayor_public_stand")` ＋ `canTrigger`：`ctx.authority.police_obedience >= 0.60`（`TODO(balance)`，镜像引擎 `PersonalRuleMinObedience`）；`fire_mode: "ONCE"` |
| 效果 | 无机制效果（纯里程碑，`apply` 返回空） |
| 完成键 | `special:police_obedience_shift` |
| DSL 能力 | 全部现有（`ctx.authority.police_obedience` 已暴露） |
| 依赖 | 无 |

### H-1 succession_pure_rule / succession_seed_procedure（继承抉择）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/player_actions/succession_pure_rule.js`／`succession_seed_procedure.js` |
| 动作规格 | 各为 `PLAYER_EVENT`、`target_type: "NONE"`、`unavailable_behavior: "hide"`；`ap_cost: 0`、`duration_months: 0`（`TODO(balance)`） |
| 触发条件 | 各自 `playerActionIs(<动作 ID>)`；`fire_mode: "ONCE"` |
| 效果 | 均无机制效果（`apply` 返回空）；种子键注释指依赖：《公议》接触卡为其追加解锁路径 |
| 完成键 | `player_action:succession_pure_rule`／`player_action:succession_seed_procedure` |
| 门控 | 两键同解：`unlock: event_completed "special:mayor_public_stand"` |
| DSL 能力 | 全部现有 |
| 依赖 | 无 |

### 门控改动：charter_contact_offer 追加种子路径

| 字段 | 规格 |
| --- | --- |
| 文件 | `events/action_gates.json` 既有 `CHARTER_CONTACT_OFFER` 条目 |
| 改动 | `unlock` 改为 `all(relation Judicial min 0.2, any(event_completed "special:order_chain_breach", event_completed "player_actions:succession_seed_procedure"))`：规则断点或程序种子任一成立即可接触，关系门槛共用 |
| 依据 | 16 分册 §2"埋程序种子 → 转《公议》节点群"；关系门槛保留（接触方仍是司法侧程序派） |

## 2. 清单1：可直接写的节点

不依赖新机制，全部现有能力：

| 节点 | 文件 | 备注 |
| --- | --- | --- |
| M-1 legitimacy_survey | `events/player_actions/legitimacy_survey.js` | 接触卡管线就绪 |
| M-2 二选一 | `events/player_actions/mayor_ally_{accept,bypass}.js` | 兄弟选择隐藏是已知缺口 |
| F-1 mayor_evidence_step | `events/special/mayor_evidence_step.js` | 心跳语义同 charter_draft_step |
| S-1 mayor_crisis_stage | `events/special/mayor_crisis_stage.js` | 危机事件键均已存在 |
| S-2 mayor_public_stand | `events/special/mayor_public_stand.js` | 全部现有能力 |
| S-3 police_obedience_shift | `events/special/police_obedience_shift.js` | ctx.authority 已暴露 |
| H-1 继承二选一 | `events/player_actions/succession_{pure_rule,seed_procedure}.js` | 种子路径改 charter 门控 |
| 门控条目 | `events/action_gates.json` 新增 5 条＋改 1 条 | 见 §1 |

## 3. 清单2：必须等机制的节点

| 节点/能力 | 等的机制 | 未到位前的处理 |
| --- | --- | --- |
| 资格中 W ≤ 30 的稳定达成 | **W 离散负冲击通道**（并行轨：1 号覆灭／终止链失效／大规模公开证据） | 内容按现有接口读 `ctx.authority.w`，文档标依赖；不自造 W 判定。W ≤ 30 窗比约章窗（≤18）宽，E 自然累积更易达成，但仍不保 |
| Mayor 成为撤离公开领头 | 《越岭》节点群（未编写） | 30 号 §7 留依赖注释；持有者离开后资格自然失效（引擎读现任 holder） |
| M-2 兄弟选择隐藏 | 门控否定谓词 | 两键同解，下游只读接受键 |
| 继承空洞的剧情后果 | 无需等：引擎 `refreshMayorHolder`／`resetJudicialForNewMayor` 已是机制对应物 | 文档说明即可 |

## 4. 缺口汇总（给机制批次）

1. W 离散负冲击通道（并行轨在做）：落地前 W ≤ 30 靠 E 自然累积，达成不保。
2. 门控否定谓词：接触卡二选一的互斥呈现（与《公议》共用缺口）。
3. 派生指标的内容写入语义：正统性 delta 只活一个 tick 的现状是既有机制（派生表），本线按"服从响应月度杠杆"使用；若要持久正统性写入，需机制批次提供非派生载体。

# 《人治》事件链 · 完整节点设计

> [返回总索引](../silo40-endings.md) · [链骨架](./16-chain-personal-rule.md) · [剧情台阶](./02-personal-rule.md)
> 性质：**内容节点设计文档**。沿用 [26《公议》节点设计](./26-rule-of-law-nodes.md) 的格式与纪律：只写设计，数值一律 `TODO(balance)`，依赖未落地机制处只留注释。
> 口径：司法部只在 IT／SWING／约章三态间迁移，**永不归市长**；市长对司法部只有影响力修正（施压、换人、个案松弛），不构成归属（[ADR-0005](../../../knowledge/adr/0005-judicial-not-mayors.md)）。《人治》资格＝警察服从＋市长正统性＋W 退场＋司法部不在 IT 手里；司法部中立或敌意不妨碍成立，约章锁定可并存，同分按既定序让《公议》。玩家是干涉者不是指挥者（[ADR-0010](../../../knowledge/adr/0010-player-as-intervener.md)）。

## 0. 全线总览

```text
共同开局（落脚／裂缝／材料离手）
        │
        ▼
  M-1 正统性调查接触卡（legitimacy_survey）
        │
   ┌────┴────────┐
   ▼             ▼
 部分盟友       绕过            ← 二选一完成键（M-2）
   │             └── 线关闭：材料留在手里，局面滑向《深闭》惯性（normalcy_hardens 已有征兆）
   ▼
  F-1 喂证步骤（mayor_evidence_step，REPEATABLE 心跳）
   │   责任链材料喂给 Mayor 圈层；正统性小步垫高（引擎次月折算为服从响应）
   ▼
  S-1 危机舞台（mayor_crisis_stage，REPEATABLE：危机给市长合法舞台）
   ▼ 正统性 ≥ 0.5
  S-2 公开担当时刻（mayor_public_stand，REPEATABLE：首次是收束前提，其后抵御回摆）
   │   正统性大步垫高 → 警察服从逐月被拉过去
   ▼ 警察服从越过资格线
  S-3 警察转向（police_obedience_shift：法警开始问"这命令合法吗"）
   ▼
  H-1 继承抉择（succession 二选一）
   ├── 纯粹人治：本线收束，等仲裁读资格
   └── 埋程序种子：转《公议》节点群（charter_contact_offer 追加解锁路径）

并行：资格由引擎读事实（警察服从／正统性／W／Regime），内容不写资格
```

顺序：接触（M-1→M-2）→ 喂证（F-1）→ 舞台（S-1）→ 担当（S-2）→ 警察转向（S-3）→ 继承（H-1）。并行：危机舞台可被别的线同时消费（同一危机也是《断层》的燃料、《遗忘》的掩护）；资格判定全程在引擎侧。

## 1. 接触簇：唯一可见的替代权力

### M-1 正统性调查（legitimacy_survey）

| 项 | 内容 |
| --- | --- |
| 形态 | 行动设计 15 §5.3 标准接触卡：`action_gates` 解锁 → 玩家打开接触并完成 offer 动作 → 二选择解锁 |
| 解锁条件 | `stage_at_least first_signal` ＋ 玩家与 Mayor 关系 ≥ 0.15（sim 校准起点）。接触方是 Mayor 身边的联系人；玩家调查的不是权力黑箱，而是**居民实际信任谁**：过去危机中的记录、职位之外是否被执行、是否为驻地承担过个人代价（剧情台阶 P-2） |
| 产出完成键 | `player_actions:legitimacy_survey` |
| 效果 | 无机制效果（纯接触节点，`apply` 返回空） |

### M-2 二选一（mayor_ally_accept / mayor_ally_bypass）

| 项 | 内容 |
| --- | --- |
| 形态 | 两个 `PLAYER_EVENT` 内容行动：部分盟友／绕过，各一键（抉择 P-A） |
| 接受（部分盟友） | 线继续：他不需要知道全部真相也能做对的事；玩家用本地层证据与关系支持他，对他保留跨堡层（保留不是欺骗） |
| 绕过 | 线关闭：他不知道真相所以不可用，玩家转入纯秘密路线。**下落**：驻地从此没有一个能公开承接秩序的本地人，局面按惯性滑向《深闭》（`normalcy_hardens` 征兆照常被点亮）；本局不再重开本线（接触键已消耗） |
| 产出完成键 | `player_actions:mayor_ally_accept`／`player_actions:mayor_ally_bypass` |
| 兄弟隐藏 | 与《公议》接触卡同型：`not`＋`hide_when_unavailable` 已实现并在用，选定后兄弟选择隐藏；下游只读接受键 |

## 2. 喂证簇：材料喂给制度内的另一种人

### F-1 喂证步骤（mayor_evidence_step）

| 项 | 内容 |
| --- | --- |
| 触发条件 | `player_actions:mayor_ally_accept` 与 `player_actions:action_share_info` 两完成键均已触发；`REPEATABLE`，冷却 3 月（sim 校准起点） |
| 心跳语义 | 与《公议》`charter_draft_step`、《共证》`replication_progress` 同型：完成键持久存在，月度持续评估使喂证按冷却节奏推进；不拦截、不替代 Share Info 本身的职务结算（放 `special/` 组，不放 `player_actions/`） |
| 效果 | `siloMetricDelta("legitimacy", +0.02)`（sim 校准起点）：责任链材料垫高市长声望。**机制化身**：正统性是持久状态，delta 跨月累积，并被次月权威推进读取——警察服从按引擎响应率向正统性移动一次。喂证因此是"服从被逐月拉过去"的第一股力（§5） |
| 产出完成键 | `special:mayor_evidence_step`（计数键） |
| 分岔 | 抉择 P-B 的机制投影：本节点只走"交给 Mayor"一支；"留作筹码"一支不设机制——不喂证则本线停在接触步骤，局面滑向《深闭》（下落见 §7） |

## 3. 舞台与担当簇：戏肉在警察

### S-1 危机舞台（mayor_crisis_stage）

| 项 | 内容 |
| --- | --- |
| 触发条件 | `all`：接受键（`player_actions:mayor_ally_accept`）已触发 ＋ `any`：`crisis:food_poisoning`／`crisis:water_leak`／`special:energy_rolling_blackout` 任一已触发，或 `rebellion ≥ 0.3`（sim 校准起点）；`REPEATABLE`，冷却 9 月（sim 校准起点）。舞台属于本线：未接受接触的市长不会自行开场（线是玩家可达的路径，不是后台进程；资格仍是引擎读数） |
| 含义 | 危机给 Mayor 合法舞台（抉择 P-C：预期走向是等舞台而不是推他提前对抗）。机会窗口而非倒计时——同一危机也被《断层》《遗忘》消费，不设互斥闸 |
| 效果 | 全部门 `panic_value +0.03`（sim 校准起点） |
| 产出完成键 | `special:mayor_crisis_stage` |

### S-2 公开担当时刻（mayor_public_stand）

| 项 | 内容 |
| --- | --- |
| 触发条件 | `all`：`special:mayor_crisis_stage` 已触发 ＋ `legitimacy ≥ 0.5`（sim 校准起点）；`REPEATABLE`，冷却 9 月（sim 校准起点）。【裁决】允许重复：首次是收束前提，其后每一次都在抵御秩序回摆（W 回升、危机更替、法警试探） |
| 效果 | `rebellion −0.05`、全部门 `panic_value −0.05`、`siloMetricDelta("legitimacy", +0.05)`（sim 校准起点）。正统性大步垫高 → 引擎次月把服从向它拉近——**个人统治就是一连串担当时刻把警察一个月一个月拉过去的**（§5） |
| 产出完成键 | `special:mayor_public_stand`（计数键） |
| 玩家反馈 | Mayor 以个人权威公开拒绝或改写一道不合法命令；证人在场、部门能确认、Police 愿意执行。他拒绝的是"不合法的本地命令"，不是"来自 Silo 1 的命令"（设定自检：他自始至终不需要知道其他地堡存在） |
| 失败形态 | `legitimacy < 0.5` 时本节点永不点火：线停在喂证步骤，局面按实际权力状态滑向《深闭》（下落见 §7） |

### S-3 警察转向（police_obedience_shift）

| 项 | 内容 |
| --- | --- |
| 触发条件 | `special:mayor_public_stand` 已触发 ＋ `canTrigger` 读 `ctx.authority.police_obedience ≥ 0.60`（sim 校准起点，镜像引擎 `PersonalRuleMinObedience`）；ONCE |
| 含义 | 本线的里程碑读数：警察服从越过资格线的那一刻——法警与警员开始问"这命令合法吗"，开始有选择地执行。这条线的戏肉在警察：司法部从不归市长，警察才是人治秩序的实际承接者 |
| 效果 | 无机制效果（纯里程碑，`apply` 返回空） |
| 产出完成键 | `special:police_obedience_shift` |

## 4. 继承簇：人治的已知弱点

### H-1 继承抉择（succession_pure_rule / succession_seed_procedure）

| 项 | 内容 |
| --- | --- |
| 形态 | 两个 `PLAYER_EVENT` 内容行动，`event_completed special:mayor_public_stand` 后解锁（抉择 P-D） |
| 纯粹人治 | 秩序与此人同寿；本线收束，等仲裁按事实读资格。效果：无（纯完成键） |
| 埋程序种子 | 转《公议》节点群：`charter_contact_offer` 的解锁条件追加种子路径（`all(relation Judicial ≥ 0.2, any(order_chain_breach, 种子键))`）。效果：无机制效果；代价与下落见 §6 |
| 产出完成键 | `player_actions:succession_pure_rule`／`player_actions:succession_seed_procedure` |

## 5. 警察服从与市长正统性：机制化身

- **正统性的真源**：`legitimacy` 是持久状态：离散写入（本线喂证/担当、跨堡行动、谈判层、18 号链）累积，月度叛乱侵蚀（引擎 `runMonthlyPoliticalDispatch`，年率 0.10×叛乱，TODO(balance)）是唯一连续下行项。曾有的"每 tick 重设为派系加权凝聚力"是缺陷，已修复。权威推进每月读取正统性：警察服从按 `PoliceObedienceResponseRate`（引擎 0.35，TODO(balance)）向正统性收敛一次。
- **拉过去**：F-1 每次 +0.02、S-2 每次 +0.05 的正统性 delta 持久累积，各换来次月一次服从响应。担当可重复（cd9），服从因此被一连串担当**逐月**拉向资格线——这就是"戏肉在警察"的机制表达。
- **丢回来**：叛乱高企时月度侵蚀持续削正统性，服从跟着回落；Mayor 更替时引擎 `resetJudicialForNewMayor` 丢弃离任者影响力（继承空洞的机制对应物）；W 回升使 Regime 回摆 IT，资格失效（服从数值仍在，秩序已不归它）。
- **内容不写资格**：资格完全由引擎读事实（`personalRuleEligible`：现任具名 Mayor、W ≤ 30、正统性 ≥ 0.35、服从 ≥ 0.60、Regime ≠ IT）。节点只改变速率与可达性（ADR-0010）。

## 6. 与《公议》的软互斥：分岔、改道与代价

- **分岔点**：材料的去向。同一份责任链材料（法警调度证据，共享 `bailiff_*` 链）喂给 Mayor 圈层是《人治》（F-1），喂给制度内程序派是《公议》（`charter_draft_step`）。两线的接触卡（M-1 与 `charter_contact_offer`）是同一时期的两扇门，关系门槛分别挂在 Mayor 与 Judicial。
- **中途改道（人治→公议）**：H-1 埋程序种子即改道——种子键为 `charter_contact_offer` 追加解锁路径，即使规则断点未点亮也能进约章线。**代价**：约章线从接触重新开始，锁定日仍要等 W 窗口；本线已消耗的喂证与担当不转移为约章步骤。
- **中途改道（公议→人治）**：拒绝/暂缓约章接触后，材料转喂 Mayor 圈层——M-1 接触卡不受约章线影响，照常可开。**代价**：约章接触键已消耗，本局没有第二次约章接触；起草动能归零。
- **并存**：约章锁定（Regime=PACT）不阻挡《人治》（资格只要求 Regime ≠ IT）；两收束可同池竞争，同分按既定序 RULE_OF_LAW 先于 PERSONAL_RULE（既有 P3 顺序，不改）。W 领地嵌套：约章要 W ≤ 18，《人治》只要 W ≤ 30——18 < W ≤ 30 的中间地带只有《人治》可成立。
- **不设硬锁**：两条路径不构成附着层面的分叉（司法部不归市长），互斥只在材料与时间层面，无二选一闸门。

## 7. 失败与放弃路径的下落（不悬空）

| 路径 | 下落 |
| --- | --- |
| M-2 绕过 | 线永久关闭（接触键已消耗）；纯秘密路线，局面按惯性滑向《深闭》，`normalcy_hardens` 照常给征兆 |
| P-B 留作筹码（不喂证） | 线停在接触步骤；影子权威路径违背 the Witness 判据，不设机制；局面滑向《深闭》 |
| 正统性不足（<0.5） | `mayor_public_stand` 永不点火；线停在喂证，局面按实际权力状态滑向《深闭》 |
| W 回升、IT 重新接管 | 引擎事实：Regime 回摆 IT，资格失效；滑向《深闭》（既有仲裁口径） |
| Mayor 死亡/更替且无继承安排 | 引擎 `refreshMayorHolder`／`resetJudicialForNewMayor` 接管：正统性与影响力重算；视后续状态入《深闭》（旧秩序回填）或《断层》（争夺空缺）。担当节点不绑持有者身份，新市长攒够正统性可重新点亮 |
| Mayor 成为撤离公开领头 | 《越岭》节点群已落地；持有者随队离开的耦合仍未接线（撤离结算按聚合人数，不按名清退 holder）。留依赖注释：持有者随队离开则本线失去承载（02 分册 §4 已定口径）；持有者死亡/更替时引擎读现任 holder，资格自然失效 |
| 埋程序种子 | 改道《公议》：代价与路径见 §6 |

## 8. 与树上成就的对应表

| 树成就 | 点亮依据 | 备注 |
| --- | --- | --- |
| 《人治》（收敛层） | 人治秩序事实（仲裁读 `personalRuleEligible` 四条件） | 树与仲裁读同一组事实；链完成键是路径投影，不另造计数器 |
| 接触／喂证／担当／警察转向／继承（路径节点） | 对应完成键 | 树上收敛层目前为单节点；若树批次细分，按 M-1／F-1／S-2／S-3／H-1 一一对应 |

成就与事件链完成键一一对应，不另造计数器（已定）。

## 9. TODO(balance) 汇总与明确不做

| 留空数值 | 所在节点 |
| --- | --- |
| 接触卡解锁关系阈值（Mayor ≥ 0.15）与 offer 动作成本 | M-1 |
| 喂证冷却（3 月）、喂证正统性增量（+0.02） | F-1 |
| 危机舞台冷却（9 月）、恐慌增量（+0.03）、叛乱门槛（0.3） | S-1 |
| 担当冷却（9 月）、正统性门槛（0.5）、效果（rebellion −0.05、panic −0.05、legitimacy +0.05） | S-2 |
| 警察转向门槛（服从 ≥ 0.60，镜像引擎） | S-3 |

明确不做：不写资格判定（引擎读事实）；不给司法部设市长附着（三态口径）；不新造计数器与成就编号体系；不引入影视改编人名；不新增阶段（stage）； Mayor 自始至终不知道其他地堡存在。

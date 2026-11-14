# 《公议》事件链 · 完整节点设计

> [返回总索引](../silo40-endings.md) · [链骨架](./15-chain-rule-of-law.md) · [剧情台阶](./01-rule-of-law.md)
> 性质：**内容节点设计文档**。沿用 [23《共证》节点设计](./23-information-nodes.md) 的格式与纪律：只写设计，数值一律 `TODO(balance)`，依赖未落地机制处只留注释。
> 口径：司法部只在 IT／SWING／约章三态间迁移，市长只有影响力修正（[ADR-0005](../../../knowledge/adr/0005-judicial-not-mayors.md)）；不设显式判定门，锁定经事实结算（[ADR-0008](../../../knowledge/adr/0008-escape-gate-to-outcome-layer.md) 的同源原则）；玩家是干涉者不是指挥者，只能改变速率与可达性（[ADR-0010](../../../knowledge/adr/0010-player-as-intervener.md)）。

## 0. 全线总览

```text
共同开局（落脚／裂缝／材料离手）
        │
        ▼
  B-1 规则断点（order_chain_breach）
        │ 材料指向规则漏洞而非个人罪行（抉择 L-A 投影）
        ▼
  T-1 接触卡（charter_contact_offer）
        │
   ┌────┼────────────┐
   ▼    ▼            ▼
 接受  拒绝        暂缓          ← 三选一完成键（T-2）
   │    └──┬──┘ 本线关闭；材料可转喂 Mayor 圈层（转《人治》节点群，待编写）
   ▼
  D-1 起草步骤（charter_draft_step，REPEATABLE 心跳）
   │   玩家供证不代笔：既有 Share Info 流通喂给起草会（抉择 L-C 投影）
   ▼ 步骤计数达标
  D-2 起草会议召开（charter_draft_convened）
   │
   ▼ 相对 Y5 起 ＋ 引擎锁定窗打开（W≤18 ∧ Q≥0.85 ∧ 独立事实≥2）
  K-1 锁定日（charter_lock_day，抉择 L-D 投影）
   │   锁定本身＝引擎约章动力学的事实（PactLocked），内容不写、不批准
   ▼
  仲裁读 PactLocked → RULE_OF_LAW 候选

并行：K-2 守卫超时（charter_window_expiry，接受后 +6 月，未锁定则动能流失）
并行：资格供给经共享法警证据链（bailiff_*，与《共证》共用键）
```

顺序：断点（B-1）→ 接触（T-1→T-2）→ 起草（D-1→D-2）→ 锁定（K-1）。并行：守卫（K-2）自接受起挂表；资格供给全程由共享证据链与引擎动力学承担，内容链不另造计数器。

## 1. 断点簇：问题从"谁"换成"依据什么"

### B-1 规则断点（order_chain_breach）

| 项 | 内容 |
| --- | --- |
| 触发条件 | 首次分享成立（`player_action:action_share_info` 完成键）且 `history_burden ≥ 0.10`（sim 校准起点）。历史包袱由正史锚点（`archive_truth_broadcast` +0.08）与早期流通自然累积，开局数月内可达 |
| 地点与来源 | Judicial 侧第一份断点材料：执法调度缺本地合法来源、事后补签倒填日期、IT 以非正式渠道调动物资人员。这类断点**不需要跨堡知识就能核验**（剧情台阶 L-2） |
| 消耗 | 无直接消耗（由分享动作带出） |
| 产出完成键 | `special:order_chain_breach` |
| 效果 | `professionFragmentAdd("Judicial")`（断点材料入司法侧）；`history_burden +0.02`（sim 校准起点） |
| 玩家反馈 | 材料状态从"某人做了坏事"改写为"这条命令没有合法来源"——可核验的缺口被标出 |
| 分岔 | 抉择 L-A 的机制投影：本节点只走"指向规则漏洞"一支；指向个人责任的一支转《人治》前奏（该线节点群未编写，见 §7） |

## 2. 接触簇：程序派联系人

### T-1 接触卡（charter_contact_offer）

| 项 | 内容 |
| --- | --- |
| 形态 | 行动设计 15 §5.3 的标准接触卡：`action_gates` 解锁 → 玩家打开接触并完成 offer 动作 → 三选择解锁 |
| 解锁条件 | `event_completed special:order_chain_breach` ＋ 玩家与 Judicial 关系 ≥ 0.2（sim 校准起点）。接触方是 Judicial 程序派联系人：他未必怀疑体制，只是无法忍受手续不全的命令 |
| 产出完成键 | `player_action:charter_contact_offer` |
| 效果 | 无机制效果（纯接触节点，`apply` 返回空） |
| 玩家反馈 | 一张有来源的接触卡；未决定前只保留一张，不刷屏 |

### T-2 三选一（charter_contact_accept / decline / defer）

| 项 | 内容 |
| --- | --- |
| 形态 | 三个 `PLAYER_EVENT` 内容行动，接受／拒绝／暂缓各一键（接触链范式约定） |
| 解锁条件 | `event_completed player_action:charter_contact_offer`（三键同解） |
| 接受效果 | `schedule_event(charter_window_expiry, 6)`：守卫挂表（K-2） |
| 拒绝/暂缓效果 | 无机制效果；本线关闭，材料可转喂 Mayor 圈层——转《人治》节点群（未编写，事件内留注释指依赖） |
| 产出完成键 | `player_action:charter_contact_accept`／`..._decline`／`..._defer` |
| 已知缺口 | 门控条件不支持否定：选定其一后，兄弟选择不能被隐藏（`action_gates.json` 无"未发生"谓词）。下游只读接受键，多选不构成机制矛盾；隐藏能力等门控批次 |

## 3. 起草簇：约章不是玩家的文件

### D-1 起草步骤（charter_draft_step）

| 项 | 内容 |
| --- | --- |
| 触发条件 | `player_action:charter_contact_accept` 与 `player_action:action_share_info` 两完成键均已触发；`REPEATABLE`，冷却 2 月（sim 校准起点） |
| 心跳语义 | 与《共证》`replication_progress` 同型：完成键持久存在，月度持续评估使起草按冷却节奏推进；每一步对应一轮"证据流通喂给起草会"。不拦截、不替代 Share Info 本身的职务结算 |
| 消耗 | 无直接消耗（由既有流通动作带出） |
| 效果 | `beliefObserve("Judicial", 核心法警调度事实, 0.5)`（sim 校准起点）：制度内喂证深化司法层信念——这是本线玩家作用的速率表达（ADR-0010）：供证加速司法层越过资格门槛，不替代门槛本身。事实 ID 按步骤计数奇偶在两份核心事实间轮换（确定性） |
| 产出完成键 | `special:charter_draft_step`（计数键；步骤档由 `trigger_count` 表达，不另立键） |
| 玩家反馈 | 起草会逐条争吵的进展：条款、妥协、缺陷，以及本地起草人的名字（功能位） |
| 分岔 | 抉择 L-C 的机制投影：本节点只走"让他们逐条争吵、玩家供证不代笔"一支；替起草的一支不设机制（玩家烙印风险属叙事纪律） |

### D-2 起草会议召开（charter_draft_convened）

| 项 | 内容 |
| --- | --- |
| 触发条件 | `special:charter_draft_step` 已触发且 `trigger_count ≥ 2`（sim 校准起点） |
| 效果 | `history_burden +0.03`（sim 校准起点） |
| 产出完成键 | `special:charter_draft_convened` |
| 玩家反馈 | 约章文本从本地人手里长出来：Judicial 引用旧规程、部门代表争论条款、Mayor 一方核算执行成本 |

## 4. 锁定簇：锁定日与守卫

### K-1 锁定日（charter_lock_day）

| 项 | 内容 |
| --- | --- |
| 触发条件 | `special:charter_draft_convened` 已触发 ＋ 相对 Y5 起（代码历 `timestampAtOrAfter(127,1)`，sim 校准起点）＋ 引擎锁定窗已打开 |
| 锁定窗读数 | `canTrigger` 读 `ctx.authority`：`w ≤ 18` 且 `public_evidence_q ≥ 0.85` 且 `independent_public_facts ≥ 2`（sim 校准起点；三值镜像引擎 `PactOpenW / PactMinQ / PactMinFacts`，引擎改值时内容同步） |
| 效果 | `rebellion −0.04`、全部门 `panic_value −0.04`（sim 校准起点） |
| 产出完成键 | `special:charter_lock_day` |
| 锁定的归属 | **锁定本身不是本节点的效果**：`PactLocked` 由引擎约章动力学在锁定窗持续满足后写入（单向事实，市长更替不撤销）。本节点是抉择 L-D 的叙事投影——玩家用剩余关系与威望为锁定程序争取那一段时间；结局资格读引擎事实，不读本完成键（ADR-0010：结局不靠"玩家必须做过什么"的硬闸） |
| 玩家反馈 | 第一次按约章裁决的案件：裁决者引用约章与公开证据，而不是等待上级指示 |
| W 依赖 | `W ≤ 18` 是引擎锁定窗的一部分，内容不写 W、不自造 W 判定。**依赖**：W 的离散负冲击通道（1 号覆灭／终止链失效／大规模公开证据三类触发）在并行轨实装中；落地前锁定窗主要靠暴露高水位 E 的自然累积打开，本线节点在窗外等待，不催、不绕 |

### K-2 守卫超时（charter_window_expiry）

| 项 | 内容 |
| --- | --- |
| 形态 | 纯调度事件（`canTrigger` 恒假），接受时 `schedule_event(+6 月)` 挂表；到期无条件派发（机制底座 §3 守卫约定） |
| 守卫逻辑 | `apply` 读 `ctx.states`：`special:charter_lock_day` 已触发 → 空转；未触发 → 未锁定分支 |
| 未锁定效果 | `rebellion +0.04`、全部门 `panic_value +0.04`（sim 校准起点）：动能流失，线不关闭——K-1 仍可在锁定窗打开后点亮 |
| 产出完成键 | `crisis:charter_window_expiry`（超时事件本身不作失败条件被订阅） |

## 5. 资格供给：共享键消费与不发明

- **数值侧供给已由共享链完成**：`bailiff_dispatch_evidence_a/b → bailiff_evidence_public_a/b`（《共证》同源消费）把两份核心"法警受 IT 直接调度"事实（`Judicial_1`／`Judicial_2`）登记为公开证据（provenance 1.0／coverage 0.9），并喂司法层信念。两事实达标时 `Q ≈ 0.95 ≥ 0.85`、独立公共事实 = 2——约章资格的数值门槛由共享链满足，本线**不重复登记、不另造事实键**。
- **本线对供给的唯一增量是速率**：D-1 的制度内喂证（`beliefObserve` 司法层）加速司法信念越过 0.85 门槛、推高 IT 归因版本权重，使 Q 更早成形。这是"玩家做事改变速率"的本线表达。
- **内容侧持久供给只走登记＋信念层**：`AdvanceAuthorityMonthly` 每月按"登记 × 司法信念层"重算 Q／独立事实／事实键表，内容对 `authority_evidence_delta`／`authority_public_fact` 的直写会被次月重算覆盖；`PactQualified` 由引擎在事实达标时单调置真。因此本线内容不写这三个通路，只读 `ctx.authority`。
- **接口换代免疫**：并行轨正把自动公开节点换为玩家发布动作。本线读登记状态（`ctx.public_evidence` 与 `ctx.authority`），不订阅供给事件的完成键，换代不影响本线。

## 6. 与《人治》的软互斥（新三态口径）

- 司法部只在 IT／SWING／约章三态间迁移；市长对司法部只有影响力（施压、换人、个案松弛），不构成归属。**任何判定与文案不得再读"司法部归市长"状态**（ADR-0005）。
- 约章锁定 → `Regime = PACT`。《人治》资格要求 `Regime ≠ IT`——PACT 不阻挡《人治》：两候选可同池竞争，同分时仲裁顺序 RULE_OF_LAW 先于 PERSONAL_RULE（既有 P3 顺序，不改）。
- W 领地上两者嵌套：约章锁定窗 `W ≤ 18` 是《人治》低威势窗 `W ≤ 30` 的子集。威势退场而未锁定的中间地带（18 < W ≤ 30）只有《人治》可成立；锁定成立时两者按事实并存、按加权与同分顺序裁决。
- 不设二选一的硬锁（the Witness 收敛层已定）：硬互斥与"两者可部分并存、同分按序让《公议》"的既有裁决冲突。

## 7. 顺序、并行与互斥收口

- **顺序**：B-1 → T-1 → T-2（接受）→ D-1（×2 档）→ D-2 → K-1。
- **并行**：K-2 守卫自接受起挂表；资格供给（共享证据链＋引擎动力学）全程并行。
- **收口点一 · 拒绝/暂缓**（线内抉择）：本线关闭；材料转喂 Mayor 圈层属《人治》节点群（未编写，留依赖注释）。
- **收口点二 · 与《共证》的共用与分流**：两线共用法警证据链的完成键与登记。《共证》把材料推向公共扩散（部门信念覆盖），《公议》把同一份材料留在制度内渠道（司法层深化）。同一份材料在两条线里的去向由玩家流通方向决定，不设互斥闸；仲裁口径既有：约章锁定作为《共证》的制度尾声候选，同分时《共证》优先。
- **收口点三 · 清洗链抢先**：锁定日前清洗链完成（`reset_completed` 事实键）→ 转《遗忘》：硬失败护栏先于制度成就（既有 P0 口径）。
- **终局时钟**：锁定是单向事实；窗口走完（相对 Y8 覆灭结算）时已锁定的约章进入仲裁，未锁定的按实际权力状态滑向《深闭》或《人治》。

## 8. 与树上成就的对应表

| 树成就 | 点亮依据 | 备注 |
| --- | --- | --- |
| 《公议》（收敛层） | 约章锁定事实（仲裁读 `Judicial.PactLocked`） | 树与仲裁读同一组事实；链完成键是路径投影，不另造计数器 |
| 断点／接触／起草／锁定日（路径节点） | 对应完成键 | 树上收敛层目前为单节点；若树批次细分，按 B-1／T-1／D-2／K-1 一一对应 |

成就与事件链完成键一一对应，不另造计数器（已定）。

## 9. TODO(balance) 汇总与明确不做

| 留空数值 | 所在节点 |
| --- | --- |
| 断点触发门槛 `history_burden ≥ 0.10` | B-1 |
| 断点效果 `history_burden +0.02` | B-1 |
| 接触卡解锁关系阈值（Judicial ≥ 0.2）与 offer 动作成本 | T-1 |
| 起草冷却（2 月）、喂证可靠性（0.5）、步骤档（2 次） | D-1／D-2 |
| 会议召开效果 `history_burden +0.03` | D-2 |
| 锁定日起点（相对 Y5）、锁定窗三读数（18／0.85／2，镜像引擎） | K-1 |
| 锁定效果 `rebellion −0.04`、`panic_value −0.04` | K-1 |
| 守卫延迟（+6 月）与未锁定效果 `rebellion +0.04`、`panic_value +0.04` | K-2 |

明确不做：不自造 W 判定、不写 W（等并行轨冲击通道，只标依赖）；不重复登记共享链已供给的事实键；不给结局资格加"玩家必须完成本链"的硬闸；不引入影视改编人名；不新增阶段（stage）；不新造计数器与成就编号体系。

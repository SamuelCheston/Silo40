# 18 · 市长与叛乱方谈判/妥协层设计（《越岭》立场生产层 · 谈判机制）

> 性质：**机制设计卷**。填 [36 节点设计](../story/endings/36-rebellion-nodes.md) §3 与 [37 事件规格](../story/endings/37-rebellion-event-spec.md) §1 登记的「完整谈判机制（市长三条路中的『谈判』）本期留空占位」。数值一律 `TODO(balance)`。
> 对齐：[ADR-0007](../../knowledge/adr/0007-weighted-random-stance.md)（立场是带权重的随机决策、磋商可持续重估）、[ADR-0008](../../knowledge/adr/0008-escape-gate-to-outcome-layer.md)（无显式判定闸、结算只读逃出人数）、[16 准备度设计](./16-preparation-outcome-design.md) §3.2。

## 1. 要覆盖的玩法（设计者原始意图）

1. **叛乱降低市长正统性**：叛乱持续侵蚀现任市长个人正统性（接既有 `MayorLegitimacy`，不新造状态；ADR-0007 后果条的誊录落地）。
2. **市长面对叛乱有三条路**：镇压、谈判、妥协。三条路是**带权重的随机决策**，权重读状态但不锁死——不存在「叛乱 ≥ X → 必须妥协」。
3. **谈判不是一次性通牒交换**：双方可以一直通气、反复试探；市长不必因为对方报了一个人数就立刻表态（拖延是合法结果）；即使叛乱方已经打到市长办公室，市长仍可在此刻妥协，**只要对方接受**——接受是叛乱方的带权决策，不是自动成立。
4. **妥协成立后，叛乱方可以选择不接管市长职位**：带着既有组织能力直接推进逃跑（或作为退路的外溢）计划。接管与否是玩家可做的取舍（参照 R-D 抉择的先例：玩家可选，未选时结算按中间口径）。
5. **三方取舍都是带权重的随机决策**：市长、叛乱方领导层、IT 各自有自由裁量空间；权重可归因（相对实力、正统性、压力、既有承诺、性格/特质），每个决策点在决策日志里留下可复盘的依据。
6. **这一层只改变「谁在阻碍谁」**：产出写入顶层立场的权重输入与组织事实，**不直接发放胜利**；最终结算仍然只看逃出去多少人（ADR-0008 纪律，另立 ADR-0013 长期约束）。

## 2. 状态（`model.NegotiationState`，挂堡聚合，存档 v12→v13）

```text
Initialized            是否开过信道
ChannelOpen            信道是否畅通
OpenedYear/Month       建信道时间
MayorDisposition       市长性情（开信道时一次性抽取，持久）：hardliner 强硬派 / pragmatist 务实派 / survivor 求生派
Rounds                 已交换回合数
MayorPath              市长最近一次决策的路径：suppress 镇压 / talk 谈判 / concede 妥协
MayorLean              玩家可感知的市长倾向 (0-1)：由回合产出的信号，不是立场本身
RebelPosture           叛乱方最近姿态：press 施压 / probe 试探 / offer 递条款
StalledRounds          连续「谈判无进展」回合数（谈判权重的软衰减输入，不是阈值闸）
SuppressRounds         累计镇压回合数（叛乱方拒绝权重的输入）
OfferOnTable/OfferedBy/OfferTerms   桌上的妥协提案（谁提的、赦免/职能保留/不清算口径）
CompromiseEstablished  妥协是否成立（市长让步＋叛乱方接受）
CompromiseYear/Month
Takeover               妥协后的选择：""（未选）/ takeover 接管 / hands_off 不接管
Closed/CloseReason     信道崩塌：crackdown 镇压断裂 / it_exposure 被 IT 曝光
Log                    决策日志，环形保留最近 24 条
```

**零值＝无谈判＝旧档语义逐位不变。** 升级器 `UpgradeV12ToNegotiationSave`：备份＋单事务＋回读校验，不回填任何推测状态。

### 决策日志条目（可复盘依据）

每条：年/月、决策方（mayor/rebels/it）、决定、归一化权重（三位小数）、因子摘要（正统性、叛乱、W、警察服从、保障可信度、性情、压力、拖延疲劳等当时读数）、结果。日志随存档持久化，前端呈现最近数条，全部条目可经快照审计。

## 3. 决策点（全部走引擎单一 RNG 流，固定取样顺序）

### 3.1 市长路径决策 `drawMayorPath`

- **时机**：(a) 信道畅通期间每个月度 tick 一次低强度通气；(b) 玩家执行 `NEGOTIATION_PROBE` / `NEGOTIATION_OFFER_TERMS` 时一次高强度回合。
- **三档**：镇压 suppress / 谈判 talk / 妥协 concede。
- **权重输入**（各系数 `TODO(balance)`）：
  - 基线 `{suppress 0.40, talk 0.40, concede 0.20}`；
  - 个人正统性（`MayorLegitimacy`）：高 → 镇压仍可行（suppress↑ concede↓）；
  - 叛乱水平：高 → 镇压不可行（suppress↓ concede↑），读权重不设阈值；
  - 相对实力：IT 残余 W 与警察服从构成的镇压力量 vs 叛乱 → suppress±；
  - 保障可信度（`TopStance.Guarantee`）：可信的出路 → concede↑ talk↑；
  - 性情（开信道时抽取，持久）：强硬派 suppress 加权、务实派 talk 加权、求生派在命令源枯竭/低 W 时 concede 加权；
  - 压力：`order_pressure`（既有）与开窗事实 → concede↑；
  - 拖延疲劳：`StalledRounds` 软衰减 talk 权重——谈判可以一直拖，但拖不是永远免费的；这是权重衰减，不是强制阈值。
- 取样后写决策日志（权重＋因子摘要）。

### 3.2 路径结果

| 路径 | 结果 |
| --- | --- |
| 镇压 | `rebellion −`、全部门 `panic_value +`、市长正统性 `−`（镇压有代价）、保障可信度 `−`（背信）、`SuppressRounds++`；再按强度加权抽信道是否被打断（`Closed=crackdown`，合法失败结果）。反馈：镇压叙事 |
| 谈判 | `Rounds++`、`StalledRounds++`、`MayorLean` 按性情微漂移；**无决议**。反馈明说「谁在拖延」，不伪装成进展 |
| 妥协 | 市长让步：桌上有叛乱方提案 → 市长接受之；无提案 → 市长开出赦免/职能保留条款。随后**触发叛乱方接受决策**（3.3） |

### 3.3 叛乱方决策 `drawRebelChoice`

- **接受决策**（市长让步时）：接受 / 拒绝，带权抽取。输入：叛乱水平（高 → 更敢拒绝）、保障可信度与条款内容（可信 → 接受↑）、近期镇压（`SuppressRounds` → 拒绝↑）。接受 → `CompromiseEstablished`；拒绝 → 提案作废，市长下一轮镇压权重上升（记日志）。**「只要对方接受」的落点。**
- **姿态决策**（信道畅通且市长上一轮为谈判时，月度）：施压 press（`rebellion +` 小量）/ 试探 probe（`MayorLean` 观测信号）/ 递条款 urge（提示玩家可递提案，无直接数值）。

### 3.4 IT 反应 `drawITReaction`（第三方，信道畅通期间月度）

- 三档：不干涉 stand_aside / 破坏 undermine / 观望 watch。
- 输入：W 残余＋1 号专线仍在下令 → 破坏↑；保障可信度（求生算计）→ 不干涉↑；叛乱水平 → 观望/不干涉。
- 破坏：保障可信度 `−`，再按强度加权抽信道是否被曝光（`Closed=it_exposure`，`panic_value +`、`rebellion +`）。IT 的裁量独立于市长与叛乱方。

### 3.5 月度回合内的固定取样顺序

`市长路径 → 叛乱方（接受/姿态）→ IT 反应`。顺序固定写死在代码注释里，保证同种子复现。

### 3.6 妥协成立的即时效果

- 保障可信度 `+`（妥协本身是最大的保障，增量夹 [0,1]）；
- `topStanceReeval("both")`——保障输入已抬升，立场重估是权重与种子的产物（最后一刻妥协也算，ADR-0007）；
- `rebellion −`（诉求部分兑现），**组织能力不受损**（叛乱方的组织事实不被妥协消耗）；
- 写妥协成立完成键，解锁妥协后二选一。

### 3.7 妥协后二选一（玩家动作，参照 R-D 先例）

| 动作 | 效果 |
| --- | --- |
| `NEGOTIATION_TAKEOVER` 接管市长职位 | 正统性改写为委员会正统性（`TODO(balance)`）；市长立场以「叛乱方主导顶层」的权重输入重估（仍是带权抽取，不写死）；`rebellion −`（诉求兑现）；写接管事实 |
| `NEGOTIATION_HANDS_OFF` 不接管，带组织走 | 市长留任；叛乱方组织转向离开：出发结算的组织动员读 `hands_off` 加成（`TODO(balance)`），外溢信道可靠度小量上升（夹取） |

未作选择时结算不读加成（与 R-D 未选择用中间系数同口径）。**两个动作都不发放胜利、不写资格**：接管改的是「谁在阻碍谁」，不接管改的是组织事实，结算仍只看逃出人数。

## 4. 叛乱侵蚀正统性（ADR-0007 誊录）

月度政治更新追加：`legitimacy −= erosionRate × rebellion × deltaYears`，`erosionRate TODO(balance)`。叛乱为 0 的局逐位不变。这是纯算术，不消耗 RNG。

## 5. 内容与动作清单

新增封闭集效果（37 §0 风格）：`negotiation_open`（开信道＋抽性情）、`negotiation_round`（flag=probe|terms，高强度回合）、`negotiation_resolve`（flag=takeover|hands_off）。JS helper：`negotiationOpen()/negotiationRound(kind)/negotiationResolve(kind)`。ctx 增读 `negotiation`（只读视图）。门控指标增 `negotiation_channel` / `negotiation_compromise` / `negotiation_takeover_decided`（0/1）。

| 动作 | 事件文件 | 规格 |
| --- | --- | --- |
| `NEGOTIATION_OPEN_CHANNEL` | `events/player_actions/negotiation_open_channel.js` | ap 2，ONCE；门控 `rebellion ≥ 0.30`（`TODO(balance)`）；效果 `negotiationOpen()` |
| `NEGOTIATION_PROBE` | `negotiation_probe.js` | ap 1，REPEATABLE 冷却 1 月；门控信道畅通；效果 `negotiationRound("probe")` |
| `NEGOTIATION_OFFER_TERMS` | `negotiation_offer_terms.js` | ap 2，ONCE；门控信道畅通；效果 保障可信度 `+`（条款可信）＋ `negotiationRound("terms")` |
| `NEGOTIATION_TAKEOVER` / `NEGOTIATION_HANDS_OFF` | `negotiation_takeover.js` / `negotiation_hands_off.js` | ap 0；门控妥协成立；效果 `negotiationResolve(...)` |

既有 `ESCAPE_NEGOTIATION_GUARANTEE`（保障清单）**保持不变**：它仍是最小保障动作，与本层并存（保障可信度是同一权重输入）。

## 6. 月度钩子与 RNG 纪律

- 钩子位置：单堡路径 `runOneMonthSingle` 与世界路径阶段 1（各堡本地结算），`UpdateSiloState` 之后、时钟推进前；仅当 `ChannelOpen && !Closed && !CompromiseEstablished` 时执行回合。
- **未开信道的运行不消耗任何新 RNG**：全部既有验收运行（《公议》《人治》《无人裁决》《共证》）轨迹逐位不变。这是本层的非回归核心保证。
- 开信道后的运行消耗新 RNG，轨迹改变是机制预期。
- 妥协成立后月度回合停止（谈判阶段结束，进入选择阶段）。

## 7. 玩家侧反馈

- 每个决策点产出中文 `StoryEvent`：谁提了什么、谁在拖延、市长此刻的倾向（用 `MayorLean` 的可感知口径）、镇压与曝光的失败叙事。失败与拖延明说，不伪装成进展。
- 决策日志条目含归一化权重与因子摘要，供复盘。
- 前端：`LinesPanel` 增谈判块（信道状态、回合数、市长倾向、桌上提案、妥协状态、最近日志）；二选一动作走既有动作按钮通道。

## 8. 非回归论证

1. 未开信道 → 无新 RNG 消耗、无新状态读写 → 既有验收逐位不变（唯一例外是 §4 侵蚀，属既有裁决誊录）。
2. 仲裁永不读谈判状态；出发结算仅在 `Takeover=hands_off` 时读新加成输入，零值路径不变。
3. `EscapedGroup` 资格接口不触碰；立场仍不作判定闸、不作玩家按钮。
4. 若既有测试因 §4 侵蚀需要重调，按「侵蚀是设计前提」说明判据（优先保守校准侵蚀系数），不放宽谈判机制本身。

## 9. 明确不做

- 不做谈判胜利：任何谈判结果不写资格、不发结局。
- 不做通牒交换式的一次性谈判（与 1 号反应链的通牒是两回事，不互相接线）。
- 不让谈判状态进仲裁、进结局权重。
- 不为 NPC 堡代开信道（信道只由玩家动作开启；机制本身是堡通用的）。
- 不新增随机源：全部取样经 `engine.DrawFloat64`。

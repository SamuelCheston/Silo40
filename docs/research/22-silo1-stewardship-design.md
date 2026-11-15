# 22 — 1 号地堡权衡层设计（Silo 1 Stewardship）

> 状态：设计写定（实现进行中）
> 对齐：[ADR-0015](../../knowledge/adr/0015-silo1-stewardship-blackbox.md)（1 号权衡层：黑箱博弈与 selfPreservation 动力学）、[ADR-0016](../../knowledge/adr/0016-safeguard-from-accumulation-to-assignment.md)（safeguard 由后果累加改为 1 号指派）、[01 世界观](./01-worldbuilding.md)、[12 结局系统](./12-ending-system-design.md)。
> 上游裁决记录（按时间序）：
> 1. 离心不是压风险度的负反馈刹车——执行层腐烂与「下一个 40 号」风险进入带权决策；
> 2. IT 执行强度对风险度是单峰曲线，1 号选的是峰，曲线参数化 TODO(balance)；
> 3. 曲线对 1 号是黑箱——决策代码不得读曲线参数，只能读观测，日志体现「它在猜」；
> 4. 风险度不直接乘执行强度，而是决定隐藏指标 `selfPreservation` 的增长率；指标有月级时间常数，禁止快速探针；峰是风险与指标共同形成的动态结果；占位命名 `selfPreservation`（底下人不知道 1 号存在，不构成「对 1 号的离心」），正式命名待研究轨候选。

## 1. 目标与范围

给 1 号地堡加一套「它自己也在做权衡」的指标与决策层。1 号按 order 办事，order 的目的是让实验顺利跑完；本层让 1 号在三个维度上付出真实代价并做出可归因的取舍：

- **指标一：实验履约与保留多样性（1 号侧）**。
- **指标二：剩余人员平均寿命**（1 号人员轮流休眠，行动与时间都消耗寿命）。
- **指标三：safeguard 风险度改由 1 号指派**，指派是黑箱下的带权随机决策，伴随执行层的 selfPreservation 动力学。

**非目标（本轮不做）**：唤醒人数机制；探索策略/信念更新的最优解（另有研究轨出规格，本实现只留接口与保守默认策略）；对玩家结局资格的任何新判定。

**硬约束**：

- 与现有 1 号反应链（`order_pressure ≥ 0.45` 且宣传 ≥ 2 触发整肃/终止/投送，见 [content_events.go L76-105](../../internal/engine/content_events.go)）是**两件事**：反应链门槛不变，本层不新增触发门槛。
- 指标一不直接判负、不直接决定处刑；只影响「最终可保留地堡的多样性」。
- 单堡（空邻堡）配置下**逐位不变**：本层全部早退，不消耗 RNG、不写状态。
- 确定性：同种子 + 同操作序列逐字节复现；所有随机取样走引擎单一 RNG 流（[rng.go](../../internal/engine/rng.go)），取样顺序固定（阶段 2b 内按 SiloID 升序）。
- 18 号正典链按时间戳强制触发，不读本层任何状态；本层不得阻塞正典。
- 世界观安全：底下人不知道 1 号存在。玩家侧叙事中，执行层的态度变化只能表述为「对 order/上层指令」的态度，不得点名 1 号。

## 2. 总体架构

```
┌─────────────────────────── 物理层（世界侧，黑箱壁之内） ───────────────────────────┐
│ safeguard_physics.go（engine 包）                                                    │
│   S（selfPreservation）动力学：dS = α·max(0, r−r_crit) − β·S，夹 [0,1]              │
│   执行强度 E(r, S) = bump(r) · (1−S)^η   ← bump 单峰，参数仅物理层可读              │
│   观测生成：带延迟与噪声的观测包（ObservationBatch）                                  │
└──────────────▲──────────────────────────────────────────────┬───────────────────────┘
               │ 读取指派值 r（Silo.SafeguardRisk）            │ 观测包（不含真值）
┌──────────────┴──────────────┐                  ┌────────────▼───────────────────────┐
│ 本地堡机制消费方（黑箱壁内）  │                  │ 1 号决策层（service 包，黑箱壁外）   │
│  escape_response 权重        │                  │  信念状态（估计±不确定性）            │
│  it_authority 稽查强度        │                  │  带权随机指派（风险度+监控档位）      │
│  玩家阻力/抗命事件呈现        │                  │  决策日志（归因+「它在猜」）          │
└─────────────────────────────┘                  └────────────────────────────────────┘
```

**黑箱壁纪律**：1 号决策层代码（`internal/service/stewardship_*.go`）禁止 import/调用物理层函数、禁止读 `Silo.SelfPreservation` 真值、禁止读 bump 参数；它只能消费 `ObservationBatch`（观测结构体）。物理层可以读指派值。违反即验收失败（见 §9 A6）。

**挂载点**：[world.go](../../internal/service/world.go) 多堡月度管线阶段 2b（`runSilo1StateMachine` 之后、阶段 2c 之前），新增 `runSilo1StewardshipMonth`。单堡路径（`runOneMonthSingle`）不挂载。1 号覆灭（`!world.Silo1.Alive`）后本层早退（已指派值冻结，S 仍按最后指派值继续物理演化一个月后冻结——冻结语义见 §6.4）。

**文件布局**：

| 文件 | 属格 | 内容 |
|---|---|---|
| `internal/engine/safeguard_physics.go` | 格三 | S 动力学、E(r,S)、观测生成、参数（全 TODO(balance)） |
| `internal/service/stewardship_experiment.go` | 格一 | 指标一：履约与多样性 |
| `internal/service/stewardship_lifespan.go` | 格二 | 指标二：平均寿命 |
| `internal/service/stewardship_assignment.go` | 格三 | 指标三：信念、带权指派、决策日志 |
| `internal/service/silo1_stewardship.go` | 主线 | 调度器（固定顺序调用三格月度函数） |
| `internal/service/save_upgrade.go`（v15 段） | 格四 | 迁移器 |
| `internal/service/stewardship_acceptance_test.go` | 格五 | 无头实测 |
| `frontend/src/components/WorldPanel.tsx`（1 号动向节） | 格五 | 最小呈现 |

## 3. 指标一：实验履约与保留多样性

### 3.1 语义

1 号按 order 办事，order 要求清除失控的叛乱地堡；但实验的最终目的是保留一个——每清除一个候选，「最后可以保留谁」的多样性就差一分。

- **该清除的没清除** → 履约缺口（`FulfillmentDeficit`）恶化。
- **每清除一个地堡** → 多样性损耗（`DiversityLoss`）恶化。
- 清除带来的这部分恶化**随时间衰减**，且**越接近实验结束越轻**（候选本来就要收拢到一个）。
- 该指标只影响「最终可保留地堡的多样性」：作为 1 号后续指派决策的输入因子（多样性低时更忌讳极端手段）与终局审计的呈现项。**不判负、不处刑、不进玩家结局资格。**

### 3.2 事实口径

- **清除义务登记（scheduled）**：某堡被签发通牒（`silo1_ultimatum_cross`）时，登记 `(siloID, 月份)`。通牒是反应链的唯一入口，即「按要求应清除」的起点。
- **清除成立（cleared）**：被登记堡的 `VictoryStatus.Type == DEATH`（任意来路——终止投放、物理打击、整肃后崩溃——都算候选减少；多样性关心的是候选数量，不是处刑方式）。检测点：阶段 2b 内按 SiloID 升序扫描 `siloEnded`/DEATH。
- **未清除（miss）**：登记后超过反应链完整阶梯窗口 `clearDeadlineMonths`（= 24，TODO(balance)：通牒 U+0 到打击 U+19 再加缓冲）仍未 DEATH，记一次 miss，`FulfillmentDeficit += missPenalty`（TODO(balance)，初始 0.10）。每个登记只产生一次 miss（一次性评估 + 状态标记，不重复计费）。

### 3.3 数值

```
实验结束参照：Y130M1（18 号正典终局信号，18-chain N9）。totalWindow = 96 月（Y122M1 起）。
proximity(m) = clamp01(monthsRemainingUntilY130M1(m) / totalWindow)   // 越接近终局越小
清除时：DiversityLoss += clearPenalty × proximity(m)                   // clearPenalty TODO(balance)，初始 0.15
每月衰减：FulfillmentDeficit ×= (1 − decayRate)；DiversityLoss ×= (1 − decayRate)
          decayRate TODO(balance)，初始 0.02/月
```

**保留多样性读数**（供决策因子与呈现）：`RetentionDiversity = clamp01(candidatesAlive/initialCandidates − wD·DiversityLoss − wF·FulfillmentDeficit)`，wD、wF TODO(balance)。candidatesAlive = 未覆灭的非 1 号堡数。

## 4. 指标二：剩余人员平均寿命

### 4.1 语义

1 号的人是轮流休眠的。平均寿命是一个**必然消耗**的预算：时间本身让它下降（什么都不做也保不住），行动按档位额外扣。所以 1 号的理性是「横竖会掉，把寿命用起来做事」，而不是囤积。

### 4.2 数值

```
初始 AvgLifespan = lifespanStart（TODO(balance)，初始 20.0 年）
每月自然衰减：− lifespanNaturalDecay（TODO(balance)，初始 0.12 年/月）
监控档位（每堡一份，1 号指派）：
  HIGH（高度监控某堡）    − postureCostHigh   /月（TODO(balance)，初始 0.10）
  NORMAL（正常监控）      − postureCostNormal /月（TODO(balance)，初始 0.04）
  REPORT_ONLY（只听取 IT 主任汇报） − postureCostMin /月（TODO(balance)，初始 0.005）
一次性行动成本：签发通牒 − opCostUltimatum；整肃链点火 − opCostPurge；终止链点火 − opCostTermination（均 TODO(balance)）
```

### 4.3 耗尽语义（能力退化，不是判负）

- `AvgLifespan ≤ postureDowngradeThreshold`（TODO(balance)，初始 3.0）：所有 HIGH 档强制降为 NORMAL。
- `AvgLifespan ≤ 0`：夹 0；全部档位强制 REPORT_ONLY；1 号**不再签发新的通牒**（反应链入口关闭），已点火阶梯继续走完。
- **正典豁免**：18 号链与一切时间戳强制节点不读本指标；寿命耗尽不得阻塞正典。
- 档位指派本身是决策层动作（§6.3），与风险度指派同一回合并计入决策日志。

## 5. 指标三总览：从累加到指派

现状（本次改动前）：`Silo.SafeguardRisk` 由后果累积——协议偏离（[protocol.go recordDeviation](../../internal/engine/protocol.go)）、IT 权威行动 RSDelta（[it_authority.go](../../internal/engine/it_authority.go)）、筹备反制不作为/蒙蔽崩塌（[escape_response.go](../../internal/engine/escape_response.go)）、内容事件效果；`≥ 1.0` 触发硬失败 DEATH（[engine.go L798](../../internal/engine/engine.go)、core_systems.js VICTORY_CHECK）。

新模型：

1. `Silo.SafeguardRisk` 的语义改为 **1 号对该堡指派的风险度**。无上限（可以 0.99，也可以更高）；不再由后果直接累加。
2. 原累加点全部改为**事件登记**（incidents），进入 1 号决策的输入因子：不作为、蒙蔽崩塌、协议偏离、IT 过度干预等照旧发生、照旧记录，但不再直接改写风险度。
3. **移除 `≥ 1.0` 硬失败**（Go 与 JS 两侧）。高指派不再当场处刑，而是通过 §6 动力学腐蚀执行层——这是上游裁决的直接推论（「它可以固执地维持 99%，代价是下面执行层废掉、玩家那边因此变容易」）。既有依赖硬失败的测试按新语义改写（清单见 §8.2）。

## 6. 指标三：selfPreservation 动力学与黑箱决策

### 6.1 物理层：隐藏指标 S

每个被 1 号监控的堡有一个隐藏指标 `S ∈ [0,1]`（占位名 `selfPreservation`，正式命名待研究轨候选；底下人不知道 1 号存在，S 表达的是「对 order 压力的自保倾向」而非「对 1 号的离心」）：

- `S = 0`：底下完全按 order 办事。
- 中段：消极拖延、阳奉阴违。
- `S → 1`：公开叫板。

月度动力学（物理层唯一真源，`engine.UpdateSelfPreservation`）：

```
dS = α · max(0, r − r_crit) − β · S          // 夹 [0,1] 后写回
α  = spGrowthRate      TODO(balance)，初始 0.30/月（每单位超额风险）
r_crit = spCritRisk    TODO(balance)，初始 0.35（正常水平上界；低于它 S 自然回落）
β  = 1/τ，τ = spTimeConstantMonths TODO(balance)，初始 8 月（月级时间常数）
```

**时间常数的设计意图**：τ 月级意味着「试几个回合定位峰」不成立——任何快速探针策略在 τ 内得到的都是过渡态读数。**默认决策策略禁止探针行为**（§6.3），决策冷却 ≥ 3 月（TODO(balance)）。

### 6.2 物理层：执行强度 E(r, S)

```
bump(r)：单峰。峰位 μ、峰高 H、左宽 σL、右宽 σR，四参数全部 TODO(balance)，
         初始 μ=0.45, H=1.0, σL=0.25, σR=0.30（分段二次，超出 [μ−σL, μ+σR] 为 0）。
         低端（风险很低）：IT 觉得没必要卖力，稽查与镇压松懈 → 趋近 0。
         中段：「服从能换命」，唯一峰值区。
         高端（风险很高）：服从换不来命 → 回落。
E(r, S) = bump(r) · (1 − S)^η，η = spErosionExp TODO(balance)，初始 1.0。
```

**峰是动态的**：S 由 r 的历史路径驱动，E 的有效峰位随 (r 历史, S) 移动——不存在 1 号可读的静态最优值。bump 四参数与 S 真值都在黑箱壁内。

**E 的消费方（本地堡机制，壁内）**：

1. `EscapeResponseDecisionWeights`：原 `rs := clampUnit(silo.SafeguardRisk)` 对镇压/不作为的加减项改由 `E` 驱动（E 高 → IT 积极替 order 了结异常；E 低 → 睁只眼闭只眼）。
2. IT 权威稽查强度：月度 IT 行动（protocol_disposal / suppress_signal 类）的触发概率与力度乘 E。
3. 玩家阻力：玩家动作的被稽查概率/成本修正乘 E（具体钩子由格三在 it_authority 与动作结算路径上实现，保持既有动作语义不变，只改 IT 侧响应强度）。
4. **阳奉阴违与抗命呈现**（S 驱动，E 之外的独立通道）：
   - `S ≥ spPassiveThreshold`（TODO(balance)，初始 0.40）：消极拖延——稽查「查无实据」率上升、上报延迟（观测包中体现为报告缺口；玩家侧阻力下降）。
   - `S ≥ spDefianceThreshold`（TODO(balance)，初始 0.75）：公开叫板征兆——产生可观测事件（抗命、把指令当耳边风、放走嫌疑），并以故事事件呈现（世界观口径：对「上层指令」的抵制，不点名 1 号）；同时登记 `defiance_incident`，进入 1 号观测。
   - 「40 号化」= S 长期高位：该堡叛乱值获得正向漂移加成（TODO(balance)），通牒阶梯在该堡的执行节点出现「空转」（执行层不配合，整肃减员效果按 (1−S) 折减）。

### 6.3 1 号决策层（黑箱壁外）

**状态（每堡一份，存于 `World.Silo1Stewardship.Assignments[siloID]`）**：

- 信念：`belief_effect ± belief_effect_unc`（对「指派能有效驱动执行」的估计）、`belief_disaffection ± belief_disaffection_unc`（对底下自保倾向的估计）。**这些是 1 号自己的猜测，不是真值。**
- 观测缓冲：最近 Δ 月的观测包队列（延迟 Δ = obsDelayMonths，TODO(balance)，初始 2 月）。
- 决策冷却：`decisionCooldownMonths`（TODO(balance)，初始 3 月）。

**观测包 `ObservationBatch`（物理层生成，决策层唯一输入）**：

```
{
  reported_anomalies   int     // IT 上报的异常数（延迟+噪声）
  detected_player_ops  int     // 玩家动作被举报数（延迟+噪声）
  report_gaps          int     // 应上报而未上报的缺口（阳奉阴违迹象）
  defiance_signs       int     // 抗命/怠工征兆计数
  ultimatum_compliance float64 // 通牒后压力回落幅度（观测到的服从效果）
}
```

噪声：物理层在生成观测时用引擎 RNG 加泊松状抖动（固定取样顺序）；延迟：观测滞后 Δ 月。**观测包不得包含 S、E、bump 参数。**

**决策（每堡冷却到期时一次，按 SiloID 升序，走引擎 RNG）**：

候选动作集（风险度）：`hold / step_up_small / step_down_small / step_up_big / step_down_big`（步长 TODO(balance)）。档位指派候选：`raise_posture / keep / lower_posture`。两决策同一回合顺序抽取（先风险后档位），取样顺序固定。

权重因子（全部可归因、写入决策日志）：

| 因子 | 来源 | 对权重的影响 |
|---|---|---|
| 控制需求 | 目标堡 rebellion、order_pressure、incidents（原累加点改登记的不作为/蒙蔽崩塌/偏离事件数） | 推高 step_up 权重 |
| 执行回报信念 | belief_effect（猜的） | 低回报 → 倾向 hold/换档而非加码 |
| 「下一个 40 号」风险 | 观测到的 defiance_signs、report_gaps、belief_disaffection（猜的） | 推高 step_down 权重——**是风险项，不是硬刹车**：控制需求足够强时决策仍可能落在高位 |
| 保留多样性（指标一） | RetentionDiversity | 低多样性 → 压制 step_up_big |
| 寿命预算（指标二） | AvgLifespan 档位 | 低寿命 → 档位候选倾向 lower，风险候选倾向 hold/down（管得少省寿命） |

**日志（决策记录的强制字段）**：`{年月, 目标堡, 选中风险度, 选中档位, 各因子读数, 信念估计±不确定性, 候选权重表, 随机落点}`。玩家侧呈现的「指派依据」即此记录的中文摘要。**日志必须体现「它在猜」**：信念字段以估计±不确定性形式出现，而非查表值。

**默认策略纪律**：保守、慢变、不探针——冷却期内不决策；不确定性高时 hold 权重上浮；任何「为了看反应而故意大幅变动」的逻辑禁止出现。探索策略的正式规格由研究轨另行给出，`stewardship_assignment.go` 必须以可替换接口（`type StewardshipPolicy interface`）封装默认实现。

### 6.4 1 号覆灭后的冻结

`!world.Silo1.Alive`：决策层停摆（不再指派、不再消耗寿命）；物理层按最后指派值再演化一个月后冻结（执行层失去指令来源，S 按 r_crit 以下路径自然回落——「没人找事了，大家跟着 order 走」）。

## 7. 存档升级 v14 → v15

- `CurrentSaveVersion = 15`；新增迁移器 `UpgradeV14ToStewardshipSave`，仅当 `SaveVersion == 14` 时执行，固定写入 `stewardshipSaveVersion = 15`（遵守「升级器必须固定目标版本」约束，[save_upgrade.go L607-610](../../internal/service/save_upgrade.go)）。
- 迁移链连续：v12→v13（谈判）→ v14（渗透）→ v15（权衡层），**前两段迁移器与其测试不得改动语义**。
- 新字段零值语义 = 旧档行为不变：`World.Silo1Stewardship` 零值时调度器早退（等价于未启用），`Silo.SelfPreservation` 零值 = 完全按 order 办事。
- 新增模型字段（models.go，全部带 json tag）：
  - `World.Silo1Stewardship Silo1StewardshipState`
  - `Silo.SelfPreservation float64`（json `self_preservation`）
  - `Silo1StewardshipState{ Experiment, Lifespan, Assignments map[uint]Silo1AssignmentState, Decisions []Silo1DecisionRecord }`（字段详见脚手架代码注释）
- **版本冲突记录**：并行轨 feat/opening-variability（未合入）已在其分支占用 v15（archetypeSaveVersion）。本轨按任务指定取 v15；冲突由集成时裁决（先合入者保留 v15，后者顺延）。

## 8. 与现有实现的关系（改动清单）

### 8.1 改为决策输入的累加点（格三负责，语义替换，测试同步改写）

| 位置 | 现状 | 改为 |
|---|---|---|
| `escape_response.go` 不作为（+escapeResponseIgnoreRS） | 直接加 SafeguardRisk | 登记 `incident:ignore`，不再改风险度 |
| `escape_response.go` 蒙蔽崩塌（+escapeResponseCoverCollapseRS，两处） | 直接加 | 登记 `incident:cover_collapse` |
| `protocol.go recordDeviation`（+rS*periods） | 直接加 | 登记 `incident:deviation`（W 惩罚与偏离记录本身不变） |
| `it_authority.go`（+RSDelta，两处） | 直接加 | 登记 `incident:it_overreach` |
| `content_events.go`/`mechanics.go` 的 `safeguard_risk` delta/set 效果 | 直接加减/覆写 | 保留但语义为「1 号指派的变更通道」（内容事件仍可表达 1 号意志；既有使用点逐个核对，无内容事件实际使用后改为仅由决策层写入） |
| `engine.go` L798 与 core_systems.js VICTORY_CHECK 的 `≥1.0 → DEATH` | 硬失败 | **移除**（见 §5.3） |

### 8.2 需要改写的既有测试

- `batch_validation_test.go`（safeguard race → hard fail）：改为断言高指派 → S 上升 → 执行强度回落（无即死）。
- `escape_response_test.go`（ignore/collapse must raise safeguard risk）：改为断言 incident 登记。
- `it_authority_test.go` L175-182（protective disposal 后 0.16）：改为断言 incident 登记。
- 其余引用 `SafeguardRisk` 的测试按新语义核对（字段仍在，语义为指派值）。

### 8.3 不变项

反应链门槛与阶梯（通牒/整肃/终止/投送）不变；`OrderPressure` 公式不变；单堡路径不变；18 号正典不变；JS 内容事件对 `safeguard_risk` 的**读取**保持可用（值现在是 1 号指派值）。

## 9. 验收（全部实测，无头宿主 + 固定种子 + 真实对局，不接受「应该会」）

| # | 断言 |
|---|---|
| A1 | 某堡风险度长期压在极高值：该堡 IT 服从与稽查显著退化，玩家操作阻力明显低于中等风险情形（三档对比数字） |
| A2 | 该堡出现「40 号化」可观测征兆（抗命/出走/指令当耳边风），且 1 号决策日志可见「下一个 40 号」作为风险项参与权衡——不存在「风险度被自动下调」的机械回路 |
| A3 | 差异指派成立：不同堡拿到不同风险度，各自表现出对应服从水平 |
| A4 | 寿命随档位分档下降；不操作时也自然下降 |
| A5 | 清除地堡后可见多样性损耗；损耗随时间衰减、越接近 Y130M1 清除的损耗越轻 |
| A6 | 黑箱纪律：决策层不读曲线参数与 S 真值（包级依赖检查 + 单测：决策只依赖 ObservationBatch） |
| A7 | 低/中/高三风险区段各跑一次：稽查强度与玩家阻力对比数字证明峰在中段、两端低；长程运行中 1 号带权决策落在峰附近而非贴边 |
| A8 | 同种子双跑逐字节一致；单堡配置逐位不变（既有 parity 锚点全绿） |

实测宿主：仿 [ending_acceptance_harness_test.go](../../internal/service/ending_acceptance_harness_test.go) 模式（`newGatedService` + `threeSiloRequest(seed)` + BeginPassTime/ProcessNextEvent 驱动），固定种子，策略为观测态的确定性函数。

## 10. 并行分工与禁区

五格并行，各自独立 worktree，互不进入对方 worktree、互不代提交；合入 main 仅由主线在自检通过后做一次。

- **格一（指标一）**：`stewardship_experiment.go` + 测试。
- **格二（指标二）**：`stewardship_lifespan.go` + 测试。
- **格三（指标三）**：`safeguard_physics.go`、`stewardship_assignment.go`、§8.1 累加点改造与 §8.2 测试改写。
- **格四（存档）**：`save_upgrade.go` v15 段、`models.go` 版本常量、`app.go` 升级链调用 + 迁移测试。
- **格五（实测与呈现）**：`stewardship_acceptance_test.go`、WorldView 填充消费、`WorldPanel.tsx` 1 号动向节、`frontend/wailsjs/go/models.ts` 增量。

**禁区（并行轨文件，任何格不得触碰）**：`internal/engine/information.go`、`internal/engine/mechanics.go`、`internal/engine/content_events.go`、`internal/engine/authority.go`、`internal/engine/archetype*.go`、`internal/engine/productivity_clamp_test.go`、`events/action_gates.json`、`events/mechanics/common_actions.js`、`frontend/src/components/HeaderStats.tsx`、`frontend/src/components/SetupPanel.tsx`、`internal/service/action_gates.go`、`internal/service/archetype_opening_test.go`、`internal/service/authority_save_test.go`、`internal/service/trait_gates_test.go`。

**共用文件（仅主线/指定格以纯增量方式触碰，冲突在集成时裁决）**：`internal/model/models.go`（脚手架已加字段）、`internal/service/game_service.go`、`internal/service/save_upgrade.go`、`app.go`、`frontend/wailsjs/go/models.ts`。若需修改上述文件的**既有函数语义**，停下回报。

## 11. TODO(balance) 汇总

missPenalty、clearDeadlineMonths、clearPenalty、decayRate、wD/wF、lifespanStart、lifespanNaturalDecay、postureCost{High,Normal,Min}、opCost{Ultimatum,Purge,Termination}、postureDowngradeThreshold、spGrowthRate、spCritRisk、spTimeConstantMonths、bump 四参数（μ/H/σL/σR）、spErosionExp、spPassiveThreshold、spDefianceThreshold、obsDelayMonths、decisionCooldownMonths、风险度步长、观测噪声幅度、S 对叛乱漂移加成率、整肃折减指数。

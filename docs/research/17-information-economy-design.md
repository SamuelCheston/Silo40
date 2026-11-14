# Silo40 《共证》线经济设计：AP、成本与恒定淡忘

> 日期：2026-11-15
> 性质：机制设计与量化核对，不包含剧情内容。
> 范围：《共证》线"玩家推不动"问题的量化、修复与前后对比。
> 上游约束：[the Witness](../story/silo40-witness.md)（干涉者不是指挥者）、[23 节点设计](../story/endings/23-information-nodes.md)、[24 字段级规格](../story/endings/24-information-event-spec.md)、[25 收束段与终局呈现](../story/endings/25-information-endgame.md)、[10 IT 控制可信度](./10-it-control-credibility.md) §3。

## 0. 标注口径

- **【实测】**：无头宿主（`internal/service/info_economy_harness_test.go`，种子 7/8，120 月上限）实测结果。
- **【推导】**：由实测或既有公式推出的设计数值，全部标 `TODO(balance)`，等待 sim 校准覆盖。
- **【裁决】**：用户既定口径，必须遵守。

## 1. 问题（实测）

改动前基线（种子 7/8，三策略）：

| 策略 | 动作数 | 碎片门（旧） | 信念门 | 终局月 | 结局 |
| --- | --- | --- | --- | --- | --- |
| 零操作 | 0 | 未到 | 未到 | 97 | **TIME（赢）** |
| 最低维持 | 9 | 未到 | 未到 | 88/74 | TIME／EXCLUSIONIST |
| 有节奏推进 | 12 | 未到 | 未到 | 94 | TIME |

三个事实：

1. **AP 饥饿**：AP 年恢复 `10 + 威望×0.05`（≈1 点/月），而动作成本 10–30（权威来源是 `common_actions.js` 的 `ap_cost`，Go 侧 `ACTION_COSTS` 只是回退）。开局 50 点花完后，一次分享要空等约 20 个月。有节奏策略 94 个月只做出 12 个动作。
2. **门槛脱锚**：INFORMATION 资格门（`allDepartmentsHoldFragments`）读的是遗留 `KnownFragments` 层；信念层改造后 Share Info 写入的是 `Profession.Beliefs`，遗留层只被被动机制缓慢填充——主动玩法到不了门槛，被动填充（实测中未发生，但 LLM NPC 在场时发生）却可能绕过玩家直达门槛。
3. **零操作获胜**：恐慌→`pro_foreign` 的旧单轴漂移让零操作局在覆灭结算时拿到开放快照，赢下 TIME。这与 13 号思潮轴设计"恐慌推向闭守与排外，不产生开放"直接冲突。

结论：这条线当时是"玩家无事可做地等着赢"，而想推进的玩家反而推不动。

## 2. 修复方向（两条裁决口径的落法）

- **情报存在恒定的淡忘系数，玩家做的事是让流通速度超过淡忘**【裁决】→ 修复 A：部门信念层恒定淡忘；INFORMATION 门重锚到信念层——门槛只在流通跑赢淡忘时为真。
- **玩家是干涉者不是指挥者**【裁决】→ 本次改动全部是速率与可达性参数（恢复率、成本、淡忘系数、门槛深度），没有任何"宣布某条路获胜"的直接通道；胜负仍由事实经仲裁判定。

## 3. 改动清单

### 修复 A：恒定淡忘（`engine.ApplyBeliefForgetting`）

- 部门信念层每个事实每月 `B -= ForgetRateMonthly`（下限 0）。确定性：无随机源，切片按持久化顺序遍历。
- 仅部门层衰减。特工层不衰减（见证者记得）；具名人物层暂不衰减。
- 挂钩点：`EVENT_METRICS_UPDATE` 月度 dispatch，每月恰好一次。
- `ForgetRateMonthly = 0.01`【推导，TODO(balance)】：单次分享吸收 ≈ `B_player·transmit·receptivity` ≈ 0.22–0.34；0.01/月使一次充分吸收（~0.6）约 30 个月跌回分享地板 0.30，维持需求 ≈ 每事实每 30 个月一次再流通。

### 修复 B：INFORMATION 资格门重锚到信念层

- `allDepartmentsHoldFragments`（仲裁侧）与 `victory_check`（`core_systems.js` 结构桥）同改：每部门须持有 ≥ `informationGateFactCount` 个 `B ≥ QualRumor(0.30)` 的事实。两处常量必须同步（注释互指）。
- `informationGateFactCount = 3`【推导，TODO(balance)】：5→3。推导：每事实每部门约需 2 次分享过地板；3 事实 × 9 部门 × 2 次 ≈ 54 次分享 ≈ 162 AP，在生涯 AP 收入（~410）内可达；5 事实（≈270 AP＋维持）不可达。
- 引擎侧把部门信念层（`B>0` 的事实）加入 mechanics payload，供 JS 门读取（确定性：切片序）。

### 修复 C：AP 经济

- `apGainRate`：10 → **36**/年（+威望×0.05 不变）【推导，TODO(balance)】。推导：月收入 ~3 点；目标节奏为每月 1 次流通动作、成本 ~3，即"成本 ≈ 月收入"，玩家每月都有一个真实决策而不是每 20 个月一次。
- 流通三动词成本（`common_actions.js` 权威，`ACTION_COSTS` 回退同步）【推导，TODO(balance)】：
  - Gather：10 → **2**（跨域探查仍按 `ceil(base·(1+0.5·mismatch))` 上浮）
  - Share：20 → **3**（流通主动词，成本 ≈ 月收入）
  - Build Network：15 → **4**
  - 戏剧性动作不动：Incite Rebellion 30、Propaganda 20、Publicize Faction 25——大动作仍要攒。

### 修复 D：恐慌漂移方向

- `IDEOLOGY_UPDATE`：恐慌驱动的 cohort 漂移由 `pro_foreign` 改为 `loyalty`【13 号思潮轴 §0.2 的既定设计】。零操作局不再靠恐慌漂移拿到开放快照。

## 4. 前后对比（实测，种子 7/8）

| 策略 | 改动前 | 改动后 |
| --- | --- | --- |
| 零操作 | 97 月 **TIME 赢** | 97 月 **EXCLUSIONIST（深闭，不赢）**；信念门从未达成 |
| 最低维持 | 88/74 月 TIME／EXCLUSIONIST | 74 月 **EXCLUSIONIST（不赢）**；门槛从未达成 |
| 有节奏推进 | 94 月 TIME，门槛从未达成，仅 12 动作 | **87/88 月信念门达成，INFORMATION 赢**；88 动作，AP 用满月收入 |

改动后的分化：**零操作与最低维持都输（深闭），只有持续流通的玩法赢（共证）**。有意义的操作成为胜负差别。

实测细节（改动后）：

- 有节奏局：信念门在第 87/88 月达成（门槛 3 事实×9 部门），同月仲裁选中 INFORMATION（池：INFORMATION＋EXCLUSIONIST，同分序 INFORMATION 先）。
- 最低维持早于零操作终局（74 < 97）：干预而不推进会加速危机响应（配给行动改变资源/恐慌轨迹）——这是测量结果，不是设计目标；若 sim 校准认为该倒挂需要修正，属后续平衡事项。
- 有节奏局叛乱峰值 0.2，未触发反应链；恐慌峰值 1.0 但恐慌现在漂向忠诚，不再送开放快照。

## 5. TODO(balance) 汇总

| 数值 | 位置 | 推导依据 |
| --- | --- | --- |
| `ForgetRateMonthly = 0.01` | `engine/information.go` | §3 修复 A：吸收量 0.22–0.34 对 0.30 地板的维持周期 ~30 月 |
| `informationGateFactCount = 3` | `service/weighted_endings.go`（JS 门同步硬编码 3 与 0.30） | §3 修复 B：54 次分享 ≈ 162 AP 可达；5 不可达 |
| `apGainRate = 36/年` | `events/mechanics/core_systems.js` | §3 修复 C：月收入 ≈ 流通成本，每月一个决策 |
| Gather 2 / Share 3 / Build 4 | `events/mechanics/common_actions.js`＋`ACTION_COSTS` | 同上；戏剧动作不动 |
| 恐慌→忠诚漂移速率（沿用旧系数） | `core_systems.js` | 只改方向不改速率；速率校准留给思潮轴批次 |

## 6. 明确不做

1. 不新造随机源：淡忘是恒定系数，无掷骰；所有遍历按持久化顺序。
2. 不让玩家直接宣布胜利：四处改动全部是速率/成本/门槛参数，胜负仍由事实经仲裁判定。
3. 不动戏剧性动作成本与既有反应链阈值。
4. 具名人物层信念暂不衰减；暴露水位（E）阈值化等待可读性批次。

## 7. 主要依据

- 无头宿主与三策略驱动：[info_economy_harness_test.go](../../internal/service/info_economy_harness_test.go)（`SILO40_INFO_ECON_DUMP` 门控）。
- 信念层与分享吸收：[10 IT 控制可信度](./10-it-control-credibility.md) §3.3/§3.5/§3.6。
- 结构桥原口径：[07 内容作者手册](./07-content-authoring-guide.md) §12.12（主路径是分享，不是计数）。
- 恐慌不产生开放：[13 思潮轴设计](./13-ideology-axes-design.md) §0.2。

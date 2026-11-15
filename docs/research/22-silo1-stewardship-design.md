# 22 — 1 号地堡权衡层设计（Silo 1 Stewardship）

> 状态：修订版（对齐黑箱学习规格）
> **指标三（风险度指派与自保动力学）的权威规格是 [22 黑箱学习系列](./22-silo1-blackbox-learning/00-index.md)**（00 总览/01 动力学/02 观测/03 探索/04 命名与可读性 + [ADR-0015](../../knowledge/adr/0015-silo1-blackbox-learning.md)）。本卷只保留指标一、指标二与集成事项的权威口径；凡与黑箱系列冲突处，以黑箱系列为准。
> 对齐：[ADR-0017](../../knowledge/adr/0017-safeguard-from-accumulation-to-assignment.md)（safeguard 由后果累加改为 1 号指派基线）。
> 上游裁决演进记录（按时间序）：离心不是负反馈刹车 → 执行强度单峰 → 曲线对 1 号黑箱 → 风险度只驱动隐藏量增速（`self_preservation`，占位命名）→ **黑箱学习规格定稿**（20 箱后验、零 RNG 决策、观测伪装为主防线、参数重算 `g_max=0.12/τ_decay=15/θ_defy=0.70`）。

## 1. 目标与范围

给 1 号地堡加一套「它自己也在做权衡」的指标与决策层：

- **指标一：实验履约与保留多样性（1 号侧）**——本卷 §3 权威。
- **指标二：剩余人员平均寿命**——本卷 §4 权威。
- **指标三：safeguard 风险度改由 1 号指派**——实现口径全部以 [22 黑箱学习系列](./22-silo1-blackbox-learning/00-index.md) 为准（本卷 §5 只记集成关系）。

**非目标（本轮不做）**：唤醒人数机制；探索策略实现（黑箱系列 03 卷为研究轨产物，本轮只落 `ProbeProposer` 接口与 `ProbeCooldown` 计数）；对玩家结局资格的任何新判定。

**硬约束**：

- 与现有 1 号反应链（`order_pressure ≥ 0.45` 且宣传 ≥ 2）是两件事：反应链触发判定不读执行强度 `E`、不读信念 `b`（黑箱系列 00 卷裁决 1）。
- 指标一不判负、不处刑、不进玩家结局资格。
- **功能开关门控**（§2.1）：关闭时全步骤跳过、零 RNG 消耗、与 v8 parity 基线逐位一致；开启时观测产生消耗受控 RNG，parity 基线按 knowledge/08 §5 另行处理。
- 确定性：信念更新与决策全程零 RNG；唯一 RNG 消耗点是观测产生（门控、SiloID 升序、信号类型升序固定取样）。
- 18 号正典链按时间戳强制触发，不读本层任何状态。
- 世界观安全：底下人不知道 1 号存在；玩家可见文本主体是 Order/专线/稽查组/指令（黑箱系列 04 卷 §4 合规清单）。

## 2. 总体架构

### 2.1 功能开关

- `CreateGameRequest.Silo1BlackBox bool`（json `silo1_black_box`）→ 持久化为 `World.Silo1BlackBoxEnabled`。
- 关闭（默认）：黑箱回合、4.55/4.65 步骤、指标一/二月度回合全部早退；`SafeguardRisk` 行为逐位不变；零 RNG 消耗。
- 开启：权衡层整体启用（指标一/二/三）。建档时初始化 `World.Silo1Stewardship`（指标一/二状态）。
- 零值语义：旧档无开关字段 = 关闭 = 逐位不变。

### 2.2 挂载点（黑箱系列 01 卷 §2 权威）

| 事件 | 位置 |
|---|---|
| Θ 结算与 E 生效 | 阶段 1 RunLocalPhase 新步骤 **4.55**（4.5 Protocol 后、4.6 IT 权威前） |
| 观测产生（唯一 RNG 消耗点） | 阶段 1 RunLocalPhase 新步骤 **4.65**（4.6 后、4.7 前） |
| 观测到达、信念更新、r 指派（黑箱回合） | 阶段 2b `runSilo1StateMachine` 既有内容之后，SiloID 升序 |
| 指标一/二月度回合 | 阶段 2b'（黑箱回合之后） |
| RunPostPhase | 不新增 |

r 采用指派月索引：月 t 指派、月 t+1 本地相生效（`LastAssignedRisk`）。`network_jammed` 时黑箱回合早退、在途观测丢弃（00 卷裁决 2）。

### 2.3 状态布局（黑箱系列 01 卷 §8 权威）

- 每堡：`Silo.BlackBox Silo1BlackBoxState`——真值 `Theta` 与信念 `Belief`（N=20 箱）**同堡分字段存放**，读权限隔离由代码纪律与守卫测试保证（决策代码只读信念与观测）。
- 世界：`World.Silo1Stewardship`——指标一（Experiment）、指标二（Lifespan）。
- 引擎侧登记簿：`Silo.SafeguardIncidents`（Go 累加点的事件登记，ADR-0017）。

## 3. 指标一：实验履约与保留多样性（本卷权威）

### 3.1 语义

- **该清除的没清除** → 履约缺口（`FulfillmentDeficit`）恶化。
- **每清除一个地堡** → 多样性损耗（`DiversityLoss`）恶化；该损耗随时间衰减，且越接近实验结束（Y130M1）清除带来的损耗越轻。
- 只影响「最终可保留地堡的多样性」：作为 1 号决策的因子输入与终局审计项；不判负、不处刑、不进玩家结局资格。

### 3.2 事实口径与数值

- **登记（scheduled）**：通牒签发时登记 `(siloID, 月份)`。
- **清除成立（cleared）**：被登记堡 `VictoryStatus.Type == DEATH`。
- **miss**：登记满 `clearDeadlineMonths`（24，TODO(balance)）未覆灭 → `FulfillmentDeficit += missPenalty`（0.10，TODO(balance)），一次性。
- 清除时 `DiversityLoss += clearPenalty × proximity`：`proximity = clamp01(monthsRemainingUntilY130M1 / 96)`，`clearPenalty` 0.15 TODO(balance)。
- 每月两项各乘 `(1 − decayRate)`，`decayRate` 0.02 TODO(balance)。
- 保留多样性读数 `RetentionDiversity = clamp01(alive/initial − 0.5·D − 0.5·F)`（权重 TODO(balance)）。

## 4. 指标二：剩余人员平均寿命（本卷权威）

### 4.1 语义

1 号人员轮流休眠；平均寿命是必然消耗的预算——时间本身使其下降，行动按监控档位额外扣减。耗尽导致能力退化，不是判负。

### 4.2 数值（均 TODO(balance)）

```
初始 20.0 年；自然衰减 0.12 年/月
档位月成本：HIGH −0.10 / NORMAL −0.04 / REPORT_ONLY −0.005
一次性成本：通牒 −0.3 / 整肃链点火 −0.5 / 终止链点火 −0.8（台账防重复）
耗尽退化：≤3.0 → HIGH 降 NORMAL；≤0 → 全强制 REPORT_ONLY、不再签发新通牒
正典豁免：18 号链与时间戳节点不读本指标
```

## 5. 指标三：与黑箱系列的集成关系（细节以系列为准）

- `SafeguardRisk` = 1 号指派基线 `r`（菜单 0.00–0.95）+ 内容事件增量；`≥ 1.0` DEATH 硬线保留（00 卷 §5.1、ADR-0017）。
- Go 累加点（不作为/蒙蔽崩塌/协议偏离/IT 过度干预）改登记 `Silo.SafeguardIncidents`，经观测管线（举报率代理）进入信念。
- Θ 动力学、E(r,Θ) 单峰（峰位左移/峰高下降/峰宽收窄）、20 箱后验、零 RNG argmax 决策、三信号观测与伪装——黑箱系列 01/02 卷权威。
- E 消费：主消费点 `RunITAuthorityMonthly` 效用乘子；次消费点反应链触发后的烈度/窗口调制（00 卷裁决 1）。
- `Attention=Rebellion` 特权直读在功能开启时替换为分档翻转噪声观测（00 卷裁决 6）。

## 6. 存档升级 v14 → v15

- `CurrentSaveVersion = 15`；`UpgradeV14ToBlackBoxSave`（仅 `SaveVersion == 14` 执行；备份 + 单事务 + 回读校验；固定目标版本）。版本与并行轨协调：opening-variability 分支亦声明 v15，按合入顺序裁决，先合入者保留。
- 升级器不回填任何推测状态：新字段零值写入，零值 = 功能未启用 = 旧档逐位不变；`Belief` 先验初始化发生在运行时首次启用（黑箱系列 01 卷 §8）。
- 迁移链连续：v12→v13（谈判）→ v14（渗透）→ v15，前两段语义不动。

## 7. 玩家侧呈现（黑箱系列 04 卷权威）

- 1 号信念决策日志是**审计渠道**（调试端点，`GetLlmDecisions` 先例），不进玩家界面（00 卷裁决 12）。
- 玩家经 Order 渠道感知 r 档位变化（指令措辞/专线杂音，仅跨档传播）、伪装破绽（已发生事实陈述）、既有征兆键的资格衔接（04 卷 §3）。
- 指标一/二同为 1 号内部量：经调试端点与终局审计可读，不进常规 UI（00 卷裁决 7 同口径）。
- 禁止项：不点名结局、不给概率、不教程弹窗、不展示 Θ 与 b、不以「1 号」为主语、不读心反馈（04 卷 §3.d）。

## 8. 验收（全部实测，无头宿主 + 固定种子 + 真实对局）

| # | 断言 |
|---|---|
| A1 | 功能关闭：与 v8 parity 基线逐位一致（零 RNG 消耗） |
| A2 | 极高指派长期压制：该堡 IT 服从/稽查显著退化，玩家阻力低于中等风险（对比数字） |
| A3 | 40 号化可观测征兆（抗命/指令当耳边风）；决策日志体现「下一个 40 号」作为权衡因子；无机械下调回路 |
| A4 | 差异指派：不同堡不同 r 与对应服从水平 |
| A5 | 寿命分档下降 + 不操作也下降 |
| A6 | 清除后可见多样性损耗，随时间衰减、晚期清除损耗更轻 |
| A7 | 低/中/高三区段：稽查强度与玩家阻力峰在中段、两端低；决策落点不贴边 |
| A8 | 黑箱纪律：决策代码零引用物理标识符（AST 守卫）；**1 号估计不得干净跟随 Θ 真值**（观测伪装有效性检查）；决策零 RNG |
| A9 | 同种子双跑逐字节一致（开启态） |

## 9. 分工与禁区

二次扇出（各格独立 worktree、原子提交、合入仅主线一次）：

- **物理层**：常量总表、4.55 Θ/E 结算、E 消费、硬线恢复、G10 表面服从脉冲。
- **观测核心**：4.65 观测产生、伪装与噪声、Attention 替换、network_jammed 丢弃。
- **缺口通道**：G1 稽查机制 / G2 举报管道 / G3 例行指令账本 / G5 人口报告 / G7 注意力预算 / G9 异常观察计数器（02 卷 §6 清单）。
- **黑箱回合**：信念更新、摘要、零 RNG 决策、决策日志（04 卷格式）、ProbeProposer 接口。
- **存档与门控**：v15 迁移器、功能开关、启动链。
- **实测与呈现**：验收 harness、调试端点、Order 渠道叙事核对。

禁区（并行轨文件）不变：`internal/engine/{information,mechanics,content_events,authority,archetype*}.go`、`events/action_gates.json`、`events/mechanics/common_actions.js`、`frontend/src/components/{HeaderStats,SetupPanel}.tsx`、`internal/service/{action_gates,archetype_opening_test,authority_save_test,trait_gates_test}.go`。共用文件（models.go、game_service.go、save_upgrade.go、app.go、frontend/wailsjs/go/models.ts）仅主线/指定格纯增量触碰。

## 10. TODO(balance) 汇总

指标一/二全部数值（§3/§4）；指标三数值以黑箱系列 01 卷 §9 参数总表为准（该表全部 TODO(balance)，集中定义，叙事与 UI 不得写死具体时间描述）。

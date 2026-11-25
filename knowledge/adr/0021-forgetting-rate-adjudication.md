# 架构决策记录（ADR）0021 — 恒定淡忘速率定标 0.005/月

| 状态 | 日期 | 依据 |
| --- | --- | --- |
| 已接受 | 2026-11-26 | 同种子双速率对照实测（main `c61fb25` 单行差分）；裁决指令由项目所有者下达 |

> 裁决正文落在代码（`internal/engine/information.go` 的 `ForgetRateMonthly` 注释）与研究文档
> `17-information-economy-design.md` §3/§5，本篇按 README 约定做补录索引并留证据。

## 背景与问题

淡忘已活在真实月度路径上：`runMonthlyBeliefDispatch` 在事件结算之前，对每个未钉住的事实在每个
部门做乘性衰减 `B ← B×(1−r)`，然后才计 B≥0.70 的连续成熟月与 36 月新鲜度窗口
（钩点必须落在真实生效路径这条口径已由 main 代码与 17 篇 §3 承载，见下文"理由与备选"末段）。

于是 `r` 成了《共证》《公议》两条信念驱动线的一等参数，但仓内同时存在两个值：
- **main（`c61fb25`）**：`ForgetRateMonthly = 0.005`，配套 `DiminishObsCap = 4`、`IndependenceRe = 0.5`，
  并有一条断言"配置值就是校准值"的引擎测试（`forgetting_economy_test.go:92`）。
- **未合分支 `feat/belief-forgetting-live`（tip `5734a3a`，worktree `/home/lnb/silo40-forgetting`）**：
  按 17 篇 §3 的推导取 `r = 0.01`，
  带 `forgetting-empirical-d1` / `forgetting-empirical-d2` 两个实证 worktree 与五个验收重调提交，
  自称全绿但从未合入；main 已领先其分叉点 231 次提交（谈判层、共证外溢投递、archetype 旋钮均在其后落地）。

17 篇 §3 的 `0.008–0.015 取 0.01` 是**初评**推导（每事实 2–3 片、存活 6–12 月即回潮），
与 main 现行共证线经济（27 槽各连续 30 月过地板 + 深填锚 `B₀ = 0.30 + 30r`）不是同一套前提。
悬而未决的问题是：0.005 究竟是"为了让验收变绿而把淡忘调松"的应急值，还是经济上站得住的定标值。

## 决策

**`ForgetRateMonthly = 0.005/月`（保留 `TODO(balance)`）**，采纳 main 现值；`0.01` 落选，
`feat/belief-forgetting-live` 与同轨的实证产物（`feat/forgetting-empirical-d1` /
`feat/forgetting-empirical-d2`、detached worktree `forgetting-baseline-d1`）连同各自 worktree
一并清理，不再把该分支的验收重调搬回 main。配套参数 `DiminishObsCap = 4`、`IndependenceRe = 0.5` 不动。

## 对照方法与结果

两份 worktree 同指 main `c61fb25`，唯一差异为 `information.go` 那一行常量；固定种子不改，
跑五组受淡忘影响的验收（同一条命令在两处各跑一遍）：

```bash
SILO40_ENDING_ACCEPTANCE_DUMP=... SILO40_INFO_ECON_DUMP=... \
go test ./internal/service -count=1 -run 'TestEndingAcceptanceDump|TestInfoEconomyDump|^TestInformation'
# 另加：约章收敛 + 校准线；三线 × bedrock/famine/rift 可达性
```

| 速率 | 通过 | 失败 |
| --- | --- | --- |
| 0.005 | 47 / 47 | 0 |
| 0.01 | 33 / 47 | 13（含子测试节点） |

`0.01` 的失败不是断言过严，是信念经济本身熄火：

- **《共证》**：部门成熟计数全部停在 2 月（`Mines=2 Agricultural=2`，rift 另见
  `Medical=2 Supply=2 Mechanical=2`），够不到 `gate=3`；默认剧本与 bedrock/famine/rift 四条
  playable 线全部落为 `TIME`《无人裁决》——线还在，资格门永远点不着。
- **《公议》**：校准线 daily-only `minW=50.18 / qualifying=0 / shocks=[]`（日常压力通道不再下探），
  决定性场景不点火（`TestCovenantConvergenceDecisiveEventSucceeds` 失败）；
  rift 路线 `facts=0 W=53.96` 约章锁不住，`TestRuleOfLawLinePlayableFromNewGame` 在
  bedrock/rift 两个原型失败。
- **《人治》**：seed 2401 的脚本线仍成立（`PERSONAL_RULE` @97 月，W 谷底 22.7 → 23.9），
  但**同局争仲裁从 `EXCLUSIONIST`(97) 变成 `AGENT_COMPROMISED`(95)**——按
  `weighted_endings.go` 的 P0 硬失败护栏，这是除名级失败：淡忘加快迫使玩家把 AP 花在更贵的
  转述与更密的信道上，组织怀疑度顶穿。
- **引擎侧**：`TestForgettingRateBaselineCalibrationMatchesConfig` 直接 FAIL（0.01 ≠ 校准值 0.005）。

轨迹差异（同一 seed）：公议线 `pact_locked_month` 两值同为 52、W 谷底 7.8 vs 8.7、beats 82 vs 81，
说明**日常冲击与 1 号反应曲线对速率不敏感**；差异集中在信念/事实层——`TestInfoEconomyDump` 里
rhythmic 策略收局时九部门信念之和 seed7 由 15 降到 13、seed8 由 6 降到 4，成熟计数直接掉到门下。
即 `r` 只决定"流通能否跑赢消退"这一件事，而 0.01 下答案是否定的。

复现性核对：裁决落地后在主仓以同一组种子重跑五组，47 项仍全绿，且 `ending-acceptance.json` 与
`info-econ.json` 两份 dump 与基线 worktree 的输出逐字节一致——对照差异只来自那一行速率，不来自执行环境。

## 理由与备选

- **备选 A：采纳 0.01 并按分支节奏重调 main 验收。** 否。分支的五个 `test(service)` 重调提交针对的是
  分叉点之前的机制状态，其后 main 又落地了共证外溢投递、谈判层与 archetype 旋钮；把旧重调搬到新地图
  等于用验收迁就参数。且 0.01 下争仲裁除名是真实经济后果，不是断言口径问题。
- **备选 B：再低（如 0.002）换取更大余量。** 不采纳。0.005 的半衰 ≈139 月（17 篇 §3 与代码注释同口径），
  再低等于关掉淡忘，违背 [ADR-0009](0009-intel-fading.md)「情报会淡忘」与 17 篇 §4 的零成本知情约束。
- **分支的"0.005 让《人治》线发布永不发生、除名"论断不予采信**：在 main 现态下，0.005 的
  `TestBedrockPersonalRuleLinePlayable` / `TestPersonalRuleLinePlayableFromNewGame` / harness 三线全绿。
  该论断属于分叉点之前的经济，其有效部分（淡忘钩点必须落在真实生效路径、消除 JS/Go 双倍口径）
  已由 main 的 `runMonthlyBeliefDispatch` 与 17 篇 §3 承载，无需靠保留 0.01 来保住。
- **保留 `TODO(balance)` 的理由**：0.005 是"三线同时跑通且淡忘仍然有效"的裁决值，不是终局平衡值；
  它与深填锚、`PermeationMonths = 30`、`IndependenceRe = 0.5` 是同一组耦合参数，任何一项重调都必须
  重跑本对照。

## 后果与取舍

- 17 篇 §3 保留"初评 0.01、回评后下调 0.005"的叙述原样不动；本篇把那次"回评"补成一次有数据的正式裁决。
- 0.01 时代的分支验收（含分支自己的 ADR 编号 0014→0017 改号尝试）随分支作废；main 的
  `0017-no-backward-compat-before-release.md` 与其无关，仓内不留悬空的淡忘 ADR 引用（已核）。
- 已知弱断言记账：`TestEndingAcceptanceDump` 对争仲裁线只断言"结算出任意非空结局"，因此它在
  0.01 下仍然 PASS——本篇的判定不以它为准，实际把关的是 `covenant` / `lines` / `info` 三组与
  引擎校准断言。收紧该断言属测试质量范畴，不在本次裁决内改动。
- 删除分支与 worktree 不丢可复现证据：本 ADR 记录了 base 提交、差分位置与重跑命令。

## 例外

- 具名剧情事实的 `FactState.Pinned` 豁免淡忘（不逐月乘性衰减），本裁决不触碰该豁免；
  18 号地堡事件链等按时间戳权威覆写的口径不变。
- 本裁决只定恒定淡忘速率，不启动 17 篇 §4 的有向淡忘/反宣传通道（仍为设计，未落地）。

## 关联

- 代码：[information.go](../../internal/engine/information.go) `ForgetRateMonthly`、
  [mechanics.go](../../internal/engine/mechanics.go) `runMonthlyBeliefDispatch` 淡忘钩点
- 文档：[17 信息经济设计](../../docs/research/17-information-economy-design.md) §3 恒定淡忘 / §5 可调参数
- 测试：`internal/service/info_endgame_test.go`、`covenant_convergence_test.go`、`calibration_test.go`、
  `rule_of_law_chain_test.go`、`personal_rule_chain_test.go`、`archetype_*_test.go`；
  `internal/engine/forgetting_economy_test.go`、`internal/engine/permeation_test.go`
- 相关 ADR：[0009 情报会淡忘](0009-intel-fading.md)、
  [0014 反制代价只按组织规模抬怀疑度](0014-countermeasures-cost-suspicion-org-only.md)、
  [0020 上线前消耗品不承担即时重调义务](0020-pre-release-consumables-no-retune-obligation.md)

## 备选与作废内容

按本篇裁决作废的 0.01 轨产物（清理时逐个核对过工作区均为干净、无未提交内容）：

- `feat/belief-forgetting-live`（tip `5734a3a`，11 次提交未合，worktree `/home/lnb/silo40-forgetting`）：
  `r=0.01` 的推导与裁决叙述、四条 archetype 旋钮、五个验收重调、分支自己的淡忘 ADR（改号 0014→0017）。
- `feat/forgetting-empirical-d1`（tip `fc58bd3`，2 次提交未合）与
  `feat/forgetting-empirical-d2`（tip `968c68f`，4 次提交未合）：0.01 经济下的无干预消退与
  流通墙实测探针（worktree `/home/lnb/forgetting-empirical-d1` / `-d2`）。
- detached worktree `/home/lnb/forgetting-baseline-d1`（`d02b866`，该提交已在 main 历史内，纯指针）。

不保留副本；其仍有效的部分（淡忘钩点必须落在真实月度路径、消除 JS/Go 双倍扣减口径）已由 main 的
`runMonthlyBeliefDispatch`、`forgetting_economy_test.go` 与 17 篇 §3 承载。
`fix/forgetting-current`（`4dcf6a8`，已合入 main）不属本轨，未作清理。

# 25 — v15 硬切断与派生威望的测试欠账

> 状态：待收拾（快速扩充期，agents.md §四：编译通过即提交，测试问题留档）。
> 来源：C4「威望全层派生」（24 篇实现方案的第 4 个原子提交）。
> 日期：2026-11-18。

C4 落地后 `go build ./...`、`go vet ./...`、`go test ./internal/engine/` 全绿；
`go test ./internal/service/` 有 10 个失败，全部落在两个**预期内**的类别，
不是实现缺陷。本文记录现象与处理思路，留待日后统一收拾。

## 一、存档升级链测试（6 个，硬切断的直接后果）

失败清单：

- TestUpgradeV7ToAuthoritySaveReadbackAndResume
- TestUpgradeV9ToPreparationSave
- TestUpgradeV10ToOutsideSave
- TestUpgradeV11ToLineShiftsSave
- TestUpgradeV12ToNegotiationSave
- TestUpgradeV13ToPermeationSave

现象：升级链把旧档一路升到 v14，随后 resume 被
`game_service.go` 的版本闸拒绝：「存档=14，当前要求=15」。

原因：C4 按 ADR-0017 把 CurrentSaveVersion 14→15 硬切断，**不写 v14→v15
迁移**。升级链的终点 v14 现在永远无法入局——整条 v1→v14 升级机器
（save_upgrade.go）与其测试实质上已是死代码。

处理思路（二选一，倾向前者）：

1. **整体删除**升级链与这 6 个测试（ADR-0017：上线前无兼容义务）。
   顺带清理 save_upgrade.go 的 isV1Save…isV13Save 与备份逻辑。
   注意 permeationSaveVersion 等版本常量若仍被读档侧引用需一并核对。
2. 保留升级链做「考古」，把测试的 resume 断言改为**断言硬切断拒绝**
   （v14 档必须被拒），升级步骤本身只验 DB 状态。

## 二、种子调优可玩性测试（4 个，动力学漂移）

失败清单与现象：

| 测试 | 种子 | 现象 |
| --- | --- | --- |
| TestRebellionLinePlayableFromNewGame | 2402 | 逃离群体被追歼 (demise=pursuit)，结算落 EXCLUSIONIST 而非 REBELLION |
| TestEscapePreparationGradientNotReversed | 2401 | full/sparse 均未出发 (it=block)，126-08 落 FRACTURE |
| TestInformationEndingPlayableFromNewGame | — | share 接触被拒：「该部门没有回应你的接触」 |
| TestRuleOfLawLinePlayableFromNewGame | — | charter_lock_day 未触发，约章窗口从未打开，落 EXCLUSIONIST |

原因（C4 的三处值变化，RNG 抽取次数未变，共享随机流未平移）：

1. **代表野心输入轴换轨**：generateResidentAmbition 从 cohort 威望改读
   PowerLevel（min(PL,10)/10×0.40）。高权部门代表野心 +0.16~0.40，
   潜在领袖 (A≥0.62 && I≥0.25) 提前数年涌现，派系聚类/忠诚度动力学
   整体改道。engine 侧 TestPoliticalGateHysteresisPreventsFlipFlop 已
   因此重锚（fixture PowerLevel 8→4，避开领袖旁路）。
2. **平民玩家威望塌缩**：commoner 开局 A=0.1/I=0.1 → prestige 10
   （旧制 connections 重算可达 60+）。INCITE 门 (prestige≥60) 与
   PUBLICIZE_FACTION 门 (≥80) 在纯时间演化下不可达
   （influence 年增 0.01，50 年才 +0.5），必须靠事件抬升——这是设计
   后果（24 篇已接受），但旧种子策略没有这类事件动作。
3. **年度结算改期初威望口径**：AP/政治点数按期初 prestige 结算，
   影响力演化下一期生效。量级极小（每年 ≤0.25 prestige 差），
   但长局累积足以改变加权随机决策的落点。

处理思路：

- 用既有验收 harness 重调种子：escape 线
  `SILO40_ESCAPE_ACCEPTANCE_DUMP=1 go test ./internal/service/ -run
  TestEscapeAcceptance`（sweep-<seed> 批量），information/rule_of_law
  线用 ending acceptance harness 同款口径。
- **建议等 C7（事件驱动聚类）与 C9（加入决策）落地后再重调**：
  C7 会废除 isPotentialLeader 硬门、改派系生成机制，现在重调的种子
  到 C7 大概率再漂一次，双重返工不值。
- 若重调后 commoner 线仍打不开 INCITE/PUBLICIZE 门，考虑在内容侧
  补「野心/影响力抬升」事件（24 篇 C6 的 FOUND_FACTION 本身即此类
  动作的入口），而不是回退派生公式。

## 三、已随 C4 完成的测试重锚（无需再动）

- 结局/门控 fixture：PoliticalPrestige=100 → Ambition=1/Influence=1；
  =60 → 0.6/0.6；=0 → 0/0（action_gates、weighted_endings、各 chain 测试）。
- harness 快照 prestige 读法 → engine.DerivedPrestige。
- agent_stats 展示口径测试 → 派生公式（A0.8/I0.6 → 70，AP 39.5）。
- 新增 prestige_derive_test.go：四层同构 + prestige→influence 折算。
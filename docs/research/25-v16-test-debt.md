# 25 — v16 硬切断与派生威望的测试欠账

> 状态：待收拾（快速扩充期，agents.md §四：编译通过即提交，测试问题留档）。
> 来源：C4「威望全层派生」（24 篇实现方案的第 4 个原子提交）及其与
> 22/23 黑箱系列的合并。
> 日期：2026-11-18。

C4 落地后 `go build ./...`、`go vet ./...`、`go test ./internal/engine/` 全绿；
`go test ./internal/service/` 有 10 个失败，全部落在两个**预期内**的类别，
不是实现缺陷。本文记录现象与处理思路，留待日后统一收拾。

**版本顺延**：C4 原拟 CurrentSaveVersion 15 硬切断；合并 origin/main 时
发现 22/23 黑箱系列已占 v15（含 UpgradeV14ToBlackBoxSave 升级步）。裁决：
黑箱 v15 语义与升级链保留不动，威望派生硬切断顺延为 **v16**，拒载 v15
及更早存档。下文失败现象按合并前 (要求=15) 记录；合并后升级链终点 v15
同样被 v16 闸拒绝，黑箱侧的 resume 类测试（TestUpgradeV14ToBlackBoxSave
及 v12→v15 迁移链连续性测试等）并入同一欠账类别。
TestResumeRejectsWrongVersion 原硬编码「当前要求=15」，已改读
model.CurrentSaveVersion 动态断言（拒绝机制本身与版本无关）。

## 一、存档升级链测试（硬切断的直接后果）

合并前失败清单（6 个）：

- TestUpgradeV7ToAuthoritySaveReadbackAndResume
- TestUpgradeV9ToPreparationSave
- TestUpgradeV10ToOutsideSave
- TestUpgradeV11ToLineShiftsSave
- TestUpgradeV12ToNegotiationSave
- TestUpgradeV13ToPermeationSave

现象：升级链把旧档一路升到 v14，随后 resume 被
`game_service.go` 的版本闸拒绝：「存档=14，当前要求=15」。
合并后黑箱接力步把档升到 v15，仍被拒：「存档=15，当前要求=16」。

原因：C4 按 ADR-0017 硬切断，**不写威望层迁移**。升级链的终点现在
永远无法入局——整条 v1→v15 升级机器（save_upgrade.go）与其测试
实质上已是死代码。

处理思路（二选一，倾向前者）：

1. **整体删除**升级链与这些测试（ADR-0017：上线前无兼容义务）。
   顺带清理 save_upgrade.go 的 isV1Save…isV14Save 与备份逻辑。
   注意 permeationSaveVersion / blackBoxSaveVersion 等版本常量
   若仍被读档侧引用需一并核对。
2. 保留升级链做「考古」，把测试的 resume 断言改为**断言硬切断拒绝**
   （终点档必须被拒），升级步骤本身只验 DB 状态。

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

## 四、C6 发起候选的动力学漂移（新增 3 个，2026-11-18）

C6（FOUND_FACTION 动作 + NPC 发起候选）落地后，种子调优可玩性失败
从 4 个增至 7 个。新增清单：

| 测试 | 现象 |
| --- | --- |
| TestExclusionistEndingPlayableFromNewGame | 零输入路线 125-02 落 RESET 而非 EXCLUSIONIST |
| TestDeathPhysicalStrikePlayableFromNewGame | 事件链时序漂移，deceive 路线未按预期推进 |
| TestTerminationChannelCountermeasurePlayableFromNewGame | 干扰信道后真实叛乱未达自测阈值 (deceives=0) |

原因：NPC 加权决策新增 FOUND_FACTION 候选（weight = A + prestige/100 +
强度），改变共享随机流的落点；发起成功的零成员派系进入忠诚度动力学与
结局仲裁遍历（零影响力计零分，但 openFactions 计数含 pro_foreign:High
发起派系）。C6 单测与端到端执行测试全绿，失败均为长局动力学漂移，
与第二节同类。

处理：并入第二节的种子重调批次，**等 C7/C9 落地后一次完成**。
C7 废除自动聚类后发起成为派系唯一来源，届时漂移会再发生一次，
现在重调仍是双重返工。

## 五、C7 事件化落地后的账目更新（2026-11-18）

C7（废除每 tick 聚类、MaintainFactions 维护管线、筹备期解散、死亡接任）
落地后全量复测：**13 个失败 = 8 升级链 + 5 可玩性漂移**。

**版本顺延 v16 → v17**：v16 存档里的聚类派系与归属是已废除算法的产物，
语义不可延续，按 ADR-0017 拒载。升级链 8 个测试的拒绝信息变为
「存档=15，当前要求=17」，类别与第一节完全相同，不另列。

### 随 C7 修复的真实回归（不是欠账，已闭合）

1. **TestLegitimacySurvivesRealMonthlyTick**：探针派系 fixture 在 C7 下
   不合法（无领袖/无 founding 戳；组织统计改由成员 cohort 派生，注入
   Influence/Cohesion 不再生效）。重锚为零成员筹备期发起派系
   （PLAYER 代表 + founding 戳），totalInf=0 → 叛乱保持 0 → 正统性留存。
2. **TestSilo18NamedCharactersFate / TestSilo18ChainSurvivesResume**：
   MaintainFactions 的代表标志修复段原先对**所有**居民重建派生标签，
   覆写了 18 号正典具名人物的 silo18NamedTag / silo18CanonTrait，
   导致 namedResident / silo18Canon 找不到人、链条整体停摆。
   修复：不挂 cohort 的个体（正典具名人物）标签由链条节点权威维护，
   维护管线跳过（旧 RebuildImplicitFactions 本就只重建挂 cohort 的居民）。

### 可玩性漂移（5 个，并入第二节重调批次）

| 测试 | 状态 |
| --- | --- |
| TestRebellionLinePlayableFromNewGame | 持续（第二节） |
| TestEscapePreparationGradientNotReversed | 持续（第二节） |
| TestInformationEndingPlayableFromNewGame | 持续（第二节） |
| TestRuleOfLawLinePlayableFromNewGame | 持续（第二节） |
| TestCovenantConvergenceDecisiveEventSucceeds | 新增：decisive 发布成功 (publishW=52.6) 但 large_scale_evidence −24 冲击落点改道，约章未锁 (pact=0.001) |

C6 新增的 3 个漂移（Exclusionist / DeathPhysicalStrike /
TerminationChannel）在 C7 的随机流重排下**恢复通过**——漂移对 RNG
落点高度敏感，进一步说明现在重调种子仍是返工。

原因：聚类掷骰整体移除 + NPC 决策人口扩到全部存活 cohort 代表
（C7 前仅现任派系代表），共享随机流大幅平移；零成员发起派系进入
忠诚动力学与结局仲裁。C7 引擎审计测试全绿（零发起零派系、筹备期
解散、接任确定性、无单月归属往返、双跑一致）。

处理：**等 C9（加入/退出）落地后一次重调**，届时成员流动机制定型，
种子策略才有稳定靶子。

## 六、C8 感知/口碑落地后的账目更新（2026-11-18）

C8（三通道感知 + faction:<id>:founded 信念事实 + 月度口碑 + 信息动作
搭载）落地后全量复测：**13 个失败 = 8 升级链 + 5 可玩性漂移**，清单与
第五节完全相同，无新增、无恢复。

新增掷骰点全部门控：发起感知写入零 RNG；口碑与搭载只在存在政治派系
时掷骰（零派系零掷骰），玩家搭载确定性不消耗 RNG。因此**首次发起之前
的共享流与 C7 逐位一致**；漂移只发生在 NPC/玩家发起成功之后的长局。
C8 引擎测试全绿（三通道单调、发起零 RNG、faction: 永不成熟/不计共证门、
首跳折减 0.6、多跳传播、玩家搭载零 RNG、JS marker 链路、同种子双跑
逐字节一致），service 端到端发起测试通过。

### 可玩性漂移（5 个，持续，并入第二节重调批次）

| 测试 | 状态 |
| --- | --- |
| TestRebellionLinePlayableFromNewGame | 持续（第二节） |
| TestEscapePreparationGradientNotReversed | 持续（第二节） |
| TestInformationEndingPlayableFromNewGame | 持续（第二节） |
| TestRuleOfLawLinePlayableFromNewGame | 持续（第二节） |
| TestCovenantConvergenceDecisiveEventSucceeds | 持续：现象同第五节（decisive 发布成功 publishW=52.69，large_scale_evidence −24 冲击改道，pact=0.000），落点数值随 C8 流平移微漂 |

原因：发起成功后月度口碑按 (派系, 源, 目标) 升序掷骰、NPC/居民信息
动作搭载掷骰进入共享流，长局落点平移；faction: 事实可见于 mechanic ctx
但永不成熟，共证门与 INFORMATION 结局闸不受污染。C7 恢复通过的 3 个
（Exclusionist / DeathPhysicalStrike / TerminationChannel）在 C8 下仍
通过。

处理：仍**等 C9（加入/退出）落地后一次重调**。C9 的加入掷骰读
belief.B ≥ AwarenessJoinFloor(0.30)，成员流动定型后种子才有稳定靶子。

## 七、C9/C10 落地后的账目更新（2026-11-19）

C9（LeaderRelations 子矩阵 + JOIN_FACTION + NPC 月度加权加入/退出）与
C10（MERGE_FACTION）落地后全量复测：**仍为 13 个失败 = 8 升级链 + 5 可玩性
漂移**，与第五节完全一致，**无新增可玩性漂移**。月度加入/退出扫描与 NPC
合并候选虽平移了共享 RNG 流，但未把任何此前通过的测试推入失败——第五节
的 5 个漂移测试维持原现象，升级链 8 个维持 v17 硬切断拒绝。

### 随 C9/C10 修复的真实回归（不是欠账，已闭合）

**TestActionGatesStartNarrowAndUnlockFromStoryStage /
TestOpeningRequiresSuccessfulActionsNotIdleMonths**：C9/C10 新增的
action_join_faction / action_merge_faction 门控初版 `available:{}`（恒可用）
使两动作在开局即暴露，破坏「开局只暴露 Gather Info」不变量。修复：新增
门控指标 `political_faction_count`（政治派系数量，非无阵营桶），两动作门控
改 `{metric: political_faction_count, min: 1}` + `hide_when_unavailable:true`
——开局零政治派系则隐藏，发起派系后才暴露。语义诚实：加入/合并是派系存在
后的政治动作。修复后两测试恢复通过。

处理：第五节 5 个可玩性漂移的种子重调批次靶子已随 C9/C10 定型（成员流动
机制完整：发起/加入/退出/合并/接任/解散），可择期一次重调；升级链 8 个按
ADR-0017 整体删除或改断言硬切断拒绝（第一节两选项），不在本阶段处理。

## 八、C8+C9/C10 合并态 main 的最终账目（2026-11-19）

C8 与 C9/C10 分别复测时各自维持第五节的 13 个失败清单，但**合并态**
（月度管线顺序：忠诚动力学 → 口碑传播 → 加入/退出；两分支掷骰点叠加）
再次平移共享流，失败构成发生变化，总数不变：

**13 个失败 = 8 升级链（不变）+ 5 可玩性漂移（构成更新）**

| 测试 | 合并态状态 |
| --- | --- |
| TestRebellionLinePlayableFromNewGame | 持续漂移（第二节） |
| TestEscapePreparationGradientNotReversed | 持续漂移（第二节） |
| TestInformationEndingPlayableFromNewGame | 持续漂移（第二节） |
| TestRuleOfLawLinePlayableFromNewGame | 持续漂移（第二节） |
| TestDeathPhysicalStrikePlayableFromNewGame | **新漂移**：C7/C8/C9 单独跑均恢复通过，合并态下物理打击后的远程终结落点改道（bombing_terminal 129-10 已触发但 remote termination 未落地），脚本时序与动力学错位，属种子调优漂移而非代码回归 |
| TestCovenantConvergenceDecisiveEventSucceeds | **恢复通过**：pact=0.936、decisiveFired=true（第五/六/七节的 −24 冲击改道现象在合并态落点下消失） |

Exclusionist / TerminationChannel 维持通过。升级链 8 个维持 v17 硬切断
拒绝（链尾 v15，"存档=15，当前要求=17"）。

原因：合并把 C8 口碑掷骰与 C9/C10 加入/退出/合并掷骰叠进同一月度流，
相对顺序（口碑先于加入）与任一单分支都不同；长局落点再次平移。
LeaderRelations 感知接线是纯函数零 RNG，不移动流。

处理：**种子重调批次靶子以此合并态为准**（5 个漂移 = 上表前 5 行），
机制已完整定型，重调一次覆盖；升级链 8 个仍按第一节两选项处理。

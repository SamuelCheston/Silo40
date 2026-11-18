# 26 — INFORMATION 路线脚本层策略重设计

> 状态：研究完成，待所有者裁决后实施。
> 来源：25 篇 §九/§十 留档的唯一可玩性欠账
> （TestInformationEndingPlayableFromNewGame，120 种子零命中）。
> 日期：2026-11-19。所有者指令「先研究脚本层策略重设计」。
> 纪律：本文只动脚本层（测试策略），不动机制/数值/断言。
> 按 ADR-0020，本次研究属所有者明确要求，非自动重调。

## 一、结论摘要

种子 2401 实测终局：**9 个部门中 7 个已满足 3 成熟事实，仅 IT 与
Judicial 各缺 1 槽**。缺口不是 AP 耗尽（终局部门 AP 满 100），而是
三个脚本层病理叠加：

1. **只发布了 1 条核心事实**（Judicial_2，123-03）；Judicial_1 从未
   进入公共证据——每部门只有 1 个「免费粘性槽」，其余 2 槽全靠流通。
2. **分享上限永久锁死断流槽**：`shareAttempts[dept:fact] ≥ 4` 是终身
   计数；信念淡忘跌破地板、渗透计数归零后，该对仍被排除，无法重播。
   IT:Judicial_1（B=0.105, counter=0）与 Judicial:Judicial_3
   （B=0.244, counter=0）正是这样死掉的。
3. **成本探针把监视拒绝当致命错误**：T3-2 联系人监视（3 月窗口）期间
   `resolveActionCost` 返回「该部门没有回应你的接触」，脚本
   `shareCostFor`/`buildCostFor` 直接 `t.Fatalf`——120 种子中 33 个
   中途死亡纯属此 bug，不是策略失败。

替代事实 Judicial_4 在 ~88 月才补播（终局 counter=5/8），30 月渗透窗
早已来不及。**路线输在日历，不是输在资源总量。**

## 二、机制约束清单（研究核实）

### 2.1 资格门与日历

- 门 = **每个部门 ≥3 条成熟事实**（`informationGateFactCount=3`，
  `allDepartmentsHoldFragments`，weighted_endings.go:358/370）。
  9 部门 × 3 = **27 槽**。
- 成熟 = B 连续 **30 个月** ≥ 0.30（`PermeationMonths=30`，
  `ForgetRateMonthly=0.005`）。成熟粘性，不回退。
- 深填目标 B₀ = 0.30 + 30×0.005 = **0.45**：填到 0.45 恰好滑行 30 月
  到地板，无需补流通。
- 全局日历：Y122M1 开局 → **Y130M1 Silo 1 坠落强制结算 = 96 月**。
  第 M 月播种的事实第 M+30 月成熟 ⇒ **所有 27 槽必须在第 66 月前
  播种并托住**。这是硬截止。

### 2.2 AP 经济与玩家属性

- 玩家 Judicial 开局 **A=0.05 / I=0.05**（initializer.go:233）→
  派生威望 = 50×(0.05+0.05) = **5**（C4 全层派生；旧制 connections
  重算可达 60+）。
- AP 年恢复 = 36 + prestige×0.05 = **36.25/年 ≈ 3.02/月**
  （core_systems.js:57）。开局 AP 50。96 月生涯 ≈ 50 + 8×36.25 ≈ **340 AP**。
- 影响力被动增长 (0.005+0.05×0.05)=0.0075/年，8 年仅 +0.06；
  但**事件可抬 A/I**（2401 终局玩家 A=0.405/I=0.528，威望 ~47）——
  前期弱、后期强，管道建设窗口恰好在最弱的前期。
- 建联增量 = 0.05 + prestige×0.005 = **0.075/次**（旧制 ~0.35）。
  从开局关系 0.15–0.25 建到 0.5 需 **4–5 次 = 16–20 AP/部门**。
  T1-3 冷却期再减半（0.0375）。
- 动作成本：Share 3 / Build 4 / Gather 2（+规范冲突上浮）；
  verify 15 / publish 20 / investigate_dispatch 15。
  T2-3：目标戒备 ≥0.50 → Share ×2 = 6。

### 2.3 吸收公式

- 分享吸收 = B_player × transmit × receptivity，
  transmit = 0.5+0.5×connection，receptivity = 0.5+0.5×pro_foreign
  （common_actions.js:142–146）。
- 前期 connection≈0.25 / pro_foreign≈0.4 → transmit 0.625 /
  receptivity 0.7 → B_player 0.8 时单次吸收 ≈ **0.35**；
  同源再分享 IndependenceRe=0.5、递减 1/(1+0.25n) cap 4 →
  第二次 ≈0.17、第三次 ≈0.11。**2–3 次同月集中分享可达 0.45–0.63。**
- pro_foreign 以 0.35/年向低目标回落，只有持续流通托得住；
  派系忠诚动力学（C7）改道思潮演化 ⇒ 吸收强度逐种子漂移。
- 发布吸收 = `beliefObserve("departments", fact, 0.85)` +
  coverage 0.9（publish_evidence.js:31–32）——**全部门一次性深播**，
  实测 Judicial_2 发布后各部门 B≈0.46、counter=30 成熟，零维护。
  这是唯一能一次填满 9 部门同一槽的动作。

### 2.4 反制（只抬成本/收窄路径，不判负）

- **每次玩家分享**向 IT/Judicial 写 `leak:<fact>` 观察，值=吸收量
  （common_actions.js:170–183）。持续流通必然推高泄漏确信。
- T1-3：部门 leak B ≥0.45 → 该部门关系增长减半 12 月（可重复触发）。
- T3-2：部门 leak B ≥0.60 → **接触类动作 3 月不可用** + 关系 −0.10 +
  怀疑 +0.03，**每部门终身一次**（countermeasures.go:81–99）。
  到期后永久可接触。实测终局 IT/Judicial leak:Judicial_1/3/4 ≈0.78，
  戒备 0.51/0.58（Share 已 ×2）。
- 戒备月回落 0.015；失败探查 +0.10。高戒备部门探查更贵更易失败。

### 2.5 现成但未被脚本使用的工具

- **`investigate_dispatch`**（15 AP，冷却 1 月）：max(B1,B2)≥0.60 即可，
  确定性深挖较弱链 `beliefObserve(player, target, 0.9)`。
  **把 Judicial_1 抬到发布地板的现成手段，脚本完全没用。**
- `bailiff_dispatch_evidence_a`：first_local + rebellion≥0.15 时
  确定性触发，玩家 Judicial_1 B=0.95。触发时机依赖叛乱爬坡，
  发布窗口（14 月）可能错过——investigate_dispatch 不依赖它。
- `verify_evidence`（15 AP，B≥0.85）/ `publish_evidence`（20 AP，
  B≥0.90+已核验，冷却 2 月）：脚本已用，但只喂饱了一条链。

## 三、为什么合并态下系统性失败

C4 威望派生把 Judicial 玩家前期管道能力砍到 1/4.7（建联 0.35→0.075），
路线在 C4 时就已结构性吃紧（25 篇 §二首录「share 接触被拒」）。
派系事件化（C7–C10）的共享流平移改变了事件时序与思潮演化：

- evidence_a 触发时机漂移 → Judicial_1 发布窗口错过概率上升；
- pro_foreign 演化改道 → 吸收强度逐种子漂移，浅播断流更频繁；
- NPC 决策人口扩大 → 叛乱/恐慌轨迹改变（实测终局恐慌 0.8–0.97），
  lie-low（rebellion≥0.7）月数与探查失败率漂移。

结构吃紧 + 落点漂移 ⇒ 120 种子零命中。**换种子已穷尽合理区间
（25 篇 §九），修复必须在策略层。**

## 四、重设计方案（脚本层，零机制改动，断言原样）

### A. 探针鲁棒性（必做，直接回收 33/120 中途死亡）

`shareCostFor`/`buildCostFor` 先查 `engine.ContactUnderWatch(silo, dept)`，
被监视则返回「本月跳过该部门」（+∞ 成本或显式 skip），不 Fatalf。
`attemptCirculation` 对「该部门没有回应你的接触」错误同样跳过。
T3-2 每部门终身一次、3 月到期——跳过即自愈，不是路线断裂。

### B. 双事实发布（核心杠杆：27 槽中 18 槽变免费）

发布阶段改为：等 Judicial_2 到发布地板后，**用 investigate_dispatch
把较弱链（通常 Judicial_1）深挖到 ≥0.90**，verify 两条、publish 两条
（尊重冷却 1/2 月）。每部门获得 2 个深播粘性槽，流通只需补 **1 槽/部门**。

- AP：investigate ×2–4（30–60）+ verify ×2（30）+ publish ×2（40）
  ≈ 100–130；发布阶段拉长到 ~20–28 月，收入 ~60–85 + 开局 50，可行。
- 日历：流通 ~28 月开始，第三槽截止 66 月 ⇒ **38 月窗口补 9 槽**，
  每槽 ~25 AP（建联 16 + 分享 6–9）× 9 ≈ 225 AP，
  收入 38×3 ≈ 114 + 发布阶段余量 ⇒ 需配合 D/E 省 AP，见预算核算 §五。
- 副作用：第二次发布再冲击法警忠诚 −0.25 / IT −0.15 / W −5 /
  Judicial+IT 恐慌 +0.25。恐慌已 0.8+，边际影响待实测；
  叛乱若因此更频繁 ≥0.7，lie-low 月数上升——方案 F 的日历重构
  必须把 lie-low 计入截止核算。

### C. 断流感知的分享上限（直接修复 IT/Judicial 缺口）

`shareAttempts` 终身计数改为**断流重置**：当部门对该事实 B < 地板
且 counter=0（流已断），重置该对计数，允许重播。上限仍约束
「连续无效吸收」（B 在地板上方但吸收不动），不再约束「断流后重新投资」。
重播优先级高于补播第四事实——断流槽是已投资日历的回收，
第四事实是从零开始的新 30 月窗。

### D. 深填集中（把每槽从「2–4 次散播」压到「2–3 次同月集中」）

当前脚本每月每部门最多播 1 次然后轮转。改为：对选定部门+事实，
**同月连续分享直到 B ≥ 0.45**（深填目标）或 AP 不足，再换部门。
吸收 0.35+0.17+0.11 ≈ 0.63 ⇒ 2–3 次到 0.45+，滑行 30 月零维护。
避免浅播（B 0.30–0.35）→ 10 月内断流 → 补流通/重播的 AP 浪费。

### E. 监视者排序（IT/Judicial 特殊处理）

IT/Judicial 既是目标又是泄漏观察者：对它们的流通推高自身 leak B，
触发 T3-2 封锁 + T1-3 建联减半。策略：

- **早期优先填 IT/Judicial 的第三槽**（leak B 尚在 0.45 以下时），
  此时建联未减半、Share 未 ×2、无封锁。
- 其余 7 部门随后。封锁触发后跳过 3 月（方案 A），到期后永久畅通。
- 避免在高戒备（≥0.50）部门反复探查（失败 +0.10 戒备，恶性循环）；
  玩家自身 Judicial 事实用**本部门 gather**（FProfMatch 1.25、
  ProbeFitOwn 0.6、无规范冲突）深化，比跨部门 rotation 便宜且稳。

### F. 日历重构（播种截止 66 月，之后滑行）

- 阶段 1（0–~28 月）：开局 → 法警链 → 双发布。
- 阶段 2（~28–60 月）：9 部门第三槽深填（E 排序，D 集中）。
  **第 60 月后不再新播种**（60+30=90 ≤ 96，留 6 月容错）。
- 阶段 3（60–96 月）：只维护断流风险槽（upkeep）+ 链条节点
  （replication/receiver/living_chain 已在此前触发）。
  删除无用的 rotation gather 兜底（组合只需 Judicial_1/2/3/4，
  其他部门碎片对门无贡献）。
- lie-low 月（rebellion≥0.7）计入截止：若阶段 2 有效月不足，
  优先保 IT/Judicial 与低 receptivity 部门（Mayor pro_foreign 0.33）。

## 五、AP/日历预算核算（保守口径）

| 项 | AP | 月 |
| --- | --- | --- |
| 开局 gather+share | 5 | 0–1 |
| 法警链等待（实测 ~6–12） | 0（被动） | 1–12 |
| investigate ×3 + verify ×2 + publish ×2 | 115 | 12–28（含冷却） |
| 建联 9 部门 ×4 次（前期 0.075/次） | 144 | 28–60 |
| 第三槽深填 9 ×2–3 次 Share（IT/Judicial ×2 价） | ~70 | 28–60 |
| 本部门 gather 深化 Judicial_3/4 ×4 | 8 | 穿插 |
| upkeep（断流风险槽，~6 次） | 18 | 60–96 |
| **合计** | **~360** | 96 |

生涯收入 ≈ 340（§2.2）+ 事件抬威望后半年收入上浮（终局威望 47 ⇒
38.35/年）≈ **350–360**。预算平衡但紧——省 AP 的关键在 B（18 槽免费）
与 D（零维护深填）。若实测仍紧，杠杆排序：
减少建联次数（0.4 够用，transmit 0.7）> 减少 investigate 次数 >
延后非关键部门播种。**不动机制数值。**

## 六、验证计划

1. 实施 A–F 后先跑**原种子 2401、原断言**（不削弱、不白盒）。
2. 复用 `info_economy_harness`（SILO40_INFO_ECON_DUMP）口径加测：
   逐部门成熟计数曲线、发布月、流通起始月、封锁/lie-low 月数、
   AP 轨迹。临时 env 门控诊断，验毕删除（25 篇 §九同款纪律）。
3. 2401 通过后小范围确认（≤20 种子）非孤例；**不做 120 种子大扫描**
   （ADR-0020：本次是策略重设计验证，不是消耗品重调）。
4. 通过则更新 25 篇 §十（欠账闭合）+ 本文状态；不通过则留档停止，
   回报所有者裁决下一步（可能触及机制层，超出本文范围）。

## 七、风险与开放问题

- **第二次发布的冲击**是否把叛乱推过 0.7 更久（lie-low 侵蚀阶段 2
  有效月）？需实测。若显著，可把 Judicial_1 发布延到第三槽播种过半
  之后（发布只需在 66−30=36 月前？不——发布槽也要 30 月窗，
  最晚 66 月发布；延后发布压缩流通窗口，需权衡）。
- **Mayor receptivity 0.667**（pro_foreign 0.334）：第三槽吸收
  0.8×0.7×0.667≈0.37，2 次到 0.55，可行但最脆；E 排序中 Mayor
  应紧随 IT/Judicial 之后。
- evidence_a 是否触发不再关键（investigate_dispatch 兜底），
  但若它早触发可省 1–2 次 investigate（15–30 AP）。
- 本方案不改 `progressResetCountermeasureForRoute`（终止链反制
  仍按原 helper 应答）；其分享也写 leak，计入反制预算。
- 断言中的链条节点（replication≥4、receiver、living_chain、
  public_fact_cross_dept）在 2401 失败局已全部触发——重设计不触碰
  这些路径，只补成熟度缺口。

## 八、参考

- 25 篇 §九（120 种子失败形态）/§十（唯一欠账）
- 17-information-economy §3/§5（AP 预算与门深推导）
- 18-time-permeation §3/§7（渗透窗、深填滑行、公平重调）
- 20-world-countermeasures §3.2/§5.1/§6（leak 观察、T1-3/T3-2、上界纪律）
- ADR-0012（反制只读公开事实）/ ADR-0014（怀疑只由组织行为驱动）
- ADR-0020（消耗品纪律；本次研究属所有者明确要求）
- 代码锚点：weighted_endings.go:358/370、information.go:130–183/533–574、
  countermeasures.go:57–119、common_actions.js:120–242、
  core_systems.js:49–59、initializer.go:204–235、
  publish_evidence.js / verify_evidence.js / investigate_dispatch.js /
  bailiff_dispatch_evidence_a.js、info_endgame_test.go:249–605
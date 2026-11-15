# Silo40 路线图

> **对账口径**（2026-11-15，基准 main `ffdecb7`）：每条按代码与设计的实际状态重新判定——**已完成**给出实现文件与位置，**部分完成**逐项列出已落与未落，**过期**判定直接改对。设计文档的自述状态不作为依据，以代码为准。

## 待做

### 世界的反制剩余档（原「追踪的反制入口」）

**状态：部分完成。** 「反制藏进日常小动作」批次的主体已落地，剩五个档。设计与测试账见 [20-world-countermeasures-design.md](research/20-world-countermeasures-design.md) §4/§7，长期约束见 [ADR-0012](../knowledge/adr/0012-world-countermeasures-read-public-facts-only.md)，筹备侧口径见 [ADR-0014](../knowledge/adr/0014-countermeasures-cost-suspicion-org-only.md)。

已落地（代码核实）：

- **T1-2 建联摩擦 / T2-3 交接收紧**：[game_service.go](../internal/service/game_service.go) `resolveActionCost`——目标部门戒备 ≥0.30 时 Build 成本 4→5，≥0.50 时 Share 成本翻倍 3→6，只贵不堵、效果不变；配套 [engine.go](../internal/engine/engine.go) `DecayProfessionAlertness` 部门戒备月度冷却 0.015/月，避免 IT 反击把戒备永久顶满。
- **死亡线远程终止渠道绑定**：值夜检修动作 `TERMINATION_CHANNEL_JAM`（[termination_channel_jam.js](../events/player_actions/termination_channel_jam.js)）→ [termination_channel_disrupted.js](../events/special/termination_channel_disrupted.js) 写 `network_jammed`，[termination_release.js](../events/crisis/termination_release.js) 读该守卫（只免疫远程渠道，物理投送照旧）。
- **本地整肃反制**：`PURGE_ROSTER_TAMPER` → `silo1_purge_softened` 取轻档；信息侧 `PURGE_PLOT_LEAK` → `silo1_purge_exposed` 按中档，与轻档并存时按轻档。
- **逃跑组织侧 IT 响应层**：[escape_response.go](../internal/engine/escape_response.go)——组织怀疑度（只由组织行为驱动）≥0.35 且冷却 4 月已过，带权随机三选一：镇压/蒙住 1 号/不作为，后果只抬成本与风险，囤积账本永不扣减。

剩余四档（设计 §4 均标【项目外推】，逐项核实确未实现；T2-2 已随《共证》外溢投递侧落地）：

| 档 | 反制 | 形态要点（起点值均 TODO(balance)） |
| --- | --- | --- |
| T1-3 | 关系冷却 | 部门对泄漏事实 `B ≥ 0.45` → 该部门关系增长减半、持续 12 月；Build 回执措辞变化 |
| ~~T2-2~~ | ~~信道审计~~ | **已落地**（[spillover.go](../internal/service/spillover.go)）：外溢信道使用次数到档（专线类 3 次/信号类 5 次/实体类 8 次）→ 可靠度降档，跌破地板中断 |
| T2-4 | 谈判可信度侵蚀 | `prep_exposure ≥ 0.4` → 保障可信度 `Guarantee −0.15`，经既有 `addTopStanceGuarantee` 接口（[top_stance.go](../internal/engine/top_stance.go)）→ 顶层立场权重自然偏向阻挠。注意：谈判层 IT 的 `undermine` 立场（[negotiation.go](../internal/engine/negotiation.go)）已会侵蚀保障可信度，但那是磋商铺位，不是本档由筹备暴露驱动的通道 |
| T3-1b | 身份线「日常小动作」绑定 | 复核期＝地点类动作次月灰置；复查期＝职务动作当月灰置；抹除准备期＝秘密行动类动作灰置（`available: suspicion < 0.7`）。灰置机制既有；身份线四节点（T3-1）与 1 号反应链（T3-3）是既有档，不在剩余清单 |
| T3-2 | 联系人被盯上 | 部门对泄漏事实 `B ≥ 0.60` → 匿名联系人被调离/问话：关系 −0.10、接触类动作 3 月不可用（cd 保护）＋月度故事。只动匿名联系人，不碰正典具名人物 |

**批次纪律**：设计 §7.2 判据 6 要求反制不得单独落地——五档必须与 §7 测试账重调**同批合并**，不允许「先落反制、测试红了再说」的中间态进 main。

### 《共证》外溢投递侧

**状态：准备侧与投递侧均已落地。** 裁决口径保持：**外溢不进胜负判定——外溢为零同样可以判胜，外溢数量只影响结算分数**——仲裁已按此实现：[weighted_endings.go](../internal/service/weighted_endings.go) `spilloverSiloCount` 与 `spilloverSignalCount` 只进《共证》加权与结算行，不进资格。

已落地（准备侧，随《越岭》Layer 1）：

- 外溢准备账本：[models.go](../internal/model/models.go) `SpilloverPreparation`（`Channels` 信道清单＋`PackageQuality` 包裹质量）；
- 三种信道接触卡与建立动作：[preparation.go](../internal/engine/preparation.go) `applySpilloverChannelAdd`（`it_line` 专线/`cross_signal` 跨堡信号/`physical_handoff` 实体交接），玩家动作 [escape_spillover_it_line.js](../events/player_actions/escape_spillover_it_line.js)/[escape_spillover_cross_signal.js](../events/player_actions/escape_spillover_cross_signal.js)/[escape_spillover_physical.js](../events/player_actions/escape_spillover_physical.js)（建立即计组织怀疑度）；
- 信道可靠度与谈判联动：`NEGOTIATION_HANDS_OFF`（不接管市长职位）提升既有信道可靠度（[negotiation.go](../internal/service/negotiation.go)）。

已落地（投递侧，[spillover.go](../internal/service/spillover.go)，16 §3.5 Layer 3）：

- **受限渠道投递**：三种信道各自的投递因果——能带多少（专线/实体整包、信号只带 0.6）、多长时间到（专线 1 月/信号 2 月/实体 6 月）、路上损耗什么（信号衰减/实体磨损），可靠度按种子确定性抽取，成功经 `World.CrossQueue`（到期月, Seq 纪律）投递，失败记路上丢失；发送侧账本 `SpilloverPrep.Deliveries` 留审计痕迹；
- **包裹组装**：I-3 五种能力各一张封装卡（事实与来源/独立核验/反制方法/隐匿与交接/再传递，各 +0.2 包裹质量，`chain_living` 阶段门控），补齐了 `package_quality` 此前无生产者的缺口；
- **外部接收方独立核验与再复制**：包裹到达时目标堡按自身模拟状态（碎片基础、开放程度、IT 控制强度）带权裁量是否独立接收；接收后向尚未到达过的邻堡再复制一代（质量逐代衰减、无回声、代次上限），全程复用既有多堡模拟与跨堡派发阶段，不另造影子状态、不修改接收堡正典状态；
- **外溢暴露与宿主风险**：使用即付——组织怀疑度（通信规模，ADR-0014）＋1 号反谍轨道（专线异常流量）；T2-2 信道审计作为首个落点随本批落地（使用次数到档可靠度降档、跌破地板中断）。风险全部落在成本与选择空间，不没收、不判负；
- **口径收紧**：触达计数对包裹记录收紧为「到达且独立接收」（链按 `RootSilo` 归属），通用跨堡信号维持 G7 到达口径逐位不变；结算行在确有投递时追加包裹明细（尝试/丢失/独立接收/代次）。

剩余缺口（见 [KNOWN_GAPS.md](../KNOWN_GAPS.md)）：实体交接与逃出群体的顺带携带耦合（再传递任务随出发结算）未实现；`Silo1State.Attention` 被月度叛乱值覆写、无法承载累积型暴露，本期以 1 号反谍轨道与组织怀疑度替代。投递侧机制设计见 [16-preparation-outcome-design.md](research/16-preparation-outcome-design.md) §3.5。

### LLM 上下文档位可调

**状态：已实装。** 四档与四个开关沿参与度的同一条配置路径落地（`SILO40_LLM_CONTEXT` 等环境变量、Wails `SetLlmContext`、`POST /debug/game/llm/context`，切换记入决策日志 `kind=context_config`、读档恢复）；界面在开局设置区（SetupPanel「LLM 顾问上下文」）。口径落点：个人记忆 = 该 NPC 的特质＋已知碎片＋人脉；本堡主线经过 = 本堡 history 类事件；silo 设定／其他地堡关系／长期剧情线 = 服务层按既有事实组装的段落（设定仅陈述已确立事实，地堡关系给拓扑与谣言档状态，剧情线只给各结局线已点亮的里程碑）；其他地堡情报 = 谣言档迷雾视图（只给档位不给精确值，默认关，是 §3.4 提示词白名单的唯一用户自选例外）。默认 standard＋个人记忆开，与历史 payload 形状一致；LLM 关闭或确定性 NPC 时零代码路径变化。事件摘要默认条数 5 为 `TODO(balance)`。

允许用户调整喂给模型的上下文档位，与已有的 LLM 参与度档位正交：参与度决定“什么时候问”，上下文档位决定“问的时候给多少料”。

| 档位 | 上下文内容 |
| --- | --- |
| 极简 | 仅当前 NPC、候选动作、输出契约 |
| 标准 | 在极简基础上，加地堡大局指标、相关派系人物、近期事件摘要 |
| 丰富 | 在标准基础上，再加本堡主线经过 |
| 世界观 | 在丰富基础上，再带 silo 设定、其他地堡关系、长期剧情线 |

档位之外保留独立开关或配置项：个人记忆、条文法规、事件摘要条数、是否携带其他地堡情报。

动机：预算紧张的用户可压到很小的上下文，预算宽裕的用户可放开上下文，在成本与决策信息丰富度之间自行取舍。

### 22 号 · Silo 1 黑箱学习

**状态：仅设计，代码未实现。** 五卷系列 [22-silo1-blackbox-learning](research/22-silo1-blackbox-learning/00-index.md) 与 [ADR-0015](../knowledge/adr/0015-silo1-blackbox-learning.md) 已定稿：风险度驱动隐藏量自保（`self_preservation`）、执行强度单峰、1 号分箱后验信念、观测噪声与伪装、探索代价与决策日志；裁决 13「反应快暴露慢」已重算定稿（g_max=0.12/τ=15/θ_defy=0.70），主防线押观测伪装（q_fc=0.75）。代码中无黑箱/后验信念/分箱对应实现。1 号侧独立，不碰玩家侧结算。

### 23 号 · 底下 IT 盲区经营

**状态：仅设计，代码未实现。** 六卷系列 [23-it-blindspot-cultivation](research/23-it-blindspot-cultivation/00-index.md) 与 [ADR-0016](../knowledge/adr/0016-it-blindspot-cultivation.md) 已定稿：监控设备/报表回执六通道手段谱系、带权重随机决策模型与审计日志、盲区厚度与被经营偏置的累积与慢累积拆穿。**依赖 22 号**：被经营偏置必须注入 22 的观测管线（23-05 卷接口规格），须待 22 号落地后排期。

### 开局可变性剩余（19 号）

**状态：部分完成。** 设计见 [19-opening-variability-design.md](research/19-opening-variability-design.md)。

- 已落地：门控型特质「地堡土著」与「隐秘行事」——[initializer.go](../internal/engine/initializer.go) 特质赋值，「隐秘行事」经 [mechanics.go](../internal/engine/mechanics.go) `trait_stealth=0.8` 影响检测。
- 未实现：驻地三档原型（磐石/裂隙/荒年）、号数→原型的确定性映射与配套初始资源/部门构成差异。
- 纪律：上一版开局可变性原型因未先算测试账被回退——开局偏向必须与可玩路线测试重调同时落地，见 [ADR-0011](../knowledge/adr/0011-opening-bias-needs-test-retuning.md)。本地工作分支 `feat/opening-variability-current`（worktree `silo40-opening-variability`）在进行中、未合入 main，接手前先与该会话对齐。

### 见证者成就树（ADR-0004）

**状态：已立项，未实现。** [ADR-0004](../knowledge/adr/0004-witness-achievement-tree.md)：国策树即成就系统——点亮节点的是可核验的成就（驻地里真实发生、别人也能确认的事实），冲突节点二选一永久锁死，本轮不做奖励，失败结局不上树，成就与事件链完成键一一对应，树不裁决结局。仓库中无任何成就树/节点树的前后端实现。依赖面只有既有的结局资格事实，不碰仲裁与结算。

## 已完成（原 roadmap 标为待做，本次对账核实）

### IT 幕后威势、Order 与 Judicial 三方附着——七个子项全部实现

原 roadmap 判定本节「设计完成，尚未实现」，**已过期**：七个子项逐项核实全部落地。设计见 [10-it-control-credibility](research/10-it-control-credibility.md)，落地步骤与总验收见 [15-roadmap-and-acceptance.md](research/10-it-control-credibility/15-roadmap-and-acceptance.md)。

1. **威势与 Judicial 骨架** ✅ [authority.go](../internal/engine/authority.go)：`ShadowAuthority`/`JudicialAttachment`（W、IT/SWING/PACT 三态附着迁移、Mayor 低/高 W 回摆、两档开门 `AuthorityConfig`、`PactLocked` 锁定）；状态随存档持久化（[models.go](../internal/model/models.go) `Authority` 字段）。
2. **碎片与检测** ✅ [information.go](../internal/engine/information.go) `ObserveFact`（连续 `B/π`、观察去重）；检测分类 `UNSAMPLED/SAMPLED/DIRECT_ORDER`（[mechanics.go](../internal/engine/mechanics.go) `suspicionDelta`）；矿工结构性不采样由 [detection_classes_test.go](../internal/engine/detection_classes_test.go) 锁定；AP 与怀疑独立。
3. **分层曝光与公开证据** ✅ [information.go](../internal/engine/information.go) `ComputeInformationEnvironment`（部门/具名人物信念聚合 `E`、`PublicEvidenceQ`、合格独立事实）；两份独立法警公开证据链 [bailiff_dispatch_evidence_a.js](../events/special/bailiff_dispatch_evidence_a.js)/[bailiff_dispatch_evidence_b.js](../events/special/bailiff_dispatch_evidence_b.js)；内容事件写入路径 [content_events.go](../internal/engine/content_events.go)。
4. **Order 与 Silo 1 认知** ✅ [protocol.go](../internal/engine/protocol.go)：版本化条款加载、believed/true 双副本（`BelievedVersion`/`BelievedClauses`）、三类偏离 violation/lapse/unmet、介入分 J 与确定性介入链 warning→correction→override→assessment（稀有、确定、可预警、无随机）。
5. **二元联盟与隐蔽维护** ✅ [it_authority.go](../internal/engine/it_authority.go)：`ItActionCatalog`（§5.1 维护＋§5.3 七类隐蔽动作＋反击）、`itUtility` 双动机 utility、`R_op` 预算与资源枯竭反击衰减；仅接入 Judicial 法警与 IT 内部，`ItAuthorityState` 持久化。
6. **玩家拆台路线** ✅ 玩家动作 [investigate_dispatch.js](../events/player_actions/investigate_dispatch.js)/[expose_execution_failure.js](../events/player_actions/expose_execution_failure.js)/[verify_evidence.js](../events/player_actions/verify_evidence.js)/[publish_evidence.js](../events/player_actions/publish_evidence.js)，门控见 [action_gates.json](../events/action_gates.json)；IT 反击走 it_authority 反击目录。
7. **法治结局与批量验证** ✅ [weighted_endings.go](../internal/service/weighted_endings.go)：`RULE_OF_LAW` 以 `PactLocked` 为资格条件进入候选池、仍由四类主体加权；无头宿主固定种子验收已跑通（含与《人治》同局竞争仲裁）。

### 18 号 · 时间渗透 ✅

[18-time-permeation-design.md](research/18-time-permeation-design.md)：部门信念层渗透成熟度（`FactBelief.PermeationCounter`/`Matured`，B 连续过地板满渗透月数成熟），INFORMATION 资格门、victory_check 与守夜前兆改读成熟计数；存档升 v14（[save_upgrade.go](../internal/service/save_upgrade.go) `UpgradeV13ToPermeationSave`）。恒定淡忘 [information.go](../internal/engine/information.go) `ApplyBeliefForgetting` 已随月度派发启用。

### 21 号 · 谈判与妥协 ✅

[21-negotiation-compromise-design.md](research/21-negotiation-compromise-design.md)：市长/叛乱方/IT 三方带权随机立场、磋商重估、最后一刻妥协与保障可信度（`Guarantee`）；存档升 v13（[save_upgrade.go](../internal/service/save_upgrade.go) `UpgradeV12ToNegotiationSave`）。实现见 [negotiation.go（引擎）](../internal/engine/negotiation.go) 与 [negotiation.go（服务）](../internal/service/negotiation.go)；「谈判改变的是阻碍不是胜利」见 [ADR-0013](../knowledge/adr/0013-negotiation-changes-blocking-not-victory.md)。

### 《越岭》逃离模型主体（Layer 0/1/2）✅

已落地：顶层立场（市长/IT 三档，带权随机决策，地堡种子复现）、准备账本（防护服/补给/材料/预案功能位覆盖/演练/照护/生产起点/接应档位/暴露，划拨即扣减）、《越岭》事件链四节点（意愿→路线→开窗→离开，见 [36](story/endings/36-rebellion-nodes.md)/[37](story/endings/37-rebellion-event-spec.md) 分册）、出发结算级联（容量 D→存活率 S→维存 M→追踪 T→逃出人数，立场只塑形输入，阻挠自然结算为人数 0），以及 Layer 2 的堡外段：堡外月度维持（补给/防护服/路线三侧面各自结算，覆没写明冻死/耗尽/被困）、追踪四档（没人注意/已被察觉/咬住痕迹/正在追捕，由筹备暴露＋出发清点＋safeguard 风险＋1 号注意力定档，改变选择空间而非必然失败）、群体质量（老人/伤者/工匠/领头人组合事实，作玩家贡献加权项，资格仍只读群体存续）。仲裁仍只读 `EscapedGroup`，不读立场与账本（[ADR-0008](../knowledge/adr/0008-escape-gate-to-outcome-layer.md)）。追踪的反制入口剩余档见上文「世界的反制剩余档」。

**移除登记（设计者裁决）**：原待做两项——「逃出群体与原驻地的后续关系（回传、沉默或原驻地毁灭后的独立存续）」与「散逃者尾声」——已移除，不再排期。理由：**逃出发生时原堡已被摧毁，不存在可延续的后续关系；没赶上队伍的人必死，没有尾声可写。** 口径落点见 [16 §3.2 裁决 3](research/16-preparation-outcome-design.md)、[36 §5／§8](story/endings/36-rebellion-nodes.md)、[37 §3](story/endings/37-rebellion-event-spec.md) 与《越岭》结算叙事。

## 下一批可跑（按独立性与依赖排序）

| 序 | 批次 | 独立性 / 依赖 | 备注 |
| --- | --- | --- | --- |
| 1 | LLM 上下文档位 ✅ | 完全独立 | 已实装：四档 + 四开关沿参与度同一路径落地，见上文状态行 |
| 2 | 见证者成就树（ADR-0004） | 独立 | 只读既有结局资格事实与完成键，不碰仲裁；逃离/外溢未落部分只预留节点 |
| 3 | 22 号黑箱学习 | 独立（1 号侧） | 设计定稿、参数已重算；不碰玩家侧，唯一新增确定性状态在 1 号侧 |
| 4 | 19 号开局可变性剩余 | 独立但带测试账 | 必须与可玩路线测试重调同时落地（ADR-0011）；本地工作分支在进行中，接手前先对齐 |
| 5 | 世界的反制剩余档（T1-3/T2-4/T3-1b/T3-2） | 独立但带测试账 | 四档须与 20 §7 测试账重调同批合并（§7.2 判据 6），接口全部既有；T2-2 信道审计已随《共证》外溢投递侧落地 |
| 6 | 23 号盲区经营 | **依赖 22 号** | 被经营偏置注入 22 观测管线（23-05 卷），排在 22 号落地之后 |

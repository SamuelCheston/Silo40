# 《越岭》事件链 · 字段级事件规格

> [返回总索引](../silo40-endings.md) · [节点设计](./36-rebellion-nodes.md) · [作者手册](../../research/07-content-authoring-guide.md) · [准备度换算设计](../../research/16-preparation-outcome-design.md)
> 性质：**实装规格**。沿用 [34《无人裁决》](./34-time-event-spec.md)／[31《人治》](./31-personal-rule-event-spec.md) 的格式与纪律：数值一律 `TODO(balance)`；依赖未落地机制的节点只留注释；功能位、无人名。

## 0. 通用约定

- **目录与命名**：里程碑与时刻节点入 `events/special/`，玩家动作入 `events/player_actions/`，纯调度与反制入 `events/crisis/`。文件名＝事件 `id`，snake_case，本线前缀 `escape_*`／`route_*`／`advance_party_*`／`departure_*`。
- **完成键**：事件节点为 `special:<id>` 或 `crisis:<id>`；玩家动作为 `player_actions:<事件 id>`。一个键只记事实。
- **布线事实**（与 27/31/34 号相同）：完成键发总线事件；`event_triggered` 参与月度持续评估；**订阅既有职务动作的节点放 `special/`**（`route_survey_step` 订阅机械／矿业职务动作，不拦截宿主）。
- **本批新增的封闭集扩展**（16 §6.2，随本线落地）：
  - 效果 `top_stance_reeval`（flag=mayor|it|both）：带权随机重估立场，随机源走引擎单一 RNG 流（地堡种子）；
  - 效果 `top_stance_guarantee`（value=增量）：保障可信度，夹 [0,1]；
  - 效果 `preparation_delta`（metric=白名单字段）：账本写入，未知字段＝加载失败；`plan_role` 经 profession 携带功能位；`reception_stage` 经 flag 携带档位且只前进；
  - 效果 `preparation_transfer`（metric=Supplies|Materials，value=量）：划拨即扣减，余额不足＝运行期显式失败（不部分划拨）；
  - 效果 `spillover_channel_add`（flag=种类，count=目标堡，value=可靠度）：信道登记（建立侧）；
  - 效果 `escape_departure_settle`：出发结算点请求，服务层在全部效果施加后执行；
  - 触发器 `preparation_gte`（metric=账本字段，value=阈值）：供下游门控。
  - JS helper：`topStanceReeval/topStanceGuarantee/preparationDelta/preparationSetBool/preparationStage/preparationPlanRole/preparationTransfer/spilloverChannelAdd/preparationGte/escapeDepartureSettle`。
- **ctx 可读**：`top_stance`（mayor/it/guarantee，归一后）、`escape_prep`（账本全字段只读视图）、`spillover_prep`（信道列表与包裹质量）、既有 `silo_metrics/authority/resources/states` 等。立场与账本只供内容塑形动作可得性与结算输入，**永不作交互选项、永不进仲裁**。
- **存档**：v9 → v10（`UpgradeV9ToPreparationSave`，备份＋单事务＋回读校验）；新字段零值即旧档语义（无准备、立场零值＝阻挠）。

## 1. 节点规格

### W-1 departure_willingness（意愿）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/special/departure_willingness.js` |
| 触发条件 | `all(timestampAtOrAfter(123,1), professionMetricGteCount("panic_value", 0.5, 3))`（相对 Y1 起；`TODO(balance)`）；`fire_mode: "ONCE"` |
| 效果 | `siloMetricDelta("rebellion", +0.03)`；`siloMetricDelta("history_burden", +0.02)`（`TODO(balance)`） |
| 完成键 | `special:departure_willingness` |
| DSL 能力 | 全部现有 |
| 依赖 | 无 |

### W-2 route_survey_offer / accept / decline / defer（路线接触）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/player_actions/route_survey_{offer,accept,decline,defer}.js` |
| 动作规格 | `ROUTE_SURVEY_{OFFER,ACCEPT,DECLINE,DEFER}`、`PLAYER_EVENT`、`target_type: "NONE"`、`unavailable_behavior: "hide"`；`ap_cost: 1/0/0/0`（`TODO(balance)`） |
| 触发条件 | 各自 `playerActionIs(...)`；`fire_mode: "ONCE"` |
| 效果 | accept：`rebellion +0.01`；其余纯接触（`TODO(balance)`） |
| 完成键 | `player_actions:route_survey_*` |
| 门控 | offer：`all(stage first_signal, relation Mechanical 0.2, event_completed special:departure_willingness)`；三选择：`event_completed player_actions:route_survey_offer` |
| DSL 能力 | 全部现有 |
| 依赖 | 无（兄弟选择隐藏是已知缺口：门控无否定谓词） |

### W-2 route_survey_step（勘察步骤）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/special/route_survey_step.js`（订阅职务动作 → special/） |
| 触发条件 | `all(eventTriggered("player_actions","route_survey_accept"), any(playerActionIs(MECHANICAL_OVERHAUL/MECHANICAL_PIPE_TAP/MINES_DEEP_EXCAVATION/MINES_TUNNEL_NETWORK)))`；`fire_mode: "ONCE"` |
| 效果 | `preparationStage("surveyed")`；`rebellion +0.02`（`TODO(balance)`） |
| 完成键 | `special:route_survey_step` |
| DSL 能力 | 全部现有＋本批 `preparation_delta`（stage 分支） |
| 依赖 | 无 |

### W-2 route_confirmed / advance_party_dispatch / advance_party_report（先遣链）

| 字段 | 规格 |
| --- | --- |
| `route_confirmed` | `events/special/route_confirmed.js`；触发 `eventTriggered("special","route_survey_step")`；效果 `rebellion +0.04`（`TODO(balance)`）；ONCE |
| `advance_party_dispatch` | `events/player_actions/advance_party_dispatch.js`；`ADVANCE_PARTY_DISPATCH`、`ap_cost: 2`（`TODO(balance)`）；效果 `scheduleEvent("advance_party_report", 4)`；门控 `event_completed special:route_confirmed` |
| `advance_party_report` | `events/crisis/advance_party_report.js`；纯调度（`canTrigger: false`，由 dispatch 延迟点火）；`script.apply` 守卫读 `states["crisis:lower_deck_collapse"].triggered`：受损 → `panic_value +0.05`、`rebellion +0.01`（档位停 surveyed）；完好 → `preparationStage("confirmed")`、`rebellion +0.04`（`TODO(balance)`） |
| 完成键 | `special:route_confirmed`、`player_actions:advance_party_dispatch`、`crisis:advance_party_report` |
| DSL 能力 | 全部现有＋本批 `preparation_delta` |
| 依赖 | 无（延迟档允许溢出；先遣回报不是终局事实节点，无需覆灭兜底） |

### W-3 escape_window_open（开窗）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/special/escape_window_open.js` |
| 触发条件 | `all(siloMetricGte("rebellion", 0.72), eventTriggered("special","route_confirmed"), eventTriggered("special","underground_coordination"))`（`TODO(balance)`）；`fire_mode: "ONCE"` |
| 效果 | `topStanceReeval("both")`；`rebellion +0.05`；`professionMetricDeltaAll("panic_value", +0.03)`（`TODO(balance)`） |
| 完成键 | `special:escape_window_open` |
| DSL 能力 | 全部现有＋本批 `top_stance_reeval` |
| 依赖 | 无 |

### W-3 escape_window_orderly / escape_window_riot（R-D 抉择）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/player_actions/escape_window_{orderly,riot}.js` |
| 动作规格 | `ESCAPE_WINDOW_{ORDERLY,RIOT}`、`PLAYER_EVENT`、`ap_cost: 1`（`TODO(balance)`） |
| 效果 | orderly：`rebellion +0.02`、`history_burden +0.02`；riot：`rebellion +0.10`、`floorStabilityDelta("", -0.08)`（全部楼层）、`panic_value +0.06`（`TODO(balance)`） |
| 结算消费 | 出发结算读两键定秩序系数：orderly=1.0、riot=0.7、未选=0.85（`TODO(balance)`） |
| 完成键 | `player_actions:escape_window_{orderly,riot}` |
| 门控 | `event_completed special:escape_window_open` |
| 依赖 | 兄弟选择隐藏是已知缺口（门控无否定谓词） |

### W-4 escape_departure_launch（出发＝结算点）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/player_actions/escape_departure_launch.js` |
| 动作规格 | `ESCAPE_DEPARTURE_LAUNCH`、`PLAYER_EVENT`、`ap_cost: 3`（`TODO(balance)`） |
| 触发条件 | `playerActionIs("ESCAPE_DEPARTURE_LAUNCH")`；`fire_mode: "ONCE"` |
| 效果（按序） | `topStanceReeval("both")`（终局重估）→ `escapeDepartureSettle()`（服务层结算请求） |
| 结算实现 | `internal/service/escape_settlement.go`：D=min(路线,组织,防护服) → N → S（服×路×练×序）→ M（补给/照护/生产/组织）→ T（暴露折减）→ 写 `EscapedGroup`（Population/Departed/Surviving/Fragile）。全部阈值 `TODO(balance)`；M4 延迟定性、追踪四档、堡外月度维持**属 Layer 2，注释预留** |
| 完成键 | `player_actions:escape_departure_launch`（出发尝试事实） |
| 门控 | `all(event_completed special:escape_window_open, any(event_completed player_actions:escape_window_orderly, event_completed player_actions:escape_window_riot))` |
| 依赖 | Layer 2（堡外维持／追踪四档／群体质量加权）未落地前，结算为最小形态 |

### 立场生产者 escape_negotiation_guarantee（保障磋商）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/player_actions/escape_negotiation_guarantee.js` |
| 动作规格 | `ESCAPE_NEGOTIATION_GUARANTEE`、`PLAYER_EVENT`、`ap_cost: 2`（`TODO(balance)`） |
| 效果 | `topStanceGuarantee(+0.3)`（`TODO(balance)`）→ `topStanceReeval("both")` |
| 完成键 | `player_actions:escape_negotiation_guarantee` |
| 门控 | `event_completed special:escape_window_open` |
| 依赖 | 完整谈判机制（市长三条路之"谈判"）**本期留空占位**，只落最小保障动作 |

### 准备层动作（Layer 1 生产者）

| 动作 | 事件文件 | 效果 | 门控 | 备注 |
| --- | --- | --- | --- | --- |
| `ESCAPE_PREP_SUIT_REPAIR` | `escape_prep_suit_repair.js` | `suits_functional +1`、`prep_exposure +0.08` | 意愿＋Mechanical 0.25 | REPEATABLE，冷却 2 月（`TODO(balance)`） |
| `ESCAPE_PREP_SUIT_WAREHOUSE` | `escape_prep_suit_warehouse.js` | `suits_functional +4`（无暴露） | 意愿；`canTrigger` 读 `top_stance.it === "cooperate"` | 立场塑形可得性的样板 |
| `ESCAPE_PREP_TRANSFER_SUPPLIES` | `escape_prep_transfer_supplies.js` | `preparationTransfer("Supplies", 200)`；市长非配合时 `prep_exposure +0.04` | 意愿＋Supply 0.25 | 划拨即扣减；余额不足显式失败 |
| `ESCAPE_PREP_TRANSFER_MATERIALS` | `escape_prep_transfer_materials.js` | `preparationTransfer("Materials", 100)`；暴露同上 | 意愿＋Supply 0.25 | 同上 |
| `ESCAPE_PLAN_{MECHANICAL,MINES,SUPPLY,MEDICAL,AGRICULTURAL,ORDER}` | `escape_plan_*.js` | `preparationPlanRole(<role>)`、`prep_exposure +0.02` | 意愿＋对应部门关系 0.25（order 挂 Mayor） | 完备度＝覆盖率，派生重算 |
| `ESCAPE_DRILL` | `escape_drill.js` | `drill_level +0.15`、`prep_exposure +0.06` | 意愿 | REPEATABLE，冷却 3 月 |
| `ESCAPE_PREP_CARE` | `escape_prep_care.js` | `care_ready +0.4`、`prep_exposure +0.03` | 意愿＋Medical 0.25 | ONCE |
| `ESCAPE_PREP_PRODUCTION` | `escape_prep_production.js` | `production_start = true`、`prep_exposure +0.03` | 意愿＋Agricultural 0.25；触发器含 `preparationGte("materials_packed", 50)` | 材料门槛读账本 |
| `ESCAPE_RECEPTION_STOCKING` | `escape_reception_stocking.js` | `preparationTransfer("Supplies", 150)`＋`preparationStage("stocked")` | 先遣派遣；`canTrigger` 读档位 confirmed/stocked | 已囤备不重复划拨 |
| `ESCAPE_SPILLOVER_{IT_LINE,CROSS_SIGNAL,PHYSICAL}` | `escape_spillover_*.js` | `spilloverChannelAdd(kind, 3, 0.75/0.55/0.35)` | IT 专线：首档＋IT 0.3；跨堡信号：首档；实体：意愿 | 目标堡 3＝18 号（三堡布局）；投递属 Layer 3 |

### 反制事件 escape_prep_audit_seizure（审计没收）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/crisis/escape_prep_audit_seizure.js` |
| 触发条件 | `preparationGte("prep_exposure", 0.5)`（`TODO(balance)`）；REPEATABLE，冷却 6 月；`canTrigger` 读 `top_stance.it === "block"`（IT 不阻挠/配合时风险消失，16 §4.2-3） |
| 效果 | `suits_functional −2`、`prep_exposure −0.3`、`panic_value +0.02`（`TODO(balance)`） |
| 完成键 | `crisis:escape_prep_audit_seizure`（计数键） |
| DSL 能力 | 全部现有＋本批 `preparation_gte` 触发器 |
| 依赖 | 无 |

## 2. 清单1：可直接写的节点（本批已全部落地）

| 节点 | 文件 | 备注 |
| --- | --- | --- |
| W-1 意愿 | `events/special/departure_willingness.js` | 全部现有能力 |
| W-2 路线六节点 | `route_survey_{offer,accept,decline,defer}.js`、`route_survey_step.js`、`route_confirmed.js`、`advance_party_dispatch.js`、`advance_party_report.js` | 本批 `preparation_delta`（stage） |
| W-3 开窗＋R-D | `escape_window_open.js`、`escape_window_{orderly,riot}.js` | 本批 `top_stance_reeval` |
| W-4 出发 | `escape_departure_launch.js` | 本批 `escape_departure_settle`（服务层结算） |
| 立场生产者 | `escape_negotiation_guarantee.js` | 谈判占位 |
| 准备层十五动作＋审计反制 | 见 §1 准备层表 | 本批 `preparation_*` 全家 |
| 门控条目 | `events/action_gates.json` 追加 26 条 | — |

## 3. 清单2：必须等机制的节点

| 节点/能力 | 等的机制 | 未到位前的处理 |
| --- | --- | --- |
| ~~堡外月度维持检查~~ | **已落地（Layer 2）**：出发结算初始化 `OutsideState`，月度阶段 2c 结算补给/防护服/路线三侧面，覆没写明冻死/耗尽/被困 | — |
| ~~追踪四档~~ | **已落地（Layer 2）**：没人注意/已被察觉/咬住痕迹/正在追捕，热度＝出发定档基数＋1 号注意力现值−距离衰减；咬住起弃置补给，追捕封觅食并逐月损耗 | 反制入口仍待"日常动作中的反制"批次 |
| ~~仲裁读群体质量~~ | **已落地（Layer 2）**：`PlayerEscapeGroupQuality` 玩家贡献项读组合事实（老人/伤者/工匠/领头人），只对合格候选计分；资格仍只读群体存续 | — |
| 信道投递与独立接收 | Layer 3（外溢模型主体） | 信道只登记（建立侧），`PlayerPerSpilloverSilo` 口径收紧待 Layer 3 |
| 回退规则（窗口关闭 → 外溢） | "撤离窗口关闭"事实的生产 | 口径已登记（Witness 回退规则），事实生产属后话 |
| 完整谈判机制（市长三条路之"谈判"） | 另行设计 | 只落最小保障动作（占位） |
| 散逃者尾声（小规模强行出发） | 16 §3.2【待裁决】3 | 阻挠下强行出发自然结算为人数 0/极小，不构成逃出群体 |
| 立场的非叛乱生产路径 | 16 §3.2【待裁决】4 | 本批只按裁决基线由叛乱链供给立场事实 |
| 三选一／二选一兄弟选择隐藏 | 门控否定谓词 | 各线共用缺口，下游只读各键事实 |

## 4. 缺口汇总（给机制批次）

1. **Layer 2 逃离模型主体**：堡外月度维持、追踪四档、群体质量加权——出发结算的最小形态已固定输入输出契约（账本＋立场→`EscapedGroup`），Layer 2 只替换级联内部实现，不改资格接口。
2. **Layer 3 外溢模型主体**：信道投递、独立接收、暴露进 Silo 1 注意力轨道。
3. **门控否定谓词**：R-D 二选一与路线三选一的互斥呈现（多线共用缺口）。
4. **回退规则的窗口关闭事实**：逃跑条件未达成时进展改道外溢，需要"撤离窗口关闭"的可生产事实。

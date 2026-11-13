# Silo40 内容作者手册

> 一页速查：给 Silo40 加事件/改机制内容前先看这里，踩过的坑都在 §12。
> 配套资料：`events/README.md`（DSL 字段）、`references/parity-v7-three-silo.json`（当前确定性基线）。
> 本手册只描述现状与约定，不含待落地的设计。

## 0. 决策速查

| 我想… | 结论 | 动哪里 |
| --- | --- | --- |
| 加一个条件触发的剧情/危机事件 | 写内容 | `events/special/` 或 `events/crisis/` 新增 `.js` |
| 加一个按时间推进的里程碑 | 写内容 | `events/histories/`，`timestampAtOrAfter(year, month)` |
| 加一个常驻状态规则（恐慌→暴乱这类） | 写内容 | `events/rules/`，`script.canTrigger` + `script.apply` |
| 加一个玩家按钮动作 | 写内容 | `events/player_actions/`（目录未建，建了即生效） |
| 改每回合结算公式（资源/思潮/指标/胜负） | 写内容 | `events/mechanics/*.js`（mechanic 定义，不是内容事件） |
| 发/收信息碎片、改楼层稳定度 | 写内容 | 封闭集效果 `profession_fragment_add/remove`、`floor_stability_delta`（§4） |
| 加新的效果类型/触发器类型/变异类型/flag/结局 | **必须动引擎** | 见 §10，先立项 |

## 1. 目录分组与加载链路

| 分组 | 职责 | 加载方 |
| --- | --- | --- |
| `events/histories/` | 脚本化里程碑（按游戏历触发） | 内容加载（白名单） |
| `events/special/` | 条件性特殊事件 | 内容加载（白名单） |
| `events/crisis/` | special/histories 的后续升级或延迟代价 | 内容加载（白名单） |
| `events/rules/` | 常驻规则（带冷却，脚本判定） | 内容加载（白名单，`8eec461` 起） |
| `events/player_actions/` | 文件驱动玩家动作（尚未建目录） | 内容加载（白名单） |
| `events/mechanics/` | 每回合核心公式与动作定义（`defineMechanics`） | **MechanicEngine 单独加载**，不走内容白名单 |

加载链路：启动 `BootstrapContent` → `resolveContentDirectories`（优先二进制同目录
`events/`，缺失时从内嵌 FS 释放）→ `contentEventDirectories` **白名单过滤**
（`histories/history/special/crisis/rules/player_actions/player_action/events`，
`internal/service/game_service.go`）→ 逐文件 `ValidateContentEventDefinition` →
upsert 进库并禁用已删除者。**任何文件校验失败 = 整次同步失败 = 全部内容事件静默不加载**
（历史事故，见 §12.1）。

**防呆闸**（`internal/service/content_guard_test.go`，`d69cdcb`）：`go test ./internal/service`
时自动拦截——磁盘上有事件文件却被白名单漏掉的分组（死内容）、加载/校验失败、缺可用触发器
或效果、事件 ID 重复，都会红并报出分组/文件/事件名。新加事件分组后先跑它。

## 2. 事件文件格式

- 每文件恰好一次 `defineEvent({...})`；JS 由 goja 解析，函数被剥离、只留结构化数据 +
  `script` hook 标记。
- 必填：`id`、`title`、（`trigger` 或 `script.canTrigger`）、（`effects` 或 `script.apply`）；
  `fire_mode` 缺省 `ONCE`。
- 校验在加载时执行；**反例**：`events/special/outside_signal.js` 初版无 trigger 无脚本 →
  挡住全部内容加载（`bd18be1` 修复）。
- 代码注释用英文（仓库约定）；`id` 用 snake_case 并与文件名一致。

## 3. 触发器封闭集

| 触发器 | JS helper | 备注 |
| --- | --- | --- |
| `timestamp_at_or_after` | `timestampAtOrAfter(year, month)` | 游戏历基准 Year 100，30 天/月 |
| `silo_metric_gte/lte` | `siloMetricGte/Lte(metric, value)` | metric 见 §4 指标表 |
| `silo_resource_gte/lte` | **无 helper**，写裸对象 | `{ type: "silo_resource_lte", metric: "Supplies", value: 3000 }` |
| `silo_population_gte/lte` | **无 helper**，写裸对象 | `{ type: "silo_population_lte", value: 6000 }` |
| `silo_flag_true/false` | `siloFlagTrue/False(flag)` | flag 只有 `silo1_destroyed`（§9） |
| `event_triggered` | `eventTriggered(category, event_id)` | 上游须已在本局触发；带它的定义**不参与月度系统评估**，只在父事件派发时跟随触发 |
| `player_action_is` | `playerActionIs(action)` | 仅 `player_actions/` |
| `profession_metric_gte_count` | `professionMetricGteCount(metric, value, count, profession?)` | 满足条件的部门数 ≥ count |
| `profession_ideology_gte_count` | `professionIdeologyGteCount(ideology, value, count, profession?)` | 读的是 cohort 派生值（§8） |
| `all` / `any` | `all(...)` / `any(...)` | 组合 |
| （脚本） | `script.canTrigger(ctx)` | ctx 只读：`silo_metrics`/`resources`/`professions`/`runtime`/`states`/`total_population`/`year`/`month` |

**纯调度事件约定**：只经 `schedule_event` 唤起的事件，写
`script.canTrigger: function(ctx) { return false; }`——延迟队列派发不检查触发条件，
而月度评估因恒假不会提前点燃。反例：不加它则校验报 "missing trigger or script.canTrigger"。

## 4. 内容事件效果封闭集

| 效果 | JS helper | 可用字段 | 持久性 |
| --- | --- | --- | --- |
| `silo_metric_delta` | `siloMetricDelta(metric, v)` | `legitimacy` `cohesion` `rebellion` `dept_tension` `class_fragmentation` `history_burden` `event_trigger` `countdown` | 见下表 |
| `silo_resource_delta` | `siloResourceDelta(res, v)` | `Energy` `Materials` `Supplies`（下限 0） | 持久 |
| `silo_population_delta` | `siloPopulationDelta(v)` | 总人口（下限 0） | 持久 |
| `profession_metric_delta_all` | `professionMetricDeltaAll(metric, v)` | `panic_value`（年衰减 10%）/ `productivity` | 持久 |
| `profession_ideology_delta_all` | `professionIdeologyDeltaAll(ideology, v)` | `pro_foreign` / `loyalty` | **短暂**（§8） |
| `silo_flag_set` | `siloFlagSet(flag, bool)` | 仅 `silo1_destroyed` | 持久 |
| `profession_fragment_add` | `professionFragmentAdd(profession?, fragment?)` | profession 空=补给碎片最少的部门；fragment 空=种子随机抽该部门未知碎片 | 持久 |
| `profession_fragment_remove` | `professionFragmentRemove(profession, fragment)` | 两字段必填；不存在则无操作 | 持久 |
| `floor_stability_delta` | `floorStabilityDelta(zone?, v)` | zone 空=全部楼层，否则 `Upper`/`Mid`/`Lower`；0–1 截断 | 持久 |
| `schedule_event` | `scheduleEvent(event_id, delay_months)` | `delay_months > 0` | 见 §6 |

指标持久性（`metrics_update`/`ideology_update` 每 tick 重设派生值，内容 delta 活不过下一 tick）：

| 持久（内容事件的有效杠杆） | 派生/短暂（写了只有当次快照可见） |
| --- | --- |
| `rebellion`、`history_burden`、`countdown`、资源、人口、部门 `panic_value`、`productivity`、部门碎片、楼层稳定度 | `cohesion`、`legitimacy`（=派系加权凝聚力）、`dept_tension`（=1−关系均值）、`class_fragmentation`（=忠诚差）、部门 `Ideologies`（§8） |

**别碰 `event_trigger`**：空池虽已防护（`bd04aac`），正向 delta 仍无收益、只有风险。

## 5. 机制脚本（`events/mechanics/`）变异封闭集

`defineMechanics` 的 `apply` 返回 `{ mutations, emit, logs, action_result, stats, score }`；
变异由 `internal/engine/mechanics.go` 的 `applyMechanicMutations` 解释：

`actor_metric_delta|set`（字段：`action_points` `suspicion_level` `political_prestige`
`political_points` `propaganda_level`）、`actor_connection_delta`、`actor_fragment_add`、
`profession_metric_delta|set`（`panic_value` `productivity`）、`profession_ideology_delta`、
`profession_fragment_add|remove`、`silo_metric_delta|set`（比内容效果多 `safeguard_risk`）、
`resource_delta`、`resource_net_balance_set`、`cohort_ideology_delta(_all)`（**唯一能持久改
思潮的入口**，可按 `class_type`/`profession` 过滤）、`sync_profession_ideologies_from_cohorts`、
`apply_population_deaths`、`refresh_profession_population_from_cohorts`、
`faction_metric_delta_actor`、`faction_is_public_actor`、`silo_flag_set`、
`victory_status_set`、`score_set`、`log`。

机制脚本里可用 `randomFloat()/randomInt()`（引擎单一随机源，种子可复现）。
核心系统事件：`AGENT_UPDATE` `RESOURCE_UPDATE` `IDEOLOGY_UPDATE` `METRICS_UPDATE`
`VICTORY_CHECK`（`events/mechanics/core_systems.js`）——改结算公式改这里，Go 侧有降级副本
（`victory_fallback_test.go` 锁定语义一致）。

## 6. 冷却与延迟

- `fire_mode: "REPEATABLE"` + `cooldown_months`：按游戏时间戳强制（30 天/月），冷却内条件
  满足也不触发。可从以下档位开始 sim 校准：小麻烦 cd 3–4、中等 cd 6、大灾 cd 9–12、
  唯一事件 `ONCE`。
- `schedule_event(event_id, N)`：N 个月后**无条件**派发目标事件（不检查其触发条件），
  走 `delayTasks` 队列。用于延迟代价链：`silo1_ultimatum → ultimatum_aftermath(+3 月)`、
  `materials_cannibalization → lower_deck_collapse(+4 月)`。
- **时间窗约束**：可玩窗口 = Y122M1 开局 → Y130M1 `silo1_destroyed_signal` → TIME 结局，
  约 96 个月。所有事件与延迟落点必须在 Y130 前；延迟档用 2–3 月（短）/ 4–6 月（中），
  且发起时机不晚于 ~Y128。
- **第 N 次质变**：`script.apply` 读 `ctx.runtime.trigger_count`（本次触发后的累计次数，
  首次=1）分支，范例 `events/special/materials_cannibalization.js`（第 5 次起出事故）。
- 密度预算：大事件 ≤1 个/月（脚本里程碑间隔 ≥3 月）；新增循环事件合计 ≤ ~0.55 次/月。

## 7. 事件 ID 与 `schedule_event` 解析

- `schedule_event` 与状态查找按 `EventID` **首个匹配**解析（`findContentDefinition`）：
  ID 重复 = 静默影子覆盖。防呆闸会报重复（`TestContentGuardEveryEventLoadsValidAndTriggerable`）。
- 运行时事件名 = `category:event_id`（如 `crisis:purge_backlash`）；`script.apply` 的
  `emit` 与 `event_triggered` 匹配都用这个名字。
- `event_triggered` 订阅链：父事件派发 → 订阅者立即评估跟随触发（同 tick）。
  **不要订阅经延迟队列派发的事件做依赖测试以外的用途**——历史崩溃路径，见 §12.5。

## 8. 思潮：cohort 是唯一真源

**结论**：`Profession.Ideologies` 是 cohort 的人口加权派生视图，每 tick 被
`sync_profession_ideologies_from_cohorts` 覆写（设计如此，`2255de4` 起）。内容事件的
`profession_ideology_delta_all` 活不过下一 tick。

- **反例**：`outside_signal` 的 `pro_foreign +0.05` 无持久机制作用，只有触发当月快照可见。
- 内容事件想持久推动思潮，唯一现有杠杆：`panic_value`（恐慌 >0 时驱动 cohort `pro_foreign`
  漂移 `panic×0.10/年`，恐慌 >0.3 且凝聚力 <0.5 时附加漂移）。
- 读侧注意：`profession_ideology_gte_count` 触发器读的是派生值——它反映的是 cohort 现状，
  可放心用作条件。
- 需要内容直达 cohort 思潮 = 新增内容效果类型 = 引擎改动（§10）。

## 9. 内容事件目前拿不到的系统

| 系统 | 现状 | 后果 |
| --- | --- | --- |
| `safeguard_risk` | 内容 metric 集不含它（机制变异层含） | 内容事件碰不到 Safeguard 死局 |
| 新 flag | 读写只有 `silo1_destroyed` | 无法自建状态链，用 metric 阈值 + `event_triggered` 代替 |
| 楼层按 Function 过滤 | `floor_stability_delta` 只按 Zone 过滤 | 想只打"机械区"而非整个 Lower 需引擎扩展 |
| 碎片分发策略 | `profession_fragment_add` 只有"最缺部门 + 随机未知"一种策略 | 指定部门/指定碎片之外的策略需引擎扩展 |

（碎片与楼层本身已可达，见 §4。）

## 10. 内容改不动、必须动引擎的清单

出现以下需求时**先立项改代码**，不要试图用内容绕过：

1. 新效果/触发器/变异类型，或新 flag、新结局、新管线阶段（`content_events.go` 校验 +
   `mechanics.go` 变异 + JS prelude helper 三处同改）。
2. 内容事件动 `safeguard_risk`、直达 cohort 思潮（§8）、按 Function 改楼层。
3. 新事件分组的加载（白名单在 `internal/service/game_service.go`；加完跑防呆闸）。
4. 事件权重/随机导演、on-off 周期、硬保底（当前机制层不存在）。
5. 管线顺序与结算时机（`runTickPipeline` 步骤 1–9 硬编码）。
6. 结局判定优先级（TIME 先于 REBELLION 等；`victory_check` JS + Go fallback 双份同改）。

## 11. 本地验证手册

```bash
# 1. 快速门槛：构建 + 全量测试（含防呆闸与内容加载校验）
go build ./... && go test ./internal/...

# 2. 只跑内容守卫
go test ./internal/service -run 'TestContentGuard|TestRulesGroupIsLoaded' -v

# 3. 重构 GUI 二进制（事件随内嵌 FS 打包；改了 Go 代码必须重构）
wails build -skipbindings
#    注意：wails build 可能把 frontend/wailsjs/runtime/* 的文件模式改掉，提交前 git checkout -- 还原

# 4. 起服务 + 无头模拟（需要 GUI 环境，DISPLAY 可用）
rm -rf build/bin/events            # 清旧释放副本，确保加载仓库最新事件
SILO40_DEBUG_HTTP_ADDR=127.0.0.1:18080 ./build/bin/silo40 &
go run ./cmd/sim -addr http://127.0.0.1:18080 -games 20 -seed 7 -months 200
```

- **种子复现**：`-seed S` 时第 i 局种子 = `S+i`，行动策略也由 S 派生；同参数双跑必须逐局
  一致（不一致即引入了未受控随机源）。存档种子随 `Silo.RNGSeed` 持久化。
- **基线对照**：改动前后以同种子、同门控配置跑 sim 或 parity，比较结局分布与关键指标。
  当前确定性基线为 `references/parity-v7-three-silo.json`；旧版本结局分布不能直接套到
  当前三堡与门控实现，也不能把 ungated parity 当成生产平衡验收。
- **逐事件时序验证**：临时探针模板——`bootstrapRealContent(t)`（见
  `internal/service/rules_loading_test.go`）+ `CreateGame{Seed: 固定}` + 逐月 `PassTime`，
  断言 `svc.contentStates[key].TriggerCount/LastTriggeredTimestamp`。跑完删除，不入库。
- **结局分布看什么**：分布偏移、结局月份提前/推迟、终局人口均值大幅移动、叛乱接近 1.0、
  出现"必爆/从不触发"——任何一条都要在报告里给出前后对照。

## 12. 历史坑速查

| # | 坑 | 修复/守卫 |
| --- | --- | --- |
| 1 | 事件缺 trigger/脚本 → 校验失败 → **全部内容事件静默不加载** | `bd18be1`；防呆闸拦截 |
| 2 | `events/rules/` 被白名单排除，4 个规则事件成死内容 | `8eec461`；`TestContentGuardNoDeadGroups` 拦截 |
| 3 | `delayed_riot` 无触发条件不过校验（纯调度事件） | `8eec461` 起用 `canTrigger: () => false` 约定（§3） |
| 4 | 对部门思潮做持久设计——每 tick 被 cohort 覆写 | 判定为设计（`2255de4`），见 §8 |
| 5 | 订阅延迟队列派发的事件（无 `story_sink`）→ 空指针崩溃 | `51d312f` 已防护；`grief_release_ritual` 已恢复订阅 `blackout_cascade`，`TestDelayedDispatchFollowupDoesNotCrash` 锁定 |
| 6 | `EventTrigger` 空池 `rng.Intn(0)` | `bd04aac` 空池不触发；内容别加 `event_trigger` delta（§4） |
| 7 | 内容故事不进 `TickResult.Stories`（`engine.Drain` 丢 `step_stories`；step 流丢 `runOneMonth` 返回值且操作卡死） | `fd03bf7` + `326241d`；测试 `TestPassTimeReturnsContentStories` / `TestProcessNextEventSurfacesMonthStories` |
| 8 | 事件越过 Y130 窗口（`silo1_destroyed` → TIME）永不触发 | 时序约束见 §6；探针断言触发时点 |
| 9 | 对派生指标（凝聚力/合法性/部门紧张/阶层裂痕）做机制设计 | 每 tick 重设，见 §4 持久性表 |
| 10 | 叛乱门槛事件的门槛值对结局分布**极端敏感** | 门槛必须落在无人值守叛乱轨迹的双峰之间才有分水岭效果；调参务必跑 sim 对照，别凭手感 |
| 11 | 楼层稳定度消费层在稳定度 1.0 时必须**完全中性**（产出因子=1、死亡不加项），否则无声打乱基线 | `core_systems.js` 用 `0.5+0.5×稳定度` 与 `<0.8 才加死亡` 保证中性；`TestFloorStability*` 锁定 |
| 12 | 少量碎片直发不足以保证无人值守的 INFORMATION | 该结局主路径仍是玩家/NPC `SHARE_INFO`；`TestFragmentBridgeMakesInformationEndingReachable` 只锁结构可达 |

## Sources

- 代码事实：`internal/engine/content_events.go`（触发/效果/校验）、
  `internal/engine/mechanics.go`（变异集）、`internal/engine/story_events.go`、
  `internal/service/game_service.go`（白名单/调度/订阅/step 流）、
  `internal/engine/engine.go`（管线与 Drain）、`events/README.md`（DSL）。
- 当前确定性基线：`references/parity-v7-three-silo.json`。
- 历史决议：`2255de4`（cohort 思潮真源）、`bd18be1`/`8eec461`/`51d312f`/`bd04aac`/
  `fd03bf7`/`326241d`/`d69cdcb`（缺陷修复与防呆闸）。

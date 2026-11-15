# 五个失败结局 · 叙事、征兆与失败后呈现

> [返回总索引](../silo40-endings.md) · [机制底座](./10-chain-foundation.md) · [十线仲裁](./22-chain-cross-arbitration.md)
> 性质：设计文档＋本批已落地的内容。沿用 [25《共证》](./25-information-endgame.md)／[28《公议》](./28-rule-of-law-endgame.md) 的终局呈现链路（`endingDescriptions` → `decision.Reason` → `VictoryStatus.Description`），不新造展示通道。
> 口径：成立条件以现有代码为准（先读后写）；不碰引擎资格判定与仲裁逻辑；原书口径、功能位、无影视改编人名；数值一律 `TODO(balance)`；失败不等于世界消失。

## 1. 五条失败的成立条件（现有代码事实）

| 结局 | 成立条件（代码口径） | 优先级 |
| --- | --- | --- |
| 《寂堡》DEATH | 引擎 `safeguard_risk ≥ 1.0` 或人口 ≤ 0（`victory_check`）；或仲裁侧 `crisis:termination_release`／`crisis:bombing_terminal` 事实 | P0.1，先读逃出群体（存续群体改判《越岭》惨胜） |
| 《遗忘》RESET | 事实键 `special:reset_completed` | P0.2 |
| 《断层》FRACTURE | 事实键 `special:fracture_confirmed`（由 `fracture_point_of_no_return` 唤起：能源 ≤0 且叛乱 ≥0.9，或人口 ≤3000 且叛乱 ≥0.8） | P0.3 |
| 《除名》AGENT_COMPROMISED | 引擎 `suspicion_level ≥ 1.0`（`victory_check`） | P0.4 |
| 《深闭》EXCLUSIONIST | 加权候选：封闭思潮主导（人口加权忠诚 > 奉外）或 Silo 1 已毁；同分顺序最末，是默认收敛 | P1/P2/P3 |

P0 四条的 `Reason` 原为审计短句，本批替换为完整叙事＋失败后结算行；《深闭》走既有 `endingDescriptions` 追加路径。审计不损失：分数、资格池、排除原因仍在 `EndingDecision` 结构化字段。

## 2. 事前征兆（全部走既有月度事件与故事通道，无新预警机制）

### 2.1 《深闭》：`normalcy_hardens`（本批落地）
相对 Y3 起，≥4 个部门忠诚思潮 ≥0.6（公共主导档，`TODO(balance)`）时按年点亮（REPEATABLE cd12）："又一年没有改变"。计数即叙事；效果 `history_burden −0.02`、`rebellion −0.02`（常态化的双向磨平，`TODO(balance)`）。若局势仍停在 `arrival`，该事实直接打开既有终局仲裁，让零操作惯性局能够收敛；一旦真实行动已进入 `first_signal` 或更后阶段，活跃路线构成爬出，不被此入口提前截断。

### 2.2 《遗忘》：18 分册链已落地，不新增
- `reset_omen_census`（已实现）：相对 Y2 起＋`history_burden ≥ 0.2`——一份没有维护编号的异常普查，第一个无法解释的征兆；+3 月调度第二征兆。
- `reset_omen_requisition`（已实现，纯调度）：镇静与配额类物资的异常请领；解锁反制接触卡，并 +9 月调度执行守卫。
- 反制窗口：`reset_countermeasure_offer` → `copies_distributed`（写 `player_action:reset_countermeasure_done`）；`reset_execution` 到期守卫：反制完成→空转，未完成→结算并唤起 `reset_completed` 事实键。
- 并行路径征兆：`silo1_purge_omen`（通牒后＋叛乱 ≥0.8 的本地整肃准备）、`archive_purge_order`（hb ≥0.12 的缓慢绞杀）。

### 2.3 《断层》：既有节点，不新增
- `fracture_infrastructure`（已实现）：相对 Y2 起＋叛乱 ≥0.8——无人控制的破坏落到维生基础，先于不可逆点。
- 既有 `rebellion_escalation` 规则与 `underground_coordination` 亦在更早阶段给信号。

### 2.4 《除名》：20 分册三级征兆（本批落地）
- `identity_trace_accumulates`：REPEATABLE cd2，订阅秘密行动完成键（`STATION_INFILTRATE`／`STATION_DECEIVE`／`SILO1_DECEIVE`）——每次深度行动留一道纸面接缝。
- `archive_review_opened`：痕迹计数 ≥5（`TODO(balance)`）——本地 IT 档案复查，身份线与证据线一起燃烧。
- `agent_erasure_staged`：`ctx.actor.suspicion_level ≥ 0.7`（`TODO(balance)`）——抹除以本地名义准备（演出约束：永不公开审讯"他堡特工"）。

### 2.5 《寂堡》：既有节点＋补一个缺口
- 终止链：`silo1_ultimatum` → `termination_channel_test`（已实现）；物理链：`strike_reconnaissance` → `bombing_terminal`（已实现）。
- `last_interlock_warming`（本批落地）：`safeguard_risk ≥ 0.6`（`TODO(balance)`）——保障机制升温；该子路径此前无任何征兆。

## 3. 叙事与"在哪一步做错了"

每条叙事都写明失败的具体形状，不用笼统的"你输了"：

| 结局 | 做错了什么 |
| --- | --- |
| 《深闭》 | 没有做错到被阻止，也没有做对到改变什么——等得太久、藏得太深，或改革失败反成封闭养料；窗口安静地关上 |
| 《遗忘》 | 征兆被当成噪音，或反制窗口被花在了别处——输掉的是整块记忆，不是一场对抗 |
| 《断层》 | 点燃了愤怒，没有建起接住它的组织——期待跑得比组织快，维存网撕掉一条网眼不因任何"胜利"复原 |
| 《除名》 | 一直赌下一次行动是最后一次——痕迹从一开始就在，身份安全与任务进度方向相反 |
| 《寂堡》 | 按死因分支：偏离招来处置（外溢与高调叛乱送出可操作观察）／物理手段几乎无解／过度干预把保障风险推到封顶／最后一班没有人接 |

## 4. 失败之后的呈现（结算行）

失败不等于世界消失。每条在叙事末尾追加动态结算行：

| 结局 | 结算行报告什么 |
| --- | --- |
| 《深闭》 | 两变体（Silo 1 尚存：新主管就职；已毁：死后复制"秩序内化了它的主人"）＋居民数与正统性："驻地仍在运转，死掉的是改变的可能" |
| 《遗忘》 | 剩余居民数与历史包袱："人还在，记忆不在了" |
| 《断层》 | 剩余居民、能源、楼层平均稳定度："技术上还活着，已经失去了明天" |
| 《除名》 | 驻地继续：居民数、部门数、部门信念层仍可引用的事实数——"你教过的方法还在被人使用，只是没有人知道它来自哪里" |
| 《寂堡》 | 未存续的逃离（"曾有 N 人离开——残酷的注脚"）＋外部世界（按 Silo 1 存毁分支）＋设施状态（堡体矗立与否）；残余应答母题不许诺 |

## 5. 依赖与明确不做

| 依赖/不做 | 说明 |
| --- | --- |
| 反制绑定 | `silo1_purge_exposed` 的信息侧已绑定日常动作 `PURGE_PLOT_LEAK`（征兆后 + 司法/市长关系 ≥0.2，调度 1 月后点亮；执行按中档结算，与名册篡改的轻档并存时按轻档）；《遗忘》链的反制已走 `copies_distributed` 分发托管路径 |
| 不做 | 不改 P0 顺序与成立条件；不给失败结局加解锁项或再玩钩子；失败叙事不引入影视改编人名 |

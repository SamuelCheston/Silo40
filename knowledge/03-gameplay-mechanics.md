# 核心玩法机制

## 1. 驻地、部门与人口

驻地使用普通 Silo 初始化：10000 人、三类资源、九个常规部门、144 层楼。Silo 1 在相同本地规则上增加 Archives 并调整人口/权力/历史参数；Silo 18 另有正典人物和事件链。

Cohort 是思潮与人口演化的真源，部门 Ideologies 是 cohort 加权视图。Resident 用于关键个体和派系代表，不表示逐人模拟全部人口。Silo 18 的具名 Resident 是正典个体，其生死只由 N 节点改变。

资源生产、消耗、恐慌、忠诚、亲外、合法性、凝聚力、叛乱和楼层稳定度仍由本地 mechanic 驱动；普通行动和 operation 最终都落到这些状态，但跨堡行动只能修改明确允许的目标。

## 2. 行动门控三态

后端 `PlayerActionMeta` 用以下状态表达完整生命周期：

```text
LOCKED：未满足 unlock，完全不下发
  → 首次满足后写入 World.UnlockedActions，永久保留
UNLOCKED：已解锁，但 available、目标/剧情条件或 AP 不足；灰项 + disabled_reason
  → 每次快照重新求值
AVAILABLE：当前可执行
```

`unlock` 只向前，`available` 可随状态上下波动。条件支持递归 all/any，以及：

- `event_completed`：驻地 ContentEventState 中指定节点已完成。
- `stage_at_least`：达到有序阶段数组中的某阶段。
- `metric + min/max`：AP、威望、宣传、怀疑、已知碎片、合法性、凝聚力、叛乱、历史包袱。
- `relation + min/max`：玩家与指定部门的 Connection。

BeginAction 和 ExecuteAction 都在成本解析后调用同一 gate 校验，不能通过构造 HTTP 请求执行 LOCKED/UNLOCKED 行动。生产缺注册表或配置非法会显式失败并默认锁定，不回退为全按钮开放。

## 3. 当前阶段与解锁内容

| 内容 | 解锁条件 | 当前可用条件 |
| --- | --- | --- |
| Gather Info | 开局 | 总是通过 gate，仍受 AP/目标约束 |
| Share Info | first_signal | 至少 1 个已知碎片 |
| Build Connection | first_signal | 无额外 gate 条件 |
| Conduct Propaganda | Mayor 关系 ≥ 0.15 | first_signal |
| Incite Rebellion | 政治威望 ≥ 60 | Mayor 关系 ≥ 0.25 |
| Publicize Faction | archive_awakened | 政治威望 ≥ 80 |
| 全部职业行动 | first_signal | 无额外 gate 条件 |
| STATION_INTEL / SILO1_INTEL | first_signal | 无额外 gate 条件 |
| STATION_INFILTRATE | archive_awakened | Mechanical 关系 ≥ 0.20 |
| STATION_DECEIVE / SILO1_DECEIVE | archive_awakened | 宣传 ≥ 1 |
| STATION_TRADE | Supply 关系 ≥ 0.20 | first_signal |

阶段链是 `arrival → first_signal → archive_awakened`。成功 Gather 写入 `player_action:action_gather_info`，成功 Share 写入 `player_action:action_share_info`；只有执行成功才产生节点。

未在注册表声明的内容由 `default_visibility` 决定，当前默认 LOCKED。旧机制单测不加载生产注册表时保留 ungated fixture，这不是生产绕过方式。

## 4. 通用、职业与跨堡 operation

通用行动和职业行动仍由 `events/mechanics/` 定义 AP、耗时和效果，服务端覆盖客户端 cost。玩家普通行动始终指向驻地。

固定 operation 使用独立 ActionType 与 ActionID：

| 目标 | ActionID | AP / 延迟 | 主要结果 |
| --- | --- | ---: | --- |
| 驻地 | STATION_INTEL | 10 / 1 月 | 确认精确计量和部门状态 |
| 驻地 | STATION_INFILTRATE | 25 / 2 月 | 合法性/凝聚力下降，部门紧张上升 |
| 驻地 | STATION_DECEIVE | 20 / 2 月 | 叛乱、恐慌和亲外上升 |
| 驻地 | STATION_TRADE | 15 / 1 月 | 消耗驻地 Supplies，合法性/凝聚力和玩家威望上升 |
| Silo 1 | SILO1_INTEL | 10 / 1 月 | 确认 Silo 1 精确状态 |
| Silo 1 | SILO1_DECEIVE | 20 / 2 月 | Silo 1 叛乱、恐慌和亲外上升 |

下单立即扣 AP，目标 delta 到期才应用。驻地与 Silo 1 分别累计 Exposure；每跨 25% 提升反间谍阶段，增加怀疑并损害驻地合法性。黑市不向 40 号转资源，Silo 18 没有 operation 入口。

## 5. NPC 与持久派系

NPC 采用效用候选和共享 RNG 加权选择。候选动作现在显式携带 TargetDept、FragmentIds 或 ResourceTarget，执行观察器记录真实结果；早期“目标只写进 message、执行必然失败”的 KI-1 已修复。

派系由 cohort 特征距离聚类，不再按签名分桶：亲外、忠诚和阶层构成加权向量；既有归属有惯性半径，政治准入仍有 0.7/0.6 迟滞。lineage 继承持久 ID、名称、公开状态、威望、Stance 和 Memory，新 ID 单调分配；Demands 随主导 drive 派生。派系再驱动驻地合法性、凝聚力、叛乱和结局贡献。

## 6. 结局：硬护栏加历史仲裁

基础 VICTORY_CHECK 先产生触发。DEATH、人口灭绝、AGENT_COMPROMISED、Safeguard 等失败不被覆写。驻地的 INFORMATION/TIME/REBELLION/EXCLUSIONIST 胜利触发才进入 WeightedEndingResolver。

四类主体贡献分数：

- 玩家：基础值、威望、碎片、宣传。
- 驻地非 Unaffiliated 派系：影响力、开放倾向、低凝聚力。
- 驻地存活关键 Resident：影响力、开放思潮、野心。
- Silo 1：存活/覆灭基础值、叛乱和凝聚力。

派系与 Resident 先按稳定 ID 排序，候选固定为 INFORMATION → TIME → REBELLION → EXCLUSIONIST。`Selected` 从原 trigger 开始，只有候选分严格超过当前 bestScore 才替换，因此同分由固定顺序决定。最低分为 1。

仲裁写入 `VictoryStatus.Decision`，保存配置版本、候选总分、每个主体贡献和选择结果。当前全部参数是占位，不应把某个结局比例解释成平衡结论。

## 7. 基础评分与历史引导分

基础 GameScore 仍由人口、部门多样性、历史包袱、平均亲外和结局倍率构成。首次结局后，历史引导分一次性附加且重复保存不重复累加：

- 仅统计已结算驻地/Silo 1 operation 的目标 before/after delta。
- 合法性和凝聚力按 `abs(delta)×100`，叛乱 ×120，平均亲外 ×80，Supplies delta /100，四舍五入。
- 每堡最高 100，合计评级 S/A/B/C/D，阈值为 150/100/50/1/0。
- 情报行动可保留 0 分证据；普通行动、自然演化和 Silo 18 正典不计入。

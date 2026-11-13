# 扩展指南：事件、行动、阶段与楼层

## 1. 先选正确的扩展层

| 需求 | 必改位置 | 额外检查 |
| --- | --- | --- |
| 新剧情/危机/常驻规则 | events 对应分组的新 JS | DSL、目标 scope、正典边界、内容守卫 |
| 新文件驱动玩家事件 | `events/player_actions/` + action gate 注册 | 当前默认 LOCKED，未注册不会出现 |
| 改通用/职业行动 | `events/mechanics/*actions.js` | 成功完成节点、AP、目标与 gate |
| 新职业行动 | profession mechanic + action gate | 职业归属、action ID 全局唯一 |
| 新通用行动类型 | mechanic + Go/TS ActionType + gate | 服务端成本解析与 UI union |
| 新阶段/解锁条件 | `events/action_gates.json` | 有序阶段、完成节点、永久解锁与实时可用分离 |
| 新驻地/Silo 1 operation | Go operation 定义 + gate + DTO/UI | 不是只加 JSON；需目标、延迟、Exposure、历史影响 |
| 新跨堡信号 | 内容事件 + CrossSignalRequest/目标 scope | 白名单、队列顺序、ONCE 目标状态 |
| 新结局候选 | JS/Go trigger + weighted config + DTO/UI | 固定候选顺序、仲裁记录、存档版本与平衡基线 |
| 新楼层 | `initializer.go`，必要时迁移 | 新开局与旧档区别、Zone 消费、parity |
| 新 trigger/effect/mutation/flag | Go 校验/应用 + JS helper | 封闭集，不允许静默忽略 |

## 2. 新增普通内容事件

1. 选择 histories/special/crisis/rules，写与 ID 同名的 JS 文件。
2. 优先结构化 trigger/effects，复杂分支才用 script hooks。
3. 多堡效果明确 self/world/silo/neighbors/all，不默认把“玩家堡”误当 40 号。
4. 跨堡后果使用 cross_signal，不直接遍历或改写其他 Silo。
5. schedule-only 事件用恒假 canTrigger 防普通月度提前触发。
6. 检查 Y122–Y130 正典窗口和 Silo 18 不可干预边界。
7. 跑内容守卫与固定 seed 对照。

不要通过内容事件直接杀 Silo 18 具名人物、复活 Silo 1 或覆写 N 节点；这些属于 Go 正典权威。

## 3. 新增或修改行动

### 职业行动

在 `events/mechanics/profession_actions.js` 增加定义，使用现有 profession、target 和 mutation。随后必须在 `events/action_gates.json` 添加对应 ID 或确认已有匹配规则；生产默认 LOCKED，未声明不会自动开放。

成功行动会写 `player_action:<id>`，所以 ID 同时影响：

- mechanic 路由和职业归属；
- 完成节点与阶段条件；
- World.UnlockedActions 中的门控 key；
- 内容状态和历史审计。

修改 ID 不是纯重命名，要审查旧档与所有引用。

### 通用行动

修改现有六类只改 mechanic 和对应 gate 即可。新增第七种 ActionType 还需同步 `internal/model/models.go` 与前端类型，并验证服务端 cost 解析、目标 DTO、NPC 候选是否应包含它。

### 文件驱动玩家事件

`events/player_actions/` 管线可用，但当前没有正式内容。新增 player_action 元数据后仍要注册 gate；scope=faction_member 当前不启用，新的 target/effect 仍需 Go 扩展。

## 4. 新增门控阶段

阶段数组是有序状态机，不是标签集合。加入阶段时：

1. 给唯一 ID，放在正确顺序位置。
2. `when` 使用已成功完成的事件节点或可稳定读取的属性/关系。
3. 为相关行动分别设置 unlock 与 available：前者永久，后者每次快照重算并给 disabled reason。
4. 确保阶段不能由空转月份意外推进。
5. 添加开局 LOCKED、解锁后灰项、条件满足 AVAILABLE、服务端绕过拒绝、解锁存读档测试。

新增 metric 需要加入 `supportedActionMetric` 和求值函数；仅在 JSON 中发明字段会因 DisallowUnknownFields 或校验失败。

## 5. 新增 operation 或跨堡通道

固定 operation 不是普通 mechanic。需要同时实现：

- ActionID、AP、耗时、允许目标和 Go 定义。
- 下单与到期结算、目标 before/after、Exposure/反间谍。
- World 中 operation track 的持久化与公开快照。
- action gate 注册和前端展示。
- 如属玩家贡献，接入 HistoryImpact；否则明确不记分。
- 结算顺序保持驻地 ID 1 → Silo 1 ID 2，禁止 Silo 18 入口。

跨堡内容信号则走 CrossQueue，并验证到期时间、Seq、目标排序、receivedSignals 清理和读档。

## 6. 修改结局

基础硬失败不能被加权器覆写。新增/调整历史候选需要同步：

- JS VICTORY_CHECK 与 Go fallback trigger。
- `endingOutcomeWeights`、default config、候选顺序和最低分。
- 每类实体贡献与稳定 ID 排序。
- VictoryStatus.Decision、前端结局解释和存档版本策略。
- 同分、零贡献、硬失败、Silo 18 不接入、Silo 1 覆灭权威测试。

当前权重全是 TODO(balance)。结构调整可以先做定向测试，分布结论必须另做生产门控下的批量实测；ungated parity 不能替代。

## 7. 修改楼层或世界生成

普通楼层在 `initFloors` 配置，Silo 1 还会在 11–15 层覆写 Archives Function。修改时检查 Level 连续性、Zone、稳定度中性点及资源/死亡消费。

已有存档不会重新运行初始化；需要旧档获得新实体时，必须显式迁移并提升版本。三堡构成、40 号身份、驻地派生和 Silo 18 正典不是普通楼层配置，不能用同一方式顺手修改。

## 8. 最低验收

```bash
cd frontend && npm ci && npm run build && cd ..
go vet ./...
go build ./...
go test ./...
go test -race ./...
go test ./internal/service -run 'TestContentGuard|TestActionGate|TestWeightedEnding' -v
```

行为中性改动还应与 `references/parity-v7-three-silo.json` 做同 seed diff。涉及生产门控、结局权重或真实采样时，再补对应集成测试，不能用跳过门控的旧 fixture 代替全部验收。

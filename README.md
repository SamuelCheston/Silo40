# Silo40

基于《羊毛战记》世界观的单机策略游戏：地堡生存、思潮演化、多结局。
Wails v2 桌面应用（Go + React/TS + SQLite）。

## 架构（以代码为准）

**Go 后端是唯一事实来源。** 游戏规则、数值演化、结局判定全部在 Go 引擎中执行；
前端只是 UI，通过 Wails 绑定调用后端、渲染后端下发的状态快照，不持有任何
游戏规则副本。

```
main.go / app.go            Wails 入口与绑定层（前端可调用的全部 API）
internal/engine/            规则引擎：回合管线、事件总线、动作执行、
                            结局判定、Mechanic 脚本宿主（goja）
internal/service/           有状态会话：存档/读档、事件队列、延迟任务、
                            内容事件加载与逐月推进
internal/model/             GORM 模型（= SQLite schema）与前后端 DTO
internal/repository/        SQLite 初始化（GORM AutoMigrate）
internal/api/               调试用 HTTP 面（SILO40_DEBUG_HTTP_ADDR）
events/                     内容层：剧情事件、规则事件、机制脚本（JS）
frontend/src/               React UI（纯展示与输入，无游戏逻辑）
cmd/debug/                  世界生成转储工具
cmd/sim/                    无头长跑模拟器（调平衡用）
```

### 内容层：`events/**/*.js`（goja 内嵌执行）

事件与机制以 JS 脚本定义，启动时由 Go 解析、校验并执行（`internal/engine/
content_events_script.go`、`mechanics.go`）。脚本**只能读取状态快照、返回声明式
结果**（效果/变异/发射事件/延时调度），由 Go 校验后应用——不能直接改 Go 状态，
也不能新增效果类型。

| 目录 | 内容 |
|---|---|
| `events/histories/` `events/special/` `events/crisis/` | 剧情事件：结构化 `trigger`/`effects`，可选 `script.canTrigger/apply` 钩子 |
| `events/rules/` | 常驻规则事件（如叛乱升级、恐慌风暴），可重复触发带冷却 |
| `events/mechanics/` | 每回合核心系统（`AGENT_UPDATE`/`RESOURCE_UPDATE`/`IDEOLOGY_UPDATE`/`METRICS_UPDATE`/`VICTORY_CHECK`）、通用与职业行动、公式（`agent_stats`、`score`） |

**仍是 Go 硬编码**（脚本改不动）：回合管线与事件顺序、延迟任务调度、世界生成
（地堡/楼层/居民/队列初始化）、派系聚类算法、NPC 决策（`internal/engine/npc.go`）、
存档读写。每回合系统事件与结局判定走 JS 主路径，Go 内保留同语义降级副本
（JS 缺失/报错时兜底，`runEventMechanic` 分发）——**改任一侧规则必须同步另一侧**。

## 运行与构建

前置：Go（见 `go.mod`）、Node 18+、[Wails CLI v2](https://wails.io/docs/gettingstarted/installation)；
Linux 需要 `libgtk-3-dev`、`libwebkit2gtk-4.1-dev`（SQLite 驱动走 CGO）。

```bash
wails dev          # 开发模式
wails build        # 产物: build/bin/silo40（前端已内嵌）
go test ./...      # 测试（引擎/服务/调试 API）
```

### 可试玩构建

```bash
./start-playable.sh
```

普通开局中，你是 **40 号派出的特工**，不是 40 号内部居民。世界只有三个模拟实体：随机分配的驻地（2–17 号）、Silo 1、Silo 18；40 号只作为叙事中的派出方存在。行动由 `events/action_gates.json` 按剧情节点、阶段、属性和关系逐步解锁，开局不再平铺全部按钮；非灾难结局由玩家、驻地派系、关键人物与 Silo 1 加权分流，配置参数集中且尚待实测调整。Silo 18 只运行原著链条且不可干预。详见 [三分钟试玩说明](PLAYTEST.md) 与 [已知缺口](KNOWN_GAPS.md)。

存档为工作目录下 `silo40.db`，带版本标记（当前为 `model.CurrentSaveVersion = 8`）：

- **v8 当前档**：正常加载，恢复行动门控、多实体结局仲裁及 Mayor/W/Judicial 权威状态。
- **v7 旧档**：启动时经时间戳备份、单事务和回读校验升级；Mayor 个人正统性继承旧 `Silo.Legitimacy`，W 使用来源明确标注的保守初值 50。
- **v6 旧档**：缺少行动门控与多实体结局状态，显式拒绝并提示开新局；不伪迁移。
- **v5 旧身份模型档**：显式拒绝并提示“旧档基于旧身份模型，无法继续，请开新局”；不提供伪迁移，也不会静默加载。
- **无标记或 v1–v4 旧档**：启动链仍按带时间戳备份、单事务、回读校验与失败回滚升级到 v5，随后因上述身份语义断裂而明确拒绝；请保留备份并开新局。
- **版本高于当前要求**：显式报错拒绝加载，不会被静默改写。

## 调内容：哪些免编译，哪些要重编

二进制启动时优先读取**可执行文件同目录**的 `events/`（仅当不存在时才释放内嵌
副本，`ensureEventsExtracted`）。因此直接改 `build/bin/events/**` 再重启即可生效，
无需重编译：

- 免编译：事件数值/文案、触发条件、冷却、新增剧情/规则事件文件、行动成本与
  耗时（`events/mechanics/` 内的定义会覆写 Go 侧默认表）、`agent_stats`/`score`
  等公式数值、结局判定条件（在既有结局类型内）。
- 需要重编译：Go 引擎与管线（新事件类型/新变异类型/世界生成/NPC/派系聚类）、
  存档结构、前端 UI、新增结局类型（前端 `VictoryType` 联合类型需同步）。
  `events/action_gates.json` 也由嵌入副本同步到既有 events 目录，修改源文件后需重编；
  包内缺该文件会显式报错并 fail-closed，不会退回全动作开放。

## 确定性复现与模拟

全部随机性来自单一可注入随机源（`internal/engine/rng.go`），脚本侧
`randomFloat()/randomInt()` 同源。

- `SILO40_SEED=<n>`：新开局从主种子确定性派生每局种子（日志会打印）。
- `CreateGameRequest.seed`：显式指定单局种子（调试 API / 模拟器使用）。
- 种子随存档持久化（`Silo.RNGSeed`），读档后延续同一随机流。
- 同一种子 + 同一操作序列（不中断重开）复现同一局。

无头模拟（调平衡）：

```bash
SILO40_DEBUG_HTTP_ADDR=127.0.0.1:18080 ./build/bin/silo40 &   # 需要 GUI 环境
go run ./cmd/sim -addr http://127.0.0.1:18080 -games 50 -seed 7 -months 1200
```

输出每局结局与结局分布/平均时长/平均剩余人口。`-seed` 固定时整批可复现。

## LLM 辅助 NPC 决策（可选，默认关闭）

`SILO40_LLM_MODE=shadow|on` 启用（另见 `SILO40_LLM_PARTICIPATION` 参与度四档
off/events/key/full、`SILO40_LLM_BASE_URL`/`SILO40_LLM_API_KEY`/`SILO40_LLM_MODEL`）。
shadow 只记录不生效（轨迹逐位不变）；on 按参与度介入，决策与切换全部记入
`llm_decision_logs`，读档回放按日志还原，日志缺失/冲突显式失败。默认不设置任何
变量时行为与历史逐位一致。配置、端点（`/debug/game/llm/*`）与确定性不变量见
[references/llm-npc.md](references/llm-npc.md)。

`SILO40_LLM_WARMUP` 默认开启；`SILO40_LLM_PREFETCH` 默认关闭，避免全 miss 时后台求值仍产生真实模型调用成本。只有门槛月能提供约 30 秒以上的慢节奏求值窗口时才建议显式开启预取，或等将来改为批量请求后再评估。

## 文档

- [架构细节](references/architecture.md)
- [数值与平衡](references/balance.md)
- [调试 HTTP API](references/debug_api_guide.md)
- [事件总线](references/eventbus.md)
- [职业行动](references/profession-actions.md)
- [已实现功能](docs/features.md)
- [LLM NPC 决策](references/llm-npc.md)
- [公开仓库内容边界与扫描](references/public-content-policy.md)

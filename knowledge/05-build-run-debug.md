# 构建、运行与调试

## 1. 环境

- Go 版本以 `go.mod` 为准，当前为 1.26.4。
- Node.js 推荐与 CI 一致的 22；前端为 React 19、Vite 7、TypeScript。
- Wails CLI v2.13.0。
- Linux 桌面需要 GTK3、WebKitGTK 4.1 和 CGO；纯 headless 环境不能打开 Wails UI。
- 当前试玩产物只承诺 Linux amd64，Windows/macOS 未制作。

## 2. 开发、构建与测试

```bash
wails dev
wails build
```

产物默认为 `build/bin/silo40`。只构建前端：

```bash
cd frontend
npm ci
npm run build
```

`main.go` 嵌入 `frontend/dist`，干净 checkout 直接跑 Go 测试前需要先构建前端或执行 Wails build。

常规门槛：

```bash
go vet ./...
go build ./...
go test ./...
go test -race ./...
```

行动门控、加权结局和 Agent 专项：

```bash
go test ./internal/service -run 'TestActionGate|TestWeightedEnding|TestAgent' -v
go test ./internal/engine -run 'TestWeightedEnding|TestProfessionAction|TestAction' -v
go test ./internal/service -run 'TestContentGuard|TestRulesGroupIsLoaded' -v
```

## 3. 试玩模式

在仓库或完整试玩包中：

```bash
./start-playable.sh
```

脚本选择 `build/bin/silo40` 或根目录 `silo40`，也可用 `SILO40_BINARY` 覆盖。它会：

- 检查二进制旁 events 存在时必须有 `action_gates.json`，缺文件直接退出。
- 默认把数据库放到 `.playable-data/silo40.db`，可用 `SILO40_DATA_DIR` 改位置。
- 强制 `SILO40_LLM_MODE=off`。
- 默认把 debug 地址设为 `127.0.0.1:18092`，可被环境变量覆盖。

普通开发直接运行二进制时，数据库是当前工作目录下的 `silo40.db`；二进制旁的 events 决定优先加载哪份内容。

## 4. 普通 Debug HTTP

GUI 启动总会创建 debug server，未设置地址时默认 `127.0.0.1:18080`。接口无认证且 CORS 为 `*`，不要绑定公共地址。

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET | `/debug/health` | 健康检查 |
| GET | `/debug/game/active` | 是否有活跃局 |
| GET | `/debug/game/state` | 完整快照 |
| POST | `/debug/game/create` | 创建并替换当前局 |
| POST | `/debug/game/pass-time` | 同步推进月份 |
| POST | `/debug/game/action` | 同步执行行动并检查 gate |
| GET | `/debug/game/ending` | 结局叙事 |
| GET | `/debug/game/events?limit=N` | 世界事件历史 |
| GET | `/debug/game/profession-actions?profession=IT` | 职业行动清单 |
| GET | `/debug/game/profession-action?id=IT_COVER_UP` | 单个职业行动 |
| GET | `/debug/game/llm/status` | LLM 状态 |
| GET | `/debug/game/llm/decisions?limit=N` | LLM 决策日志 |
| POST | `/debug/game/llm/participation` | 切换 off/events/key/full |

创建普通三堡局的请求需要包含：

```json
{
  "agent_name": "Agent",
  "profession": "IT",
  "start_year": 122,
  "seed": 7,
  "include_silo1": true,
  "neighbor": {}
}
```

同步 HTTP 不暴露 Begin/ProcessNext 队列，也没有 SaveGame 路由；手动保存是 Wails `SaveGame` 方法。

## 5. 批量 sim

先启动带 debug HTTP 的 GUI，再运行：

```bash
go run ./cmd/sim \
  -addr http://127.0.0.1:18080 \
  -games 20 -seed 7 -months 200 -profession IT
```

`-seed S` 时逐局使用 S+i，模拟策略使用独立随机源。服务器只有一个活跃局，模拟期间不要手动操作同一进程。KI-1 修复前的批量结果已失真，不应与当前轨迹混用。

## 6. Agent 操作员模式

### 无 GUI 宿主

从包含 events 的仓库根目录运行：

```bash
go run ./cmd/agenthost -addr 127.0.0.1:18080 -db /tmp/silo40-agenthost.db
```

agenthost 从磁盘 fail-closed 加载内容和 action gate，挂普通 game 与 Agent surface，不挂 LLM surface。一个宿主同时只运行一个 Agent 会话。

### 路由与操作循环

| 方法 | 路径 | 作用 |
| --- | --- | --- |
| POST | `/debug/agent/start` | 新建局并启动会话；字段 seed/agent_name/profession/include_silo1/neighbor/timeout_ms |
| POST | `/debug/agent/step` | 推进到第一个 NPC 决策点或月末 |
| POST | `/debug/agent/choose` | 提交 decision_id、candidate_id、全候选 scores 和 rationale，继续到下一停车点 |
| GET | `/debug/agent/status` | phase、game_id、月数、记录数和待决策 ID |
| GET | `/debug/agent/export` | 导出 JSON；`?format=jsonl` 导出 NDJSON |
| POST | `/debug/agent/stop` | 中止会话，保留可导出记录 |

开始示例：

```bash
curl -X POST http://127.0.0.1:18080/debug/agent/start \
  -H 'Content-Type: application/json' \
  -d '{"seed":42,"agent_name":"Agent","profession":"Mechanical","include_silo1":true,"neighbor":true}'

curl -X POST http://127.0.0.1:18080/debug/agent/step
curl http://127.0.0.1:18080/debug/agent/export?format=jsonl
```

当 step 返回 decision 时，必须先用 choose 提交。scores 要覆盖返回的每个 candidate 且只出现一次，分值 0–10；candidate_id 是被执行选择，scores 是采样回归标签，不是让自由文本直接变成行动。

状态机为 idle → running ↔ awaiting_choice → idle，终态为 game_over/stopped/error。默认单决策等待 30 分钟，可用 `SILO40_AGENT_TIMEOUT_MS` 或 start 的 timeout_ms 调整；超时进入 error，本月按原权重收尾并记录 timeout_fallback。

记录含 prompt、candidates、executed_action、response 和 execution_status。只有 executed 是有效训练样本；execution_failed、timeout_fallback、aborted 供审计/丢弃。请求体上限 1 MiB。

## 7. 排查入口

- 缺 frontend/dist：先构建前端。
- 改事件未生效：检查二进制旁旧 events；action_gates.json 源改动需要重新构建嵌入副本。
- 行动按钮缺失：先查 World.UnlockedActions、ProgressStage、disabled_reason 和注册表，不要先改前端。
- Agent 卡住：查 status 是否 awaiting_choice，以及 decision_id/candidate scores 是否完整。
- 读档被拒绝：核对是否 v5/v6/高于 v7，不要删版本标记伪兼容。

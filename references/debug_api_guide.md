# Debug HTTP API 指南

> 本文档由 AI 维护 (project-source)，详细说明 Silo40 项目中用于调试的 HTTP API 镜像接口。

## 简介

为了方便开发调试，后端在保持 Wails 原生桥接的基础上，额外暴露了一套 HTTP 接口。这些接口与 Wails 暴露给前端的方法完全镜像，允许通过 `curl`、Postman 或 IDE 的 REST Client 插件直接调用，以验证业务逻辑、数值平衡或进行自动化测试。

## 配置

Debug HTTP 服务在应用启动时自动运行。

- **默认地址**: `127.0.0.1:18080`
- **环境变量**: 可通过 `SILO40_DEBUG_HTTP_ADDR` 覆盖默认监听地址。
- **生命周期**: 随 Wails 应用同步启动与关闭。

---

## 接口说明

所有接口均返回 JSON 格式数据。

### 1. 系统状态
- **GET `/debug/health`**: 检查调试服务是否在线。
- **GET `/debug/game/active`**: 检查当前是否有活跃的游戏会话。

### 2. 游戏管理
- **GET `/debug/game/state`**: 获取当前地堡、特工及派生统计指标的完整快照。
- **POST `/debug/game/create`**: 初始化新游戏。
  - **Payload**:
    ```json
    {
      "silo_name": "ASSIGNED_STATION",
      "start_year": 122,
      "trait_ids": ["charismatic", "native"],
      "agent_name": "Operative",
      "profession": "Mechanical",
      "include_silo1": true,
      "neighbor": {}
    }
    ```

### 3. 时间与行动
- **POST `/debug/game/pass-time`**: 推进游戏时间。
  - **Payload**: `{"months": 1}`
- **POST `/debug/game/action`**: 执行特工行动。
  - **Payload**:
    ```json
    {
      "type": "GATHER_INFO",
      "target_dept": "Judicial",
      "cost": 10.0
    }
    ```
    *注：普通动作作用于当前驻地。行动快照以 `visibility=locked/unlocked/available` 表示门控状态；LOCKED 不下发，UNLOCKED 会带 `disabled_reason`，AVAILABLE 才能执行。执行 API 会再次拒绝未解锁或条件不足的动作。*
    *注：`STATION_OPERATION` 支持 `STATION_INTEL` / `STATION_INFILTRATE` / `STATION_DECEIVE` / `STATION_TRADE`；`SILO1_OPERATION` 只支持 `SILO1_INTEL` / `SILO1_DECEIVE`。40 和 18 没有行动类型。*
    *注：`cost` 字段不再被信任——服务端按行动定义重算权威成本并扣除（客户端值会被覆盖）；`PROFESSION_ACTION` 还需带 `profession_action`（行动 ID，如 `MECHANICAL_OVERHAUL`），且必须属于当前职业，否则显式拒绝。*

### 4. 叙事与历史
- **GET `/debug/game/ending`**: 获取当前的结局叙事（如果已结束）。
- **GET `/debug/game/events?limit=10`**: 获取最近发生的剧情事件日志。事件项现在会额外包含 `category`，用于区分 `history` / `special` / `crisis` / `player_action`。

### 5. 职业行动元数据
- **GET `/debug/game/profession-actions?profession=Mechanical`**: 获取特定职业的可用行动列表。
- **GET `/debug/game/profession-action?id=MECHANICAL_OVERHAUL`**: 获取单个行动的详细元数据。

### 6. Agent 操作员模式（外部 AI 真实跑局采样）
月度 tick 在每个 NPC 决策点暂停，等待外部 agent 提交选择；决策按数据管线契约
记录。完整协议见 `references/agent-operator.md`；无头宿主为 `cmd/agenthost`。

- **POST `/debug/agent/start`**: 开新一局并绑定 agent 策略。
  - **Payload**: `{"seed":7,"silo_name":"...","agent_name":"...","profession":"IT","include_silo1":false,"neighbor":false,"timeout_ms":0}`
  - `timeout_ms=0` 时用 `SILO40_AGENT_TIMEOUT_MS`（缺省 30 分钟/决策）。
- **POST `/debug/agent/step`**: 推进一个月；响应 `{"status":"decision","decision":{...}}`
  （遇到决策点即返回）或月末总结 `{"status":"month_end"|"game_over",...}`。
- **POST `/debug/agent/choose`**: 提交当前决策点的选择。
  - **Payload**: `{"decision_id":"...","candidate_id":0,"scores":[{"candidate_id":0,"score":8.0}],"rationale":"..."}`
  - `scores` 必须恰好覆盖全部候选、取值 0-10；违规返回 400 且决策点保持不变。
- **GET `/debug/agent/status`**: 会话阶段 / 月数 / 已记录决策数 / 待决决策 ID。
- **GET `/debug/agent/export`**: 导出决策记录（JSON 对象）；`?format=jsonl`
  返回每行一条的管线就绪 JSONL。
- **POST `/debug/agent/stop`**: 结束会话（待决决策记为 aborted，本月以原权重收尾）。

*注：agent 会话活跃期间 `engine.NpcPolicy` 指向会话而非 LLM 顾问；会话只支持
单局串行，终局/stop 后才能 start 下一局。*

---

## 示例集合 (cURL)

### 初始化游戏
```bash
curl -X POST http://127.0.0.1:18080/debug/game/create \
  -H "Content-Type: application/json" \
  -d '{
    "silo_name": "ASSIGNED_STATION",
    "start_year": 122,
    "trait_ids": ["leak"],
    "agent_name": "DebugAgent",
    "profession": "IT",
    "include_silo1": true,
    "neighbor": {}
  }'
```

### 推进 6 个月时间
```bash
curl -X POST http://127.0.0.1:18080/debug/game/pass-time \
  -H "Content-Type: application/json" \
  -d '{"months": 6}'
```

### 获取当前状态并保存到文件
```bash
curl http://127.0.0.1:18080/debug/game/state > current_state.json
```

---

## 开发者工具推荐

建议在 VS Code 中安装 **REST Client** 插件，并创建一个 `debug.http` 文件，内容如下：

```http
@baseUrl = http://127.0.0.1:18080

### 健康检查
GET {{baseUrl}}/debug/health

### 获取状态
GET {{baseUrl}}/debug/game/state

### 推进时间
POST {{baseUrl}}/debug/game/pass-time
Content-Type: application/json

{
    "months": 1
}

### 执行动作
POST {{baseUrl}}/debug/game/action
Content-Type: application/json

{
    "type": "GATHER_INFO",
    "target_dept": "IT",
    "cost": 10
}
```

## 注意事项
1. **单会话限制**: 当前后端逻辑仅支持单活跃游戏会话，`CreateGame` 会覆盖当前正在进行的进程。
2. **跨域控制**: 调试服务已开启宽容的 CORS 配置（`Access-Control-Allow-Origin: *`），方便浏览器调试工具直接调用。
3. **性能影响**: 调试服务非常轻量，对主应用性能几乎无影响。

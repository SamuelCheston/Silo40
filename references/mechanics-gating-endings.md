# 行动门控与多实体结局：机制底座设计

> 本文档由 AI 维护 (project-source)。本轮只实现可运行骨架；所有权重与阈值均为 **TODO 待实测调整**，不做数值平衡、不跑批量对局、不改 Silo 18 正典时序。

## 1. 行动门控状态机

行动统一使用后端 `PlayerActionMeta` 的三态；前端不复制条件：

```text
LOCKED（未解锁，完全不下发）
  │  unlock 条件首次满足，ID 永久写入 World.UnlockedActions
  ▼
UNLOCKED（已解锁，下发但 Enabled=false，并带 DisabledReason）
  │  available 条件、现有目标条件与 AP 均满足
  ▼
AVAILABLE（可执行，Enabled=true）
```

- 解锁只向前，不因属性随后下降而重新隐藏；可用性每次快照实时计算。
- `BeginAction` 与 `ExecuteAction` 在扣 AP、排队或产生效果前再次校验，绕过 UI 也不能执行 LOCKED/UNLOCKED 行动。
- 首版真实内容只开放 `action_gather_info`，玩家初始只掌握本职业 2 个碎片。成功 Gather 完成 `player_action:action_gather_info` 并推进阶段；空转月份不会推进阶段。其余动作由剧情、阶段、属性或关系逐步出现。
- 未启用配置的旧机制单测保持原行为；生产内容和专门集成测试启用门控。

## 2. 内容声明形态

新增 `events/action_gates.json`，集中声明阶段与行动，不在各动作里散落身份/阈值 `if`：

```json
{
  "version": 1,
  "initial_stage": "arrival",
  "default_visibility": "locked",
  "stages": [
    { "id": "arrival", "when": {} },
    { "id": "first_signal", "when": { "event_completed": "player_action:action_gather_info" } }
  ],
  "actions": [
    {
      "id": "action_share_info",
      "unlock": { "stage_at_least": "first_signal" },
      "available": { "metric": "known_fragments", "min": 1 },
      "reason": "需要至少 1 个已知碎片"
    }
  ]
}
```

条件节点支持 `all` / `any` 递归组合，以及：

| 字段 | 含义 |
| --- | --- |
| `event_completed` | 驻地 `ContentEventState` 中指定剧情节点已完成 |
| `stage_at_least` | 当前阶段已达到声明数组中的指定阶段 |
| `metric` + `min`/`max` | 属性阈值，如 `political_prestige`、`propaganda_level`、`known_fragments` |
| `relation` + `min`/`max` | 玩家与指定部门的关系阈值，如 `relation:Mayor` |

阶段数组有序，`World.ProgressStage` 只前进；每次事件结算和快照前统一刷新。新增节点或动作只需增加声明，无需修改通用判定器。

## 3. 多实体加权结局

旧 JS/Go 结局阈值只负责触发终局检查；`DEATH`、人口灭绝、`AGENT_COMPROMISED` 等灾难性失败仍是硬护栏。对进入历史分流的非灾难结局，由 service 侧统一仲裁器覆盖为加权结果，Silo 18 不接入该钩子，正典节点和时序不变。

每个候选结局 `O ∈ {INFORMATION, TIME, REBELLION, EXCLUSIONIST}`：

```text
score(O) = 玩家贡献
         + Σ 驻地存活派系贡献（Influence / Cohesion / Stance）
         + Σ 驻地具名关键人物贡献（Alive / Influence / Ideologies / Ambition）
         + Silo 1 贡献（Alive / Legitimacy / Cohesion / Rebellion）
```

- 玩家基础权重最大，但不再单独决定结果。
- 派系与人物按稳定 ID 顺序累加；固定候选顺序解决同分，禁止 map 顺序影响结果。
- 仲裁记录 `VictoryStatus.Decision`，包含各主体贡献、候选总分、选中结果与配置版本，供 UI 和审计解释。
- 权重、系数、同分顺序与最低分阈值集中在 `internal/service/weighted_endings.go` 的单一配置结构，全部标注 **TODO(balance)**；本轮只证明玩家、派系、人物、Silo 1 都能实际改变分流。

## 4. 存档与验收边界

- `World` 新增 `ProgressStage` / `UnlockedActions`，`VictoryStatus` 新增 `Decision`，版本升为 **v7**。
- v6 不伪迁移，`Resume` 显式拒绝并提示开新局；v5 继续沿用旧身份模型拒绝提示。
- 单测覆盖：开局窄选择、剧情/阶段/属性/关系解锁、灰项原因、服务端绕过拒绝、解锁存读档、四类主体权重、同分确定性、硬失败不被覆盖。
- 只做定向 build/vet/test 与前端 build，不跑 AI 采样、批量对局或 parity 重建。

## 5. 骨架落地记录

- 生产启动在 `BootstrapContent` 加载 `action_gates.json`；事件扫描把该文件识别为配置而非剧情事件。已有 events 目录也会同步嵌入的注册表；缺文件时显式报错并 fail-closed，启动脚本同时检查包完整性。
- JS 结局 trigger 使用小写类型，仲裁入口统一转为大写；硬失败保持原状。
- 前端灰项允许选中以展示 `disabled_reason`，Execute 仍禁用；结局卡显示四类候选总分。
- Silo 1 的 N9 覆灭事实在 peer 同步中具有权威性；Silo 18 的节点、人物与触发时间未改。
- 旧 agent/LLM 机制测试不加载门控配置；专门集成测试显式启用正式 JSON，避免用删除断言适应新机制。

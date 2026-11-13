# 确定性约定与已知问题

## 1. 单 RNG 流与两个独立随机源

Gameplay 只有一个引擎级 RNG：Go 规则、NPC 加权选择和 goja 的 randomFloat/randomInt 共用。三堡不为每堡分片，复现依赖固定结算顺序。

种子优先级为 CreateGameRequest.seed → SILO40_SEED 主种子派生 → 默认随机。实际 seed 写入 Silo.RNGSeed 和 World.Seed。

有两个刻意独立的随机源：

- 驻地编号使用 `rand.New(seed ^ 0x40a17)`，从 2–17 选择，不消耗 gameplay RNG。
- cmd/sim 与 parity 的行动策略各自使用独立 policy RNG，不消耗游戏流。

Silo 18 正典节点、action gate 求值和情报观察不消耗 RNG。Agent 的 Force 选择仍执行官方加权抽取以保持 RNG 消耗位置，LLM prefetch 使用确定性 picker 而非真实流。

## 2. 当前月度确定性顺序

多堡每月固定为：延迟 staging → 非玩家 AI → Silo 18 正典 → 各堡本地相 → 跨堡队列 → Silo 1 裁决 → 时钟 → 驻地/Silo 1 operation → 各堡后置相。

约束包括：

- 所有逐堡循环按 SiloID 升序。
- 普通延迟任务按 delay#N 创建序号，再按目标堡发布。
- CrossQueue 按到期时间和 CrossSeq，目标按 SiloID。
- 内容定义按分组/文件/ID，派系和 Resident 仲裁按稳定 ID。
- 加权结局候选顺序固定，只有严格更高分才替换当前选择。
- Relations 浮点累加先排 profession ID 和 relation key，避免 Go map 顺序造成 ULP 漂移。

因此可复现契约是：**同代码与内容、同 seed、同操作序列、同控制模式，得到相同轨迹**。改变 KI-1 前后的执行语义、门控加载方式、LLM/Agent 控制或操作顺序，不属于同一基线。

## 3. Resume 的 RNG 事实

当前 `Resume` 仍执行 `rand.New(rand.NewSource(sessionSeed(s.silo)))`。存档保存原 seed，不保存 RNG 当前流位置，也没有重放消费计数。

所以读档会从原 seed 重建随机流，不能证明“中途读档与不中断运行是同一连续 RNG 轨迹”。v7 parity 在第 100 月显式 persist/Resume，是把这一现状纳入基线，而不是修复连续流问题。

这是按当前代码核实的结果；README 中“读档后延续同一随机流”不能解释成恢复流位置。

## 4. KI-1：NPC 目标丢失已修复

旧实现把 GATHER_INFO/SHARE_INFO/BUILD_CONNECTION 的目标只写进 message，动作本身缺 TargetDept/FragmentIds，导致 NPC 选中后必然执行失败。

当前 `scoreCandidates` 已把目标写入 AgentAction，ExecutionObserver 回填真实执行结果。决策侧 RNG 顺序保持，但修复后执行端会实际消耗碎片选择/接受率等随机数，因此旧轨迹从首个相关动作开始分岔。

修复前的 sim、v6 parity、LLM 对照和分析导出都不能当作当前分布依据；玩家行动不受该缺陷影响。

## 5. KI-2：v6 基线作废，v7 已重建

旧 `parity-v6-three-silo.json` 同时受 KI-1 和 v7 门控/结局变化影响，已删除，不再保留为有效基准。

当前基准为 `references/parity-v7-three-silo.json`：

- 种子 7–12，随机驻地 + Silo 1 + Silo 18。
- 包含 KI-1 修复、加权结局和第 100 月存读档。
- 双跑逐字节一致。
- SHA-256：`9f37dfa409a7efcf863819d6231de66042c6f8af3a4d9f742f8fa3e9196b7f84`（仅将绿地事件 ID 更新为 reconnaissance）。

注意：parity 按旧机制 fixture 不加载生产 action gate。它证明当前服务路径的确定性，不证明生产行动树或结局分布已平衡。

重建命令使用绝对输出路径：

```bash
SILO40_PARITY_DUMP=/absolute/path/parity-v7-three-silo.json \
  go test ./internal/service -run '^TestParityHarnessDump$' -v
```

行为中性改动应与现有文件 diff；有意改行为时先说明差异并重建受控基线，不保留两份都宣称有效的基准。

## 6. 其他当前限制

### 普通延迟与 pending 操作读档不完整

delayTasks、delayCounter、pendingOperation 和未处理 EventBus 队列没有完整持久化。手动保存只接受完整结算点，但重启仍会丢未到期普通延迟任务；跨堡 CrossQueue 则在 World 中持久化。

### WorldPacing 仍未被结算消费

字段会 Normalize、保存并进入快照，但当前 Go/JS 结算没有读取 ResourceRate/IdeologyRate/PoliticalRate/ActionRate/DemographyRate。修改它们不会改变实际速率；这是代码现状，不是已完成的节奏功能。

### 结局数值未平衡

所有加权系数和最低分是 TODO(balance)，v7 parity 也不是分布验收。驻地 operation 仍是固定 delta 通道，没有长期渗透网络。

### 门控内容较窄

目前只有 arrival、first_signal、archive_awakened 三阶段和一组示例阈值。生产注册表与旧测试 fixture 的加载差异必须写在验证结论里。

### 正典与可见性边界

Silo 18 只能观察，N 节点不可被普通事件/行动改写；40 号没有实体。世界事件历史尚未对每条远程事件做完整权限裁剪，精确目标计量主要来自情报行动与回执。

### Debug 与采样边界

Debug HTTP 默认开启、无认证、CORS 宽松，只能绑定可信本机地址。Agent 采样接口已可用不代表可以大规模铺量；机制继续变化会使样本作废，当前应先做 smoke。

## 7. 修改确定性相关代码的纪律

- 不新增 gameplay 私有随机源，不让 map 遍历影响随机消费或浮点累加顺序。
- 不随意改变堡序、阶段顺序、候选顺序、目标顺序和 RNG 调用次数。
- 不用跳过门控、删除断言或 continue-on-error 换取 parity 绿灯。
- 行为中性重构先同 seed 对照；KI 类执行语义修复要明确作废旧数据。
- 修复续跑时需要同时设计 RNG 状态、普通延迟任务和 pending/EventBus，而不是只更新“支持续玩”的文案。

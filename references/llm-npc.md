# LLM 辅助 NPC 决策（可选功能）

> 本文档由 AI 维护 (project-source)。`docs/` 为外部来源，不予修改。
> 设计研究全文：`~/.trae-cn/assistant/2026-11-12/silo40-llm-npc-research/outputs/report.md`（会话产物，含实测数据与推导）。

## 状态

- 已实现：`shadow` 模式（只记录不生效）、参与度四档、双通道输出（候选打分 + 意愿系数）、
  决策日志与回放、中途切换、审计端点、无 key 离线链路（确定性 fake client）。
- `on` 模式**已放行**（2026-11-13，作者批准）：默认参与度 `key`、超时 8s；
  v3 软锚点提示词 + 修订后规避口径（见"放行门槛"）。验证：影子门槛 288 点 ×3 轮、
  真实对局 10 off + 10 on 种子配对（结局逐种子同型）、key 档日志回放逐位一致。
  详见研究报告 §13/§14。
- 默认完全关闭：不设置任何 `SILO40_LLM_*` 变量时，零代码路径变化、零成本、契约原样成立。

## 配置（环境变量，作者可见）

| 变量 | 默认 | 说明 |
| --- | --- | --- |
| `SILO40_LLM_MODE` | `off` | `off` / `shadow` / `on`。shadow=记录不生效；on=按参与度生效 |
| `SILO40_LLM_PARTICIPATION` | `key` | 参与度档位：`off` / `events` / `key` / `full`（见下）。缺省 `key` 为 2026-11-13 放行决定；显式 `off` 可恢复"启用但不介入" |
| `SILO40_LLM_BASE_URL` | OpenAI 官方 | 任何 OpenAI 兼容 `/v1` 端点（Ollama/vLLM/llama.cpp） |
| `SILO40_LLM_API_KEY` | 空 | 只从环境/密钥文件读取；不入库、不入日志正文 |
| `SILO40_LLM_KEY_FILE` | 空 | 可选密钥文件路径 |
| `SILO40_LLM_MODEL` | 空 | 模型名；写入每条决策日志（回放审计） |
| `SILO40_LLM_EVENT_WINDOW` | `1` | events 档窗口（ONCE 事件触发月 + 其后 N 个月） |
| `SILO40_LLM_TIMEOUT_MS` | `8000` | mode=on 单次调用超时预算（毫秒）；超时该点回退原算法。缺省按本地端点标定（qwen3.5:4b 常态 2.5-3s；冷加载首调可能超时回退一次，属预期） |
| `SILO40_LLM_REASONING_EFFORT` | 空（不发送） | 非空时作为 `reasoning_effort` 随 `/v1` 请求发送。thinking 模型（qwen3.x 等）**必须设 `none`**，否则每次调用先烧数千 token 推理再出 JSON |
| `SILO40_LLM_WARMUP` | 开 | 启动时一次廉价探测调用预载模型，消除首个门槛决策的冷加载超时回退；结果记日志（`LLM warmup ok/failed`）并可经 advisor 审计 |
| `SILO40_LLM_PREFETCH` | **关** | mode=on 决策预取：月度 tick 结束后为下月门槛内决策点后台逐个求值，决策时只读缓存、零阻塞；未命中回退原权重（日志 `validation=prefetch_miss`），命中记 `source=llm_prefetch`。默认关闭以避免全 miss 时仍产生真实模型调用成本 |

无 key 且无 base_url 时自动使用确定性 fake client（日志会注明），用于离线验证链路；
这不是"假装 LLM 生效"——shadow 模式本就只记录，而 on 模式下 fake 只是可复现的占位模型。

## 预热与预取（mode=on 的零卡顿路径）

- **预热（warmup）**：advisor 建立时异步发一次廉价探测调用，把模型载入端点。
  消除"每段冷启动首个门槛决策必超时回退一次"的现象（20 局实测曾 7/10 局出现）。
  成本：每次服务启动 1 次调用（<50 tokens）。失败只记日志，不影响游戏。
- **预取（prefetch）**：每月 tick 结束后（`ObserveMonth`），若下月门槛开启，
  用**纯状态预测**（`engine.PredictCandidates`，不消耗 rng）枚举全部潜在决策者
  的候选集，后台 worker 逐演员求值入缓存；决策时刻只读缓存、**零阻塞**，
  未命中（预测与真实候选的动作类型序列不符、或后台未完成）⇒ 原权重 +
  `prefetch_miss` 审计。命中条目的日志 source=`llm_prefetch`。
  - 预测候选的目标部门由引擎在真实决策时用 rng 选出，预取提示词里的目标是
    确定性占位（首个合法目标），打分按类型序列指纹对齐——建议质量略低于
    同步调用（看不到真实目标），换取 tick 零等待。
  - **成本特征**：预取覆盖门槛月内全部潜在决策者（~9-14 人），而实际尝试决策
    的只有 ~0.9 人/月，故调用量约为同步模式的 10 倍（key 档 ~500 次/局、
    ~550k tokens/局；本地端点费用为 0，托管端点慎用）。命中率取决于真实节奏：
    玩家每轮间隔 ≥ 后台求值总时长时接近全命中；快速模拟（sim/harness）下
    几乎全 miss ⇒ 等价原算法。批量跑对照实验建议 `SILO40_LLM_PREFETCH=off`。
  - **默认关闭与启用节奏**：实测快节奏下预取全 miss、追不上真实决策，后台调用仍会产生真实费用，因此默认 `off`。仅当人工操作让每个门槛月约有 **30 秒以上**的后台求值窗口时，才考虑显式设 `SILO40_LLM_PREFETCH=on`；或等将来改为真正的批量请求、显著降低逐演员调用成本后再启用。sim/harness 与托管计费端点不建议开。预热 `SILO40_LLM_WARMUP` 仍默认开启，二者互不影响。
  - 确定性不变：预取不碰 rng，日志回放与预取开关无关（回放只读 `final_choice`）。

## 参与度四档（玩家可调、可中途切换）

档位单调嵌套 `off ⊂ events ⊂ key ⊂ full`，定义全部锚定现有概念：

- **特殊事件** = `category ∈ {special, crisis, history}` 且 `fire_mode = ONCE` 的事件
  （`StoryEventLog.Category` + `ContentEventDefinition.FireMode`）。repeatable 事件实测
  ~1.18 次/月，若计入则 events 档≈full，故只计 ONCE（每局至多 16 次）。
- **关键节点条款**（任一成立即关键月）：
  - `rebellion > 0.7`（core_systems.js 现有 isRebelling 态）
  - `safeguard_risk ≥ 0.5`（DEATH 触发器 1.0 之半）
  - `cohesion < 0.4`（core_systems.js 现有激进化阈值）
  - `agent.SuspicionLevel ≥ 0.5`（AGENT_COMPROMISED 触发器 1.0 之半）
  - 存在未公开且有代表的派系（PUBLICIZE_FACTION 决策点在场）
  - 派系签名/公开集合当月变化
  - （恐慌类阈值实测 75%+ 月份成立，属常态，默认不计）

| 档位 | 介入点 | 实测月份覆盖 | 复现要求 |
| --- | --- | --- | --- |
| off | 无 | 0% | 契约原样 |
| events | ONCE 事件窗口内 | ~24%（窗口1） | 全程决策日志 |
| key | events ∪ 关键条款 | ~48% | 全程决策日志 |
| full | 全部 | 100% | 全程决策日志 |

切换：`POST /debug/game/llm/participation {"level":"key"}` 或 Wails `SetLlmParticipation`。
切换以 `kind=config` 条目记入决策日志；回放按日志还原切换点，中途改档不导致回放分叉。

## 双通道输出

- **通道 A 候选打分**：每个 `candidate_id` 返回 0-10 分 → 映射为权重乘数 [0.5, 2.0]。
- **通道 B 意愿系数**：按动作类型返回倍数因子（`gather_info` / `share_info` /
  `build_connection` / `conduct_propaganda` / `incite_rebellion` / `publicize_faction` /
  `profession_action`），clamp [0.5, 2.0]，作用于基础权重。
- 叠加：`final = base × factor[type] × score_mult`，合成乘数夹 [0.25, 4.0]。
  两通道缺省 ⇒ 逐位等于原算法。目标部门选择仍由引擎 rng 完成，LLM 不能指定"对谁"。
- 校验失败（schema/候选集不符）⇒ 该点回退原权重并记 `validation` 字段；不产生假结果。

## 决策日志（`llm_decision_logs` 表，append-only）

- `kind=decision`：seq、游戏月、actor、候选集 JSON、两通道原始值、最终选择、
  source（original/llm/shadow/replay）、档位、**模型标识**、prompt 哈希、校验结果、延迟。
- `kind=config`：参与度/模型切换事件。
- 不参与 `persist()` 的整图替换；`CreateGame` 开启新时间线时清空（旧局日志随旧图作废）。
- **回放**：Resume 载入日志，按"游戏月 + actor 顺序 + 候选集指纹"匹配，命中即
  `Force` 记录的选择（加权抽样仍消耗同一次 rng，流次序不变）；日志断序/指纹冲突 ⇒
  显式失败（sticky error，PassTime 直接报错），绝不静默降级。
- 换模型 = 换决策分布：每条日志带模型标识；回放不重跑模型，只读记录，故既有存档
  复现不受模型变更影响；审计上按模型标识切分段。

## 确定性不变量

1. `SILO40_LLM_MODE=off`（默认）：零代码路径差异，锁定测试原样通过。
2. 任何模式下引擎 rng 流不被 LLM 层触碰（建议只读不写；Force 也先消耗抽样）。
3. `shadow` 对轨迹零影响（有锁定测试：shadow 与 off 逐位 DeepEqual）。
4. `on` 的可复现性由决策日志承担：同档同日志回放逐位一致（有锁定测试：非确定
   客户端下 live 与 replay DeepEqual）。
5. `cmd/sim` / parity harness / 确定性锁定测试不受影响（默认 off；日志表为空时零开销）。

## 降级顺序（全部记审计日志）

请求失败重试 1 次 → 该点原算法 → 连续失败熔断（sticky error 之外的请求级失败
当前实现为逐点回退原算法）→ schema 不合 ⇒ 原权重 → 配置缺失/解析失败 ⇒ 整体
回退 off 并显式打日志。

## 放行门槛（shadow → on）

≥200 个影子决策点，且：

1. 响应校验通过率 ≥99%（`ok_repaired` 计入）；
2. 与现算法众数一致率 ≥50%（LLM top-1 == 引擎原权重 argmax）；
3. **规避检查（2026-11-13 修订，替代旧"每类 top-1 >1%"）**：
   - **V1 压制下限**：候选出现 ≥50 次的每个动作类型，平均分数乘数 ≥0.7
     （乘数 = 0.5 + score/10×1.5）且分数 std ≥0.5（情境敏感，非常数惩罚）；
   - **V2 混合接近度**：LLM top-1 分布与引擎实际采样分布的总变差距离
     TVD ≤ 0.40（TVD = ½Σ|p−q|）。
   - **修订理由**：引擎自身 oracle（base_weight argmax）为 GATHER 95.8% +
     BUILD 4.2%、其余五类全 0——propaganda/incite 基础权重恒为 0.8，几乎永不
     成为 argmax。旧判据"每类 top-1 >1%"与众数门槛 ≥50% 在该权重结构下
     **数学互斥**（任何与引擎足够对齐的模型必然"规避"5/7 类型）。实测：
     无锚点 TVD=0.54（类型偏执，✗）、硬锚点回声 TVD=0.64（✗）、
     软锚点 v3 TVD=0.32（✓）。V1 保"无类型被压到不可达"，V2 保"偏好不过度集中"。
4. 确定性回放逐位一致；预取/延迟在预算内。

影子统计从 `GET /debug/game/llm/decisions` 读取（`source=shadow` 条目含
`shadow_choice` 与 `scores_json`）。

**实测进展（2026-11-12/13，`qwen3.5:4b`，同 3 种子 ×288 点 ×3 轮）**：
- v1（无锚点）：合规 99.65%，但 LLM 按类型贴标签（职业行动 top-1 74%），不放行。
- v2（硬锚点 ±3）：过度对齐成回声（众数一致率 99.3%、top-1 单一化），废弃。
- **v3（软锚点，当前代码）**：合规 100%、众数一致率 62.9%、类型偏置显著下降、
  确定性回放通过。规避项字面仍不过（3 类 top-1 <1%），但已证明该门槛与
  众数门槛在本候选权重结构下数学互斥（引擎 oracle 自身 95.8% GATHER）。
- **已放行**（作者 2026-11-13 批准）：规避口径按上述 V1/V2 修订，`on` 默认 key 档；
  20 局真实对照与 key 档回放验证见研究报告 §14。

## Debug 端点

- `GET /debug/game/llm/status` — 模式/档位/日志条数/模型/错误
- `GET /debug/game/llm/decisions?limit=N` — 审计日志（新→旧）
- `POST /debug/game/llm/participation {"level":"..."}` — 中途切换
- 未启用 LLM 时三者返回 503。

## 代码落点

| 位置 | 职责 |
| --- | --- |
| `internal/engine/npc.go` | `NpcPolicy` 接口、`NpcAdvice`（双通道 + Force）、clamp |
| `internal/engine/engine.go` | `GameEngine.NpcPolicy` 注入位（`RunNpcTurn`） |
| `internal/llm/` | 档位/模式解析、响应校验、HTTP/fake 客户端、日志表、Advisor |
| `internal/service/game_service.go` | env 装配、tick 观测、Resume 回放、切换 API |
| `internal/api/debug_http.go` | `/debug/game/llm/*`（可选 `LlmAPI` 面） |

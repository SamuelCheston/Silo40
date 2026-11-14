# Silo40 信息碎片：共享视角模板、证据组合与有限 LLM 变体 · 与现有及既定机制的衔接

> 拆分自 [Silo40 信息碎片：共享视角模板、证据组合与有限 LLM 变体](../11-information-fragments-design.md)；目录、章节映射与出处见该索引。

## 6. 与现有及既定机制的衔接

### 6.1 连续确信度 `B` 与版本 `π`

- `Observation` 是未来 noisy-OR 更新的输入，`base_reliability/independence_key` 决定证据强度与重复折扣。
- 卡片正文是观察材料的呈现，不是 `B` 的存储位置；同一 `observation_id` 只结算一次。
- 高 `B` 只表示“事实发生较可信”；责任归属仍由 `π` 和材料 likelihood 决定。
- 第一阶段没有 `B/π` 时只展示来源类型和基础可靠性，不提前显示虚假的连续确信。

### 6.2 部门分层

- 现有 `KnownFragments` 继续承担拥有与计数；未来部门 × 阶层的 `BeliefState` 另行保存。
- 预写模板的 `tier` 是信息源层级，不等同于部门 `ClassType`，也不伪造当前人口里不存在的具体 NPC。
- 分享时传递结构化 Observation 及其版本 likelihood；不能只把一段 LLM 文本交给接收方重新理解。
- 玩家私下获得不会自动改变部门或阶层 belief，仍必须经过分享/公开。

### 6.3 Covenant / Protocol / Directive 条款

必须区分三类材料，不能统称“Protocol 条款”：

- **Covenant**：堡内公开规则，普通居民可能知道梗概，正式条款可成为公开法律依据。
- **Protocol**：处置与应急手册，知识访问受限；片段可以说明应当怎样处置，但持有它不等于已经发生违法。
- **Directive / Safeguard 材料**：更高层级的终止机制信息，访问与公开核验更严格。

每个 `legal_ref` 包含 `document/clause_id/applicability/public_access/source_version`。证据组合遵循“事实 + 责任 + 适用条款”，条款本身不替代事实。条款注册表只保存项目原创机制文本，不拼接外部来源或搬运长段原文。

### 6.4 阴谋曝光 `E` 与公开证据 `Q`

```text
获得 Observation → 玩家 B/π
分享/公开 → 部门×阶层及具名人物 B/π
部门 belief → conspiracy_exposure E
Judicial 可核验公开材料 + 覆盖 → Q
```

- `E` 只读受众 belief，不存在 `E += player_B`，也不能因 LLM 文案更有说服力而增加。
- 普通事故、档案清洗或传闻不自动形成“IT 幕后掌权”归因；该版本必须连接到 **IT 主任直接调度 Judicial 法警的证据链**。只读到 Protocol 也不足以证明当前 IT 正在操纵 Judicial。
- `Q` 还受 `public_provenance/public_coverage` 约束；私人档案、传闻和 LLM 修辞不能绕过公开核验。
- 同一 `Fact` 的重复材料不重复扣威势；`qE/qP` 在事实层去重。
- 本文的 `E` 是阴谋曝光，与驻地/上层控制方 秘密行动的 `operation_exposure` 完全分开，UI 和存档都不得混用。

### 6.5 当前获取、分享、INFORMATION 与确定性

- 第一阶段保留 28 个 ID、`GATHER_INFO` 随机取未知、分享接受率、unexplained knowledge、全部门 ≥5 的 INFORMATION 条件。
- 内容扩展只增加模板与展示，不暗中调整碎片概率、获取顺序、AP 或结局。
- LLM 不消费 gameplay RNG，不把模型输出作为模拟输入；同种子模拟 parity 不受文本开关影响。
- 文本存档与事实存档分开审计；旧 ID 可保留 alias，新结构不靠解析 `Mayor_4` 猜事实。


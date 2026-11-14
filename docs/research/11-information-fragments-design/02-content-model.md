# Silo40 信息碎片：共享视角模板、证据组合与有限 LLM 变体 · 内容模型：事实、解释、观察必须分开

> 拆分自 [Silo40 信息碎片：共享视角模板、证据组合与有限 LLM 变体](../11-information-fragments-design.md)；目录、章节映射与出处见该索引。

## 2. 内容模型：事实、解释、观察必须分开

### 2.1 三层对象

```text
Fact：发生了什么
  └─ Version：为什么会发生、谁应负责（可互斥）
       └─ Observation：谁在哪、经什么渠道、以什么口吻看到了哪一部分
            └─ Belief：某个玩家/群体目前有多相信事实与各版本
```

同一个 `Fact` 可以被多个部门、多个层级观察；同一个 `Observation` 可以强力支持“事情发生”，却只弱力支持“IT 主导”。这与 10 篇的 `B/π` 分离一致。

### 2.2 字段设计

| 对象 | 字段 | 含义与锁定规则 |
| --- | --- | --- |
| `Fact` | `fact_id` | 稳定语义 ID，不使用 Mayor_4 这类展示编号作为真相真源。 |
| | `canonical_event_key` | 指向实际事件或原创历史事实；事件已发生的时间从权威状态绑定，不能由文案猜。 |
| | `subject/object/location_scope` | 涉及部门、人物或设施及作用范围；固定时间线场景 正典人物只能引用权威链状态。 |
| | `occurred_at` | 实际发生时间或明确的历史时间区间；实例化后锁定。 |
| | `causal_edges` | 已确认的因果来源与后果；未知动机不得写成事实。 |
| | `content_version/source_ref` | 内容版本与项目内来源；公开库只登记原创机制与原创文案。 |
| | `qE/qP` | 未来进入 `E/Q` 的事实级权重；集中在事实注册表，不复制到每份文案。 |
| `Version` | `version_id/mutex_group` | 例如“必要安全措施 / 例行处置 / 蓄意隐瞒”可互斥；事实本身不在此层争真假。 |
| | `claim_causality/claim_actor` | 该版本主张的因果与责任，不等于权威事实。 |
| | `stance/attribution_to_IT` | 叙事立场及未来 `a_e,v` 归因参数。 |
| | `likelihood_by_channel` | 不同材料对各版本的支持程度，供未来 `π` 更新。 |
| `FragmentTemplate` | `template_id` | 共享模板 ID，不含 silo ID；现有 28 个字符串保留为兼容 alias。 |
| | `fact_id/version_id` | 明确这份材料陈述什么、采用哪个解释。 |
| | `department/tier/theme` | 三个正交分类维度，见 §3。 |
| | `observation/interpretation/blind_spot` | 作者验收字段：直接看到什么、如何解释、不知道什么；防止“同段换皮”。 |
| | `channel/document_form` | 传闻、签署证词、维修单、库存记录、正式档案、实物等。 |
| | `title/body_template/locked_claims` | 预写正文及不可改写的事实锚点。 |
| | `allowed_details` | 允许 LLM 选择并改写的非事实性细节池，例如纸张受潮、楼梯噪声；不得包含新事件。 |
| | `base_reliability/independence_key` | 渠道基础可靠性与独立来源分组；同一来源转述不能刷高确信。 |
| | `related_fact_ids/next_leads` | 与哪些事实咬合、下一步适合找什么渠道；只给方向，不直接剧透答案。 |
| | `legal_refs` | 指向 Covenant/Protocol/Directive 的条款 ID，不在正文里自由编造条号。 |
| | `evidence_form` | 私人猜疑、匿名传闻、公开证词、未核验记录、原始档案等；是否真正成为公开证据仍取决于发布与核验。 |
| `Observation` | `observation_id` | 一次获取的唯一 ID，也是未来观察去重键；重复查看不新建。 |
| | `template_id/silo_id/observer_id` | 共享模板在本局、本堡、本观察者下的实例化。 |
| | `acquired_at/acquisition_event_id` | 何时、由哪个行动或事件获得。 |
| | `source_role/source_location/channel` | 本局实际来源；没有权威楼层时只显示部门或 zone，不虚构地点。 |
| | `base_text/final_text` | 预写原文永远保留；`final_text` 是玩家实际看到的预写或 LLM 版本。 |
| | `render_status/model/prompt_hash/content_hash` | `prewritten/generated/fallback`、模型与模板审计信息；模型变化不改旧档文本。 |
| `BeliefState` | `observer/fact/B/π/observation_ids` | 连续确信、版本分布和已结算观察；不属于正文，不允许 LLM 修改。 |
| `EvidenceCase` | `fact/observations/legal_ref/actor_link` | 可组合证据的结构化引用，不复制正文。 |
| | `public_provenance/public_coverage` | 发布渠道、可核验性与实际受众覆盖；玩家私下 `B` 高不能替代它们。 |

**最小实现不必一次建齐所有表。** 第一阶段只需要 `FragmentTemplate + Observation` 的只读展示子集；`BeliefState/EvidenceCase` 随对应机制落地，避免先造一套没有消费方的数据结构。


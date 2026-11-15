# 结局事件链 · 十线在同一局里的互斥与交叉

> [返回总索引](../silo40-endings.md) · [机制底座](./10-chain-foundation.md) · [剧情侧交叉](./08-cross-lines.md)

剧情侧的交叉拆台见 [08 分册](./08-cross-lines.md)；本文只回答机制问题：十条线的节点在同一局里怎样共享状态、互相让路与互相拦截。

## 1. 问题树 → 机制映射

[08 分册 §1](./08-cross-lines.md) 的问题树在机制上这样落地：

| 问题树节点 | 机制读取 | 现状 |
| --- | --- | --- |
| 还有可确认的幸存社会吗 | 人口 ≤0、`safeguard_risk ≥ 1.0` → victory_check DEATH | 既有；逃出群体例外已实现（最小状态基础 `EscapedGroup`；完整逃离模型未实装，见 roadmap） |
| 它被格式化了吗 | `special:reset_completed` 事实键 | 已实现：仲裁 P0.2 读取 |
| 维存能力已不可逆崩溃 | `special:fracture_confirmed` 事实键 | 已实现：仲裁 P0.3 读取 |
| 玩家还能继续任务吗 | `suspicion_level ≥ 1.0` → victory_check AGENT_COMPROMISED | 既有 |
| 哪些收敛事实成立 | 各线完成键与六候选加权 | 已实现：P1 资格池＋P2 加权＋P3 同分顺序；权重系数为 TODO(balance) 占位 |

**顺序纪律**：P0 四项的检查顺序（寂堡→遗忘→断层→除名）已由仲裁实现：先读人口与逃出群体，再读 `reset_completed`，再读 `fracture_confirmed`，最后读封顶。内容层只负责把事实键生产出来。

**覆灭是结算动作**：覆灭锚点到达（相对 Y8）或结算条件满足即进仲裁；覆灭后不留内容环节，一切事实键必须在结算前成立。

## 2. 共享节点与完成键的跨线消费

同一个事件键被多条线读取时，读取含义必须写明：

| 共享键 | 生产者 | 消费者与含义 |
| --- | --- | --- |
| `player_action:action_share_info` | 主干 | 《共证》感染与复制计数、《公议》起草步骤、《遗忘》副本分发、《寂堡》反制知识跨界 |
| `special:archive_purge_order` 链 | 既有 | 《遗忘》缓慢绞杀路径、《共证》宿主威胁、《无人裁决》镇压压力 |
| `special:silo1_ultimatum` 链 | 既有 | 《寂堡》终止征兆、《越岭》窗口压缩、全线暴露风险 |
| `special:mayor_crisis_stage` | 《人治》线 | 同一危机也是《断层》的燃料与《遗忘》的掩护；危机事件本身只点火一次，各线分别读 |
| `player_action:reset_countermeasure_done` | 《遗忘》线 | `reset_execution` 守卫；《共证》宿主存续的间接保障 |
| `special:termination_channel_disrupted` | 《寂堡》线 | 只免疫单一终止渠道；不改变 `bombing_terminal` 的调度 |

规则：**一个键只记录事实，不记录立场**。同一份分享既可以是《共证》的复制也可以是《公议》的喂证；由后续节点的选择键区分路线归属，不在共享键里预写路线。

## 3. 互斥点：同一时刻只能成立一个的事实

| 互斥点 | 机制载体 | 裁决 |
| --- | --- | --- |
| Judicial 附着：**IT 还是约章** | `charter_lock_day` 与约章附着动力学 | 【裁决更新】司法部只附着两个对象：IT 或约章；市长不拥有归属关系，只有影响力。剧情上《公议》《人治》两条接触卡可以同时推进，约章锁定是单向事实；《人治》的成立不占用 Judicial 附着 |
| 覆灭前夕主导 | `pre_fall_choice` 三完成键 | 忠诚动员键成立即关闭《无人裁决》资格（叙事层）；实际快照判定已由机制底座实现（覆灭落地瞬间取人口加权思潮快照） |
| 终局主结局 | 统一仲裁 | 同分顺序已实现：INFORMATION→REBELLION→RULE_OF_LAW→TIME→PERSONAL_RULE→EXCLUSIONIST |

## 4. 交叉拦截：一条线的节点怎样拆另一条线

| 拦截方 | 被拦截方 | 机制路径 |
| --- | --- | --- |
| 《遗忘》`reset_execution` 结算 | 《共证》宿主、《公议》锁定日 | 清洗完成键先于制度成就；未完成的接触卡随接收者消失而显式失效（行动设计 15 §5.1：接触方失效按显式失败处理） |
| 《断层》`fracture_infrastructure` | 《越岭》路线 | 楼层稳定度损耗使 `advance_party_report` 走受损分支 |
| 《寂堡》`bombing_terminal` | 一切本地线 | 人口归零后所有本地节点失去受众；仅逃出群体例外（最小状态基础已落地，完整逃离模型未实装） |
| 《除名》封顶 | 玩家驱动的一切节点 | 封顶后玩家行动能力归零；已成立的本地事实仍可继续结算（尾声） |
| 覆灭结算时的封闭主导 | 《无人裁决》 | 结算瞬间快照为纯堡主导即排除 TIME 资格；覆灭后不留内容窗口，无"死后"内容节点 |

## 5. 密度与时序总账

十条线的新增节点合并计算密度预算（作者手册 §6）：

- ONCE 大事件：十条线合计约 20 个，绝大多数以不同阈值与完成键错峰；同一月并发的期望 <1。
- REPEATABLE：`replication_progress`（cd3）、`open_discourse_milestone`（cd6）、`normalcy_hardens`（cd12）、`identity_trace_accumulates`（cd2）——合计触发频率须控制在 ~0.55 次/月预算内，落地前跑 sim 核对。
- 延迟档：`charter_window_expiry`（+6）、`reset_omen_requisition`（+3）、`reset_execution`（+9）、`advance_party_report`（+4）。延迟链允许溢出，不强制收进窗口；但覆灭结算是终局时钟，需要成为终局事实的节点（`reset_completed`、`fracture_confirmed`、`bombing_terminal`）必须在结算前成立，剧情上必须发生的窗口内事件设兜底强制拉起。
- 叛乱预算：新增节点的净注入（+0.03/+0.04 级别的若干项与若干负 delta）不改变"无人值守终局 rebellion ≈ 0.5 < 1.0"的既有核对结论；落地时以同种子 sim 复核。

## 6. 落地顺序建议

1. 机制底座（G1–G11）已随 `dc6be5e` 落地：六候选仲裁、事实键读取、覆灭前夕快照、逃出群体最小状态、内容→Authority/cohort/safeguard 写入通路、出站信号日志、`first_signal` 迁移。
2. 内容节点群尚未编写。先落无额外依赖的：《深闭》惯性节点、《断层》三节点、《遗忘》六节点（守卫范式）。
3. 再落政治接触链：《公议》《人治》节点群；约章资格供给已有内容写入入口（`authority_evidence_delta / authority_public_fact / authority_pact_qualify`），具体供给事件待编写。
4. 《人治》资格读取与裁决口径的对齐（司法部两附着，见 16 分册缺口）属引擎事项，内容层先按裁决口径写节点。
5. 逃离模型与外溢准备工作本轮不做（roadmap）；依赖它们的节点只预留注释，不写成看似可运行的内容。

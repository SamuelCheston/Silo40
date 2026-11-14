# 结局事件链 · 十线在同一局里的互斥与交叉

> [返回总索引](../silo40-endings.md) · [机制底座](./10-chain-foundation.md) · [剧情侧交叉](./08-cross-lines.md)

剧情侧的交叉拆台见 [08 分册](./08-cross-lines.md)；本文只回答机制问题：十条线的节点在同一局里怎样共享状态、互相让路与互相拦截。

## 1. 问题树 → 机制映射

[08 分册 §1](./08-cross-lines.md) 的问题树在机制上这样落地：

| 问题树节点 | 机制读取 | 现状 |
| --- | --- | --- |
| 还有可确认的幸存社会吗 | 人口 ≤0、`safeguard_risk ≥ 1.0` → victory_check DEATH | 既有；"逃出群体存续"例外待 G5 |
| 它被格式化了吗 | `reset_completed` 事实键 | 键已设计（18 分册）；候选判定待 G1 |
| 维存能力已不可逆崩溃 | `fracture_confirmed` 事实键 | 键已设计（19 分册）；候选判定待 G1 |
| 玩家还能继续任务吗 | `suspicion_level ≥ 1.0` → victory_check AGENT_COMPROMISED | 既有 |
| 哪些收敛事实成立 | 各线完成键与既有候选加权 | 部分既有，部分待 G1/G2/G4/G5 |

**顺序纪律**：P0 四项的检查顺序（寂堡→遗忘→断层→除名）要求仲裁先读人口与逃出群体，再读 `reset_completed`，再读 `fracture_confirmed`，最后读封顶。这个顺序是引擎职责（G1 落地时必须保持），内容层只负责把事实键生产出来。

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
| Judicial 附着 | `charter_lock_day`（约章）与 `mayor_public_stand`＋附着动力学（人治） | 引擎附着状态互斥；剧情上两条接触卡可以同时存在，但锁定/附着只发生一次。先成立者写入，后到者按"资格已被占用"改道（转另一条线或关闭） |
| 覆灭前夕主导 | `pre_fall_choice` 三完成键 | 忠诚动员键成立即关闭《无人裁决》资格（叙事层）；实际快照判定待 G4 |
| 终局主结局 | 统一仲裁 | 同分顺序按结局体系 12：INFORMATION→REBELLION→RULE_OF_LAW→TIME→PERSONAL_RULE→EXCLUSIONIST。**注意**：现状代码顺序是 INFORMATION→TIME→REBELLION→EXCLUSIONIST，TIME 先于 REBELLION 与目标口径相反，属 G9 |

## 4. 交叉拦截：一条线的节点怎样拆另一条线

| 拦截方 | 被拦截方 | 机制路径 |
| --- | --- | --- |
| 《遗忘》`reset_execution` 结算 | 《共证》宿主、《公议》锁定日 | 清洗完成键先于制度成就；未完成的接触卡随接收者消失而显式失效（行动设计 15 §5.1：接触方失效按显式失败处理） |
| 《断层》`fracture_infrastructure` | 《越岭》路线 | 楼层稳定度损耗使 `advance_party_report` 走受损分支 |
| 《寂堡》`bombing_terminal` | 一切本地线 | 人口归零后所有本地节点失去受众；仅逃出群体例外（G5） |
| 《除名》封顶 | 玩家驱动的一切节点 | 封顶后玩家行动能力归零；已成立的本地事实仍可继续结算（尾声） |
| 《深闭》`post_fall_takeover` | 《无人裁决》 | 纯堡主导键成立即排除 TIME 资格 |

## 5. 密度与时序总账

十条线的新增节点合并计算密度预算（作者手册 §6）：

- ONCE 大事件：十条线合计约 20 个，绝大多数以不同阈值与完成键错峰；同一月并发的期望 <1。
- REPEATABLE：`replication_progress`（cd3）、`open_discourse_milestone`（cd6）、`normalcy_hardens`（cd12）、`identity_trace_accumulates`（cd2）——合计触发频率须控制在 ~0.55 次/月预算内，落地前跑 sim 核对。
- 延迟档：`charter_window_expiry`（+6）、`reset_omen_requisition`（+3）、`reset_execution`（+9）、`advance_party_report`（+4）——发起时点全部不晚于 ~Y128，落点在 Y130 前。
- 叛乱预算：新增节点的净注入（+0.03/+0.04 级别的若干项与若干负 delta）不改变"无人值守终局 rebellion ≈ 0.5 < 1.0"的既有核对结论；落地时以同种子 sim 复核。

## 6. 落地顺序建议

1. 先落无缺口部分：《深闭》两节点、《断层》三节点、《遗忘》六节点（守卫范式）、共同主干迁移（G10）。
2. 再落政治接触链：《公议》《人治》节点群（剧情层先行，资格供给等 G2）。
3. G1/G4/G5/G7/G11 属引擎与编排工程，按各自立项推进；事件链文档中的对应节点以注释预留，不提前写成看似可运行的内容。

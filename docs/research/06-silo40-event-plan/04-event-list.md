# Silo40 事件内容清单与实现计划 · 事件清单（批次 1–3 与叛乱预算核对）

> 拆分自 [Silo40 事件内容清单与实现计划](../06-silo40-event-plan.md)；目录、章节映射与出处见该索引。

## 4. 事件清单

命名与分组遵循 `events/README.md` 约定；"纯调度"= `script.canTrigger: () => false`、
只经 `schedule_event` 唤起。预估次数按基线轨迹估算（±30%），以 sim 回归为准。

### 批次 1：邻堡网络与记忆政治（5 个文件）

| id | 分组 | 时间子窗 | 主题 | 机制内核 | 触发 | 效果（持久杠杆） | 与既有事件的区别 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `neighbor_silo_silence` | histories | Y124M1 起（脚本里程碑） | 旧断联同盟里的邻堡逐个失联——当年 Silo 40 带头断联、随后在坍塌处置与清扫中毁灭，如今网络在它派出的特工眼前死去 | 脚本里程碑抬升历史包袱与叛乱，为 Y130 大结局铺弧 | `timestampAtOrAfter(124,1)`，ONCE | hb +0.05、rebellion +0.04、全部门 panic +0.04 | `archive_truth_broadcast` 是堡内档案真相；本事件是**外部网络**信号；`silo1_destroyed_signal` 是 Y130 的结果，本事件是其前兆 |
| `silo1_ultimatum` | histories | Y126M1 起（脚本里程碑） | Silo 1 经旧协议通道发来定向警告：终止信号测试 | 大威胁立即压顶 + 3 月后"适应与积怨"分流（受创泄压的镜像） | `timestampAtOrAfter(126,1)`，ONCE | panic +0.08、rebellion +0.06、`schedule_event(ultimatum_aftermath, 3)` | 既有事件无一涉及 Silo 1 的**主动施压**（失联是它的下场，不是它的动作） |
| `ultimatum_aftermath` | special | Y126M4 前后（纯调度） | 恐慌回落（适应），怨气沉淀 | 延迟档：威胁后恐慌泄压、叛乱续涨 | 纯调度 | panic −0.04、rebellion +0.05 | — |
| `archive_purge_order` | special | hb ≥0.12 后（基线 ~Y124M2：broadcast 0.08 + 邻堡 0.05） | IT 对觉醒发动二次档案清洗 | 历史包袱的"对抗性操作"：清洗立即降 hb，但埋下反弹 | `siloMetricGte("history_burden", 0.12)`，ONCE | hb −0.06、rebellion +0.05、panic +0.06、`schedule_event(purge_backlash, 4)` | `history_burden_awakened` 是 hb 的**读出**；本事件是 hb 的**反向操作**（抹除），构成拉锯 |
| `purge_backlash` | crisis | 清洗后 4 月（纯调度） | 被抹除的目录在楼梯间口口相传，抹除本身成为新的历史包袱 | 延迟反弹：清洗净收益≈0（−0.06 后 +0.04） | 纯调度 | rebellion +0.06、hb +0.04、panic +0.04 | 既有 crisis 均为资源型灾害，无记忆政治链 |

批次 1 大事件错峰：neighbor Y124M1 → purge ~Y124M2（条件触发，随局浮动）→ purge_backlash ~Y124M6 → ultimatum Y126M1 → aftermath ~Y126M4。

### 批次 2：资源与劳动（9 个文件）

| id | 分组 | 时间子窗 | 主题 | 机制内核 | 触发 | 效果 | 与既有事件的区别 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `energy_rolling_blackout` | special | Y123 起（Energy ≤300 常驻） | 分区轮流停电 | 资源阈值 → 恐慌 + 2 月后级联到食物链（Energy→Supplies 咬合） | `{silo_resource_lte Energy 300}`，REPEATABLE cd9 | panic +0.06、`schedule_event(blackout_cascade, 2)` | `water_leak` 是 Materials→Supplies 链；本事件是 **Energy→Supplies** 链，且带延迟级联 |
| `blackout_cascade` | crisis | 停电后 2 月（纯调度） | 水培照明离线，一批作物报废 | 延迟档：能源危机传导为食物损失 | 纯调度 | Supplies −250、panic +0.04 | — |
| `materials_cannibalization` | special | Y123 起（Materials ≤200 常驻） | 拆解底层闲置设施应急 | 即时回血 + 第 N 次质变（`trigger_count ≥5` 起出现拆解事故）+ 每次都埋 4 月后的住户事故 | `{silo_resource_lte Materials 200}`，REPEATABLE cd12 | Materials +300、`schedule_event(lower_deck_collapse, 4)`；apply 分支：count≥5 追加 panic +0.05、人口 −50 | 既有事件只**消耗**资源；本事件首次提供资源侧**应急手段**并附延迟代价（Frostpunk 式"同一手段第 N 次质变"） |
| `lower_deck_collapse` | crisis | 拆解后 4 月（纯调度） | 被拆解的住户区发生结构事故 | 延迟代价 | 纯调度 | 人口 −50、panic +0.05 | — |
| `labor_conscription` | special | 人口 ≤6000 后（基线 ~Y126） | 强制轮班：把丧亲者编入生产线 | 即时产出 + 3 月后怨气反弹；人口阈值门（后期内容锁） | `{silo_population_lte 6000}`，REPEATABLE cd9 | Supplies +800、`schedule_event(conscription_backlash, 3)` | 既有事件无人口阈值触发，也无"即时收益+延迟政治代价"结构 |
| `conscription_backlash` | crisis | 强制轮班后 3 月（纯调度） | 疲劳与怨气累积 | 延迟政治代价 | 纯调度 | rebellion +0.04、panic +0.05 | — |
| `geothermal_surge` | special | Y123M6 起（Energy ≤500 常驻） | 地热井季节性脉冲 | **馈赠通道**（正面事件与危机共用时钟）：缓解但不解决能源缺口 | `all(timestampAtOrAfter(123,6), silo_resource_lte Energy 500)`，REPEATABLE cd12 | Energy +600、panic −0.03 | 既有 8 事件全部是负面/中性；本事件是首个正面事件 |
| `hydroponics_mutant_strain` | special | Y123M6 起（Supplies ≤800） | 水培区选育出高产变异株 | 一次性大幅回血 + 6 月后副作用（延迟健康代价） | `all(timestampAtOrAfter(123,6), silo_resource_lte Supplies 800)`，ONCE | Supplies +1000、`schedule_event(strain_side_effects, 6)` | — |
| `strain_side_effects` | crisis | 变异株后 6 月（纯调度） | 长期食用的副作用显现 | 延迟代价 | 纯调度 | panic +0.04、rebellion +0.03 | — |

### 批次 3：社会与禁忌（4 个文件）

| id | 分组 | 时间子窗 | 主题 | 机制内核 | 触发 | 效果 | 与既有事件的区别 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `stairwell_rumor_cascade` | special | 中后期（≥5 部门恐慌 ≥0.6；基线 ~Y126） | 楼梯间流言网把事故串成"被隐瞒的真相" | 恐慌广度阈值 → 叛乱/历史包袱转化（恐慌→政治化的显性节点） | `professionMetricGteCount("panic_value", 0.6, 5)`，ONCE | rebellion +0.06、hb +0.04、panic +0.03 | `history_burden_awakened` 读 hb；本事件读**恐慌广度**，是恐慌轴的升级节点 |
| `grief_release_ritual` | special | 任一资源危机之后 | 灾后互助与悼念 | **危机→减压反向链**（`event_triggered` 的缓解用法）：灾难当月减压，社区把创伤转为团结 | `any(eventTriggered("crisis","food_poisoning"), eventTriggered("crisis","water_leak"), eventTriggered("crisis","blackout_cascade"))`，REPEATABLE cd9 | panic −0.10、rebellion −0.02 | 既有 crisis 链只有升级方向；本事件是首个**缓解方向**的链 |
| `public_cleaning_sentence` | special | Y124 后且 ≥2 部门 pro_foreign ≥0.55（基线 ~Y127–128） | 当局执行一次公开清洁判决以重申信条 | 禁忌的双面性：短期压恐慌（处刑即宣传），长期激进化；+5 月后目击者余波 | `all(timestampAtOrAfter(124,1), professionIdeologyGteCount("pro_foreign", 0.55, 2))`，ONCE | panic −0.06、rebellion +0.10、hb +0.04、`schedule_event(cleaning_witness_unrest, 5)` | `outside_signal` 是外部信号的被动接收；本事件是当局**主动执行禁忌**，且带思潮阈值门 |
| `cleaning_witness_unrest` | crisis | 公开清洁后 5 月（纯调度） | 目击者在底层复述"他走出去了" | 延迟政治代价 | 纯调度 | rebellion +0.06、panic +0.06、hb +0.03 | — |

### 叛乱预算核对（无人值守基线预估）

新事件合计注入 ≈ +0.11（通牒链）+0.11（清洗链）+0.06（流言）+0.16（清洁链）+0.12（轮班链，
3 次×0.04）+0.03（菌株）−0.10（互助，~5 次×0.02）≈ **+0.49**；叠加基线自然漂移（≈0），
终局 rebellion ≈ 0.5 < 1.0——**REBELLION 结局在无人值守下保持不可达**，仍需玩家动作推高
（是否要让它可达 → §6.3 待决项）。


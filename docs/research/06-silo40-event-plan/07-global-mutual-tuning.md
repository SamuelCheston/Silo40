# Silo40 事件内容清单与实现计划 · 全局互调轮

> 拆分自 [Silo40 事件内容清单与实现计划](../06-silo40-event-plan.md)；目录、章节映射与出处见该索引。

## 7. 全局互调轮（2026-11-12，作者授权改存量）

目标与验收：旧事件 + 18 新事件 + 规则事件互调成一张网；结局不再 100% 单一；触发分布更均匀；
同种子同局数前后对照。实现提交：`bfcbada`（存量互调）/ `18e3b47`（4 个新事件）/ `57d080f`（测试）。

### 7.1 量级标尺（本轮统一）

| 杠杆 | 小 | 中 | 大 | 备注 |
| --- | --- | --- | --- | --- |
| rebellion | +0.03 | +0.05 | +0.08 | 滚雪球门槛事件 +0.10 为唯一例外（终局机制） |
| panic | ±0.04 | ±0.06 | ±0.08 | 泄压渠道 −0.06~−0.10 |
| history_burden | ±0.03 | ±0.05 | +0.08（仅 broadcast） | |
| 资源 | ±300 | ±600 | ±1000 | |
| 人口 | −40 | −80 | （−0.5%/月 规则档） | |
| 稳定度 | ±0.04 | ±0.08 | +0.10（修复） | 仅 Lower 区承受衰减 |

派生指标（cohesion/legitimacy/dept_tension/class_fragmentation/部门思潮）的 delta 已全部
从存量事件中清除，换成持久杠杆。

### 7.2 因果网与母题分工

- 记忆政治：broadcast →(hb≥0.08) awakened →(hb≥0.12) purge →(+4月) backlash；neighbor_silence
  →(订阅) network_data_cache（碎片×2）。
- Silo 1 线：ultimatum(Y126) →(+3月) aftermath；destroyed(Y130) → fallout（去掉派生凝聚力门槛）。
- 能源线：blackout(cd12) →(+2月) cascade（食物−300 + Lower 稳定度−0.04）；geothermal 为馈赠变奏。
- 材料线：cannibalization(cd12，第5次质变) →(+4月) lower_deck_collapse（人口−80 + Lower −0.08）；
  water_leak 降级为被动衰变变奏（cd9，Lower −0.04）。
- 食物线：food_poisoning（门槛收紧 ≤2000、cd9、人口−40）；mutant_strain →(+6月) side_effects；
  supplies_shortage 规则改三档温水煮青蛙（<1500/<800/<200，cd4，只注恐慌不注叛乱）。
- 恐慌线：panic_storm(cd9) →(+3月) delayed_riot——暴乱重设计为"恐慌转化为政治压力"：
  rebellion +0.04、panic −0.06（泄压）、人口 −80。
- 互助线：grief_release（cd9，订阅 food_poisoning/water_leak/blackout_cascade，恐慌 −0.08、
  叛乱 −0.02）。
- 禁忌线：cleaning_sentence →(+5月) witness_unrest。
- 劳动线：conscription(cd9) →(+3月) backlash。
- 流言节点：rumor 增加前置 event_triggered(awakened)——先有叙事种子再有流言网。
- 终局门槛：underground_coordination（§6.3）。

### 7.3 sim 前后对照（同参数：20 局，种子 7–26，IT，months 200）

| 指标 | 互调前 | 互调后 | 说明 |
| --- | --- | --- | --- |
| 结局分布 | TIME 100% | **TIME 70% / REBELLION 30%** | 40 局复测 65%/35% |
| 平均终局人口 | 4480 | 3197 | 楼层衰减 + 暴乱/镇压伤亡 |
| 平均终局叛乱 | 0.541 | 0.980 | 门槛滚雪球后双峰收敛到 1.0 附近 |
| 平均结束月份 | 90.0 | 88.9 | 叛乱局提前结束（77–88 月） |
| supplies_shortage 触发 | 63.35/局 | 20.40/局 | 三档化 + cd4 |
| water_leak | 16.95 | 10.95 | cd9 + 门槛 |
| food_poisoning | 14.45 | 9.55 | cd9 + 门槛收紧 |
| 新增事件 | — | underground 2.85、salvage 7.0、cache 1.0、reinforcement 0.60 | reinforcement 依赖材料富余，稀疏属预期 |

无"从不触发"事件；`silo1_destroyed_signal`/`silo1_fallout` 降至 0.75/0.70 是叛乱局提前结束
（未到 Y130）的自然结果，非失效。INFORMATION 仍 0%（见 §6.6 说明）。


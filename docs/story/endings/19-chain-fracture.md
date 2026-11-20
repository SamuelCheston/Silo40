# 结局事件链 · `FRACTURE`《断层》

> [返回总索引](../silo40-endings.md) · [机制底座](./10-chain-foundation.md) · [剧情台阶](./05-fracture.md)

链条形态：**失控破坏 → 维生基础损耗 → 不可逆点**。本线大量复用既有叛乱与危机节点，新增节点只做一件事：把"维存崩溃"与"高叛乱"分开记录。

## 1. 节点表

| 节点 | 分组 | 触发 | 分岔与含义 | 效果（封闭集） | 后续 |
| --- | --- | --- | --- | --- | --- |
| 既有升级节点 | — | `underground_coordination`（rebellion≥0.62 且相对 Y4+）、`delayed_riot`、`rebellion_escalation`、各资源危机链 | 抉择 C-A 的机制背景：借乱局还是止损，决定叛乱预算怎么花 | 既有 | 本线入口 |
| `fracture_infrastructure` | crisis，ONCE | `all(siloMetricGte("rebellion", 0.8), timestampAtOrAfter(3,1))`（相对 Y3 起） | 第一次无人控制的重大破坏落到维生基础上 | `floorStabilityDelta("Lower", −0.25)`、`siloResourceDelta("Energy", −400)`、`panic_value +0.06`（sim 校准起点） | 不可逆点前置 |
| `fracture_point_of_no_return` | crisis，ONCE | `any(all({silo_resource_lte Energy 0}, siloMetricGte("rebellion", 0.9)), all({silo_population_lte 3000}, siloMetricGte("rebellion", 0.8)))`（裸对象触发器） | 不可逆点：维生基础跌破恢复线 | `emit ["special:fracture_confirmed"]` | 完成事实 |
| `fracture_confirmed` | special，纯调度 | `canTrigger: () => false` | 维存崩溃事实键：只被上一节点唤起 | `history_burden +0.03` | 供终局仲裁读取（已实现） |

## 2. 分支与改道

| 事实 | 去向 |
| --- | --- |
| 高叛乱但 `fracture_confirmed` 未成立 | 不得判《断层》；继续模拟，或按逃离事实转《越岭》 |
| `fracture_confirmed` 成立且无任何幸存群体 | 改判《寂堡》（P0.1 先于 P0.3，仲裁已按此实现） |
| 独立逃出群体已先成立 | 断裂的原驻地只作《越岭》背景（仲裁例外读取与完整存续结算均已随 Layer 2 落地） |
| 玩家在不可逆点前花杠杆修复 | 抉择 C-B 的止损分支：秩序在旧旗下恢复，局面转《人治》或《深闭》节点群 |

## 3. 超时与失败结算

- 本线无超时：它是状态累积线，节点由阈值点火。
- 爬出窗口在 `fracture_point_of_no_return` 之前：`grief_release_ritual`（既有减压链）、资源馈赠事件（`geothermal_surge` 等）与玩家止损行动都能压低叛乱与损耗。判定不读本线节点，读实际资源、人口与稳定度。

## 4. 与结局判定的对接

- 机制底座已实现 FRACTURE 候选：仲裁读取 `special:fracture_confirmed` 事实键（P0.3）；`victory_check` 对叛乱触发只**请求终局评估**，无逃离的高叛乱由仲裁落向《断层》或继续模拟，不再直接判《越岭》。
- 硬失败护栏不被《共证》《无人裁决》高分覆盖；先检查逃出群体（存续结算已随 Layer 2 落地，`escape_outside.go`）。

## 5. 能力缺口

G8（终局画面展示）。逃出群体例外判定已随完整逃离模型落地（Layer 2）。原 G1/G9 已随机制底座落地。

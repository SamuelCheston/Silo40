# Silo40 界面信息架构重构：五个行动域与三级地堡剖面 · 当前界面与数据事实

> 拆分自 [Silo40 界面信息架构重构：五个行动域与三级地堡剖面](../09-ui-information-architecture.md)；目录、章节映射与出处见该索引。

## 2. 当前界面与数据事实

### 2.1 已核实现状

| 现状事实 | 对设计的影响 |
| --- | --- |
| `available_actions` 已由后端下发，包含 `group`、`target_type`、`ap_cost`、`duration_months`、`enabled` 等字段。[PlayerActionMeta](../../../internal/model/models.go#L727-L740) | 不必在前端维护第二份行动成本表；五域应是后端元数据的增量扩展。 |
| 当前前端按 `group` 分组后，每组仍以两列、80px 高按钮平铺。[App.tsx](../../../frontend/src/App.tsx#L273-L297) [行动渲染](../../../frontend/src/App.tsx#L424-L548) | 问题不是完全没分组，而是所有分组同时展开、行动详情与目标选择又继续向下堆叠。 |
| 当前通用行动实际有 Gather Info、Share Info、Build Network、Incite Rebellion、Propaganda，以及满足条件时的 Publicize Faction。[common_actions.js](../../../events/mechanics/common_actions.js) | `common` 至少跨情报、关系、影响三个意图，不能直接改名成一个抽屉。 |
| 每个职业有两个专属行动，行动内容由 JS 机制定义。[profession_actions.js](../../../events/mechanics/profession_actions.js) | “职务”按解锁来源归类最稳定；职业行动的情报/影响效果用次级标签表示，不重复放进多个域。 |
| 普通开局把特工分配到随机驻地；驻地地图使用 `world.station_silo`，Silo 40 不作为可模拟驻地。[station.go](../../../internal/service/station.go#L11-L58) [App.tsx](../../../frontend/src/App.tsx#L350-L355) | 剖面主体应写成动态的“当前驻地”，不能硬编码成 40 号。 |
| 当前驻地有情报、策反、欺骗、交易四类 delayed operation；Silo 1 只开放情报与欺骗。Silo 18 仅观察，不可操作。[station_operations.go](../../../internal/service/station_operations.go#L18-L63) [silo1_operations.go](../../../internal/service/silo1_operations.go#L30-L70) [WorldPanel.tsx](../../../frontend/src/components/WorldPanel.tsx#L54-L58) | 两类 operation 共享一个秘密行动域，以“驻地 / Silo 1”chip 切换；Silo 18 不出现在行动目标中。 |
| 地堡有 144 层，楼层含 `Level`、`Function`、`Zone`、`Stability`、`Population`；初始化形成 11 个连续功能区。[initializer.go](../../../internal/engine/initializer.go#L136-L168) | 功能区可以完全由数据派生，不需要把 11 个布局组件写死。 |
| 当前 `BunkerMap` 只按 ELITE/COMMONER 两级展示部门卡片，没有使用 `silo.floors`。[BunkerMap.tsx](../../../frontend/src/components/BunkerMap.tsx#L23-L122) | 新剖面应替代“阶层卡片墙”，而不是在它下面再叠一张地图。 |
| 部门目前只有 `Zone`，没有权威 `primary_floor`；普通快照裁掉完整 `residents`。[architecture.md](../../../references/architecture.md#L48-L49) | 部门可先做 zone 级锚点；要精确落到单层，需要增量位置字段。 |
| 楼层初始化时 `Population: 0`。[initializer.go](../../../internal/engine/initializer.go#L154-L164) | 首版不得把该值解释为“本层无人”；应显示“人口未测绘”或干脆不显示逐层人口。 |
| 秘密行动 operation 的 `DurationMonths` 当前为 0，真实 1–2 月生效延迟写在 operation 定义与描述中。[silo1_operations.go](../../../internal/service/silo1_operations.go#L62-L72) | 新草稿面板需要独立的 `effect_delay_months`，不能把本地耗时误当秘密行动延迟。 |
| Linux/WebKitGTK 已针对渲染性能修过大面积阴影、泛化 transition 和 Canvas 每帧 blur。[08-linux-frontend-rendering-perf.md](../../../docs/research/08-linux-frontend-rendering-perf.md) | 新设计必须保持静态、定向、按需绘制；不应以“地图”为理由重新引入持续动画。 |

### 2.2 设计边界

**【设计提案】**本轮只重构信息架构与交互，不改变行动效果、AP、Elo、匹配、事件时间戳、派系聚类或结局语义；不增加主角走楼，不要求每个 NPC 实时显示，不把普通部门行动偷偷改成楼层目标。现有执行请求仍使用部门、资源或全堡目标，楼层只是选择这些目标的入口。


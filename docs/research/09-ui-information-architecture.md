# Silo40 界面信息架构重构：五个行动域与三级地堡剖面

> 日期：2026-11-13  
> 性质：研究与施工设计，不包含代码改动。  
> 目标：让玩家在有限空间内回答“此刻能做什么、代价是什么、会留下什么痕迹”，并把楼层、部门、人群变成可点击的行动锚点。

## 0. 阅读约定

本文使用四种标记，避免把参考游戏、现有实现和新设计混为一谈：

- **【现状事实】**：已在 Silo40 当前代码中核实。
- **【参考事实】**：已在参考游戏的官方资料、实机资料或操作指南中核实。
- **【设计提案】**：为 Silo40 提出的新界面方案，不声称参考游戏原本如此。
- **【数据缺口】**：现有快照不足以真实表达，施工时必须补只读数据或明确显示未知，不能由前端编造。

## 1. 结论摘要

采用一个稳定的三区外壳：**顶部特工状态、中央地堡剖面、右侧情境 Dock**。右侧 Dock 只有“行动”和“详情”两个页签；行动页内固定为五个域，不再把所有按钮同时平铺。

五个行动域固定为：

1. **情报**：获取、分享与处理信息。
2. **关系**：人脉、庇护、组织与派系关系。
3. **影响**：宣传、思潮、恐慌、秩序与叛乱。
4. **职务**：由当前职业或职业阶层解锁的专业行动。
5. **秘密行动**：当前驻地与 Silo 1 的 delayed operation；它们共享生效延迟、Exposure 与反间谍上下文，目标是域内筛选，不再各成一个顶级分类。

剖面固定为三级：**全堡功能区 → 单层 → Inspector**。第一级只显示由连续楼层派生的功能区，不同时绘制 144 张卡片；第二级显示一层的设施、部门与已知人群摘要；第三级在右侧 Dock 显示选中对象的详情和相关行动。

技术选择是 **DOM 为主、静态 SVG 为辅，不用 Canvas**。楼层、按钮、列表和 Inspector 都是可访问的 DOM；SVG 只用于楼梯井、连接线和少量静态状态符号。不引入地图引擎、粒子、常驻 NPC 动画、大阴影或 `transition: all`。

两个必须正视的数据边界：

- 现有 `group` 可以作为行动来源和解锁条件继续复用，但**不能直接作为五域分类字段**；它把三种驻地通用意图都放进 `common`，又把驻地与 Silo 1 的 operation 按通道拆成 `station`、`silo1`。
- 现有楼层快照不足以显示普通地堡每层真实居民。首版应显示设施与部门锚点；真实逐层人口需要新增只读 map projection，不能把 0 人口当作真空楼层，也不能随机摆人。

## 2. 当前界面与数据事实

### 2.1 已核实现状

| 现状事实 | 对设计的影响 |
| --- | --- |
| `available_actions` 已由后端下发，包含 `group`、`target_type`、`ap_cost`、`duration_months`、`enabled` 等字段。[PlayerActionMeta](../../internal/model/models.go#L727-L740) | 不必在前端维护第二份行动成本表；五域应是后端元数据的增量扩展。 |
| 当前前端按 `group` 分组后，每组仍以两列、80px 高按钮平铺。[App.tsx](../../frontend/src/App.tsx#L273-L297) [行动渲染](../../frontend/src/App.tsx#L424-L548) | 问题不是完全没分组，而是所有分组同时展开、行动详情与目标选择又继续向下堆叠。 |
| 当前通用行动实际有 Gather Info、Share Info、Build Network、Incite Rebellion、Propaganda，以及满足条件时的 Publicize Faction。[common_actions.js](../../events/mechanics/common_actions.js) | `common` 至少跨情报、关系、影响三个意图，不能直接改名成一个抽屉。 |
| 每个职业有两个专属行动，行动内容由 JS 机制定义。[profession_actions.js](../../events/mechanics/profession_actions.js) | “职务”按解锁来源归类最稳定；职业行动的情报/影响效果用次级标签表示，不重复放进多个域。 |
| 普通开局把特工分配到随机驻地；驻地地图使用 `world.station_silo`，Silo 40 不作为可模拟驻地。[station.go](../../internal/service/station.go#L11-L58) [App.tsx](../../frontend/src/App.tsx#L350-L355) | 剖面主体应写成动态的“当前驻地”，不能硬编码成 40 号。 |
| 当前驻地有情报、策反、欺骗、交易四类 delayed operation；Silo 1 只开放情报与欺骗。Silo 18 仅观察，不可操作。[station_operations.go](../../internal/service/station_operations.go#L18-L63) [silo1_operations.go](../../internal/service/silo1_operations.go#L30-L70) [WorldPanel.tsx](../../frontend/src/components/WorldPanel.tsx#L54-L58) | 两类 operation 共享一个秘密行动域，以“驻地 / Silo 1”chip 切换；Silo 18 不出现在行动目标中。 |
| 地堡有 144 层，楼层含 `Level`、`Function`、`Zone`、`Stability`、`Population`；初始化形成 11 个连续功能区。[initializer.go](../../internal/engine/initializer.go#L136-L168) | 功能区可以完全由数据派生，不需要把 11 个布局组件写死。 |
| 当前 `BunkerMap` 只按 ELITE/COMMONER 两级展示部门卡片，没有使用 `silo.floors`。[BunkerMap.tsx](../../frontend/src/components/BunkerMap.tsx#L23-L122) | 新剖面应替代“阶层卡片墙”，而不是在它下面再叠一张地图。 |
| 部门目前只有 `Zone`，没有权威 `primary_floor`；普通快照裁掉完整 `residents`。[architecture.md](../../references/architecture.md#L48-L49) | 部门可先做 zone 级锚点；要精确落到单层，需要增量位置字段。 |
| 楼层初始化时 `Population: 0`。[initializer.go](../../internal/engine/initializer.go#L154-L164) | 首版不得把该值解释为“本层无人”；应显示“人口未测绘”或干脆不显示逐层人口。 |
| 秘密行动 operation 的 `DurationMonths` 当前为 0，真实 1–2 月生效延迟写在 operation 定义与描述中。[silo1_operations.go](../../internal/service/silo1_operations.go#L62-L72) | 新草稿面板需要独立的 `effect_delay_months`，不能把本地耗时误当秘密行动延迟。 |
| Linux/WebKitGTK 已针对渲染性能修过大面积阴影、泛化 transition 和 Canvas 每帧 blur。[08-linux-frontend-rendering-perf.md](../../docs/research/08-linux-frontend-rendering-perf.md) | 新设计必须保持静态、定向、按需绘制；不应以“地图”为理由重新引入持续动画。 |

### 2.2 设计边界

**【设计提案】**本轮只重构信息架构与交互，不改变行动效果、AP、Elo、匹配、事件时间戳、派系聚类或结局语义；不增加主角走楼，不要求每个 NPC 实时显示，不把普通部门行动偷偷改成楼层目标。现有执行请求仍使用部门、资源或全堡目标，楼层只是选择这些目标的入口。

## 3. 参考案例核查与借用边界

### 3.1 Hearts of Iron IV：借任务域，不借“一个统一行动栏”的想象

**【参考事实】**HOI4 不是一个统一的“政治、外交、情报行动列表”。顶层入口包括 Government、Decisions、Research、Diplomacy、Trade、Construction、Production、Recruit & Deploy、Logistics、Officer Corps；La Résistance 增加 Intelligence Agency。Government 打开政治屏，外交先选国家再显示双边行动，情报机构另有 upgrades、Operations、decryption 等结构。[cite:1][cite:3]

Decisions 集中于 Events and Decisions，每项属于 category；`visible` 不满足时隐藏，`available` 不满足时保留但灰置，成本可以带自定义 tooltip。[cite:2] La Résistance 的 Operations 会展示 Operatives、Network strength、Risk、Days、Additional Cost、Requirements 等字段，准备与执行可能跨多个阶段。[cite:3][cite:4] 分类的官方目的之一是避免大量 Decisions 造成杂乱。[cite:5]

- **解决了什么【设计分析】**：把不同心智任务拆到稳定入口，并区分“当前不可见”和“知道存在但条件不足”。
- **代价【设计分析】**：多屏跳转增加返回成本；灰项长期占位；情报 Operation 的字段不能自然套到所有普通行动。
- **可直接借**：固定任务域、不可用原因、成本与条件并列、目标作为域内筛选。
- **只能借思路**：情报行动的阶段与风险摘要。Silo40 只有已有证据的怀疑、Exposure 和延迟才可显示，不能造一个统一“Risk%”。
- **不适合**：复刻其国家层级、多屏资源管理，或声称它原本就有一根包含所有行动的侧栏。

### 3.2 Beholder：借空间热点，不借走楼

**【参考事实】**Beholder 把一栋公寓切开，房间、公共空间和人物活动同屏；人物会离开房间，形成可潜入时段。[cite:6][cite:8] 玩家点击地面移动 Carl，点门后选择窥视、钥匙或敲门，点物件后搜索；滚轮缩放、拖动平移，人物信息进入 Notes。[cite:7]

- **解决了什么【设计分析】**：空间位置直接解释“谁在哪里、何时能做什么”，动作与门、物件形成稳定关联。
- **代价【设计分析】**：总览和细节依赖缩放平移，视野外信息会消失；小门、小物件形成二次点选负担。
- **可直接借**：空间热点 → 相关行动；人物/部门 Notes → Inspector。
- **只能借思路**：总览到局部的层级，不保留其镜头负担。
- **不适合**：点地移动、潜入时机、主角跑楼。Silo40 本轮没有这些玩法，加入会让界面承诺不存在的机制。

### 3.3 This War of Mine：借设施状态，不借人物优先操作

**【参考事实】**This War of Mine 使用不规则的多层毁屋剖面，幸存者可以清障、搜柜、建造，设施与人物状态同屏。[cite:9][cite:10] 典型操作是先选幸存者，再点衣柜、门、瓦砾或工作台；另有全局分工界面。[cite:10] 实机评测记录过图标与人物重叠、镜头边界问题。[cite:11]

- **解决了什么【设计分析】**：设施、人物和执行状态处于同一空间上下文，适合理解并发活动。
- **代价【参考事实+设计分析】**：小图标可能重叠，镜头取景影响点击；人物优先的选择顺序不适合单特工远程操作。
- **可直接借**：设施热点、静态执行状态、全局记录另置。
- **只能借思路**：选中后聚焦；Silo40 应以部门/设施为主，而非先选幸存者。
- **不适合**：双击跑动、多人派工矩阵、未经聚合的小图标群。

### 3.4 Frostpunk：借单主题 Overlay，不借环形城市或径向菜单

**【参考事实】**Frostpunk 初代城市围绕中央 Generator 组织，建筑与热区具有径向关系。主机版 Command Hub 使用径向入口和分组子菜单；PC 原版建造入口是底部 building tab，不能混称为同一种界面。[cite:12] PC 的温度检查通过独立 heat map，用颜色查看住宅冷暖。[cite:13]

- **解决了什么【设计分析】**：空间形态直接解释中心热源的服务范围；单主题 overlay 能在需要时覆盖同一空间。
- **代价【设计分析】**：布局强绑定单一中心；径向深层菜单不适合频繁比较成本，overlay 切换也要求稳定图例。
- **可直接借**：恐慌、稳定、派系、情报一次只开一个 overlay；始终保留图例和“关闭”。
- **只能借思路**：设施影响范围。若未来某设施确实影响相邻楼层，可显示上下覆盖范围。
- **不适合**：同心圆城市、径向网格、为主机摇杆设计的 radial wheel。Silo40 的核心空间关系是上下层级，不是距中心远近。

### 3.5 RimWorld：借渐进披露，不借全量 HUD 与 Work 矩阵

**【参考事实】**RimWorld 常显资源、colonist bar、alerts；无选择时左下显示指针格信息，选中对象后同位置变成 inspect pane。[cite:14] 底部主 tabs 分开 Architect、Work、Schedule 等，Architect 再分类命令；Work 是 colonist × 工种矩阵。[cite:15][cite:16] 右下还有多个独立显示开关。[cite:14]

- **解决了什么【设计分析】**：用“摘要常显 → hover/指针摘要 → 选中详情 → 全局配置”承载大量实体，主地图不被所有数字淹没。
- **代价【设计分析】**：图标和开关学习成本高，详情依赖选择后才可见；大矩阵占用空间。
- **可直接借**：稳定 Inspector、选中后详情、摘要与精确值分层。
- **只能借思路**：overlay 开关；Silo40 应一次只开一个主题，避免多个开关叠加成噪声。
- **不适合**：把全量资源和所有开关常展开，或在没有多角色×工种需求时复制 Work 矩阵。

## 4. 总体布局：稳定外壳与单一情境 Dock

### 4.1 桌面标准线框（建议基准 1440 × 900）

**【设计提案】**

```text
┌──────────────────────────────────────────────────────────────────────────────────┐
│ Y122 M3 · 当前驻地 │ AP 50 │ 怀疑 [••□□□□□□□□] │ 合法性/叛乱 │ 特工身份 │ 保存 │ 56
├────┬───────────────────────────────────────────────────────┬─────────────────────┤
│地图│ 驻地 / 全堡功能区                         [态势▾] [图例] │ 行动 | 详情         │
│派系│ ┌───────────────────────────────────────────────────┐ │ ┌──────┬──────────┐ │
│世界│ │ 楼梯井 │ 功能区剖面                     │ 状态栏   │ │ │情报 2│ Gather…  │ │
│记录│ │        │ 1–10  Administrative         │  ✓       │ │ │关系 2│ 10AP 即时│ │
│    │ │   ║    │ 11–20 Public Facilities       │  !       │ │ │影响 2├──────────┤ │
│    │ │   ║    │ 21–30 Residential A           │  ···     │ │ │职务 2│ Share…   │ │
│    │ │   ║    │ 31–60 Residential B           │  ◐       │ │ │秘密行动 6│ 20AP 即时│ │
│    │ │   ║    │ 61–75 Hydroponics             │  ≋ !     │ │ │      ├──────────┤ │
│    │ │   ║    │ …                             │          │ │ │当前目│ 目标筛选 │ │
│    │ │   ║    │ 141–144 Mines                 │  !       │ │ │标:全部│ [全堡▾] │ │
│    │ └───────────────────────────────────────────────────┘ │ └──────┴──────────┤ │
│    │ 选中摘要 / 最近事件（一行，不挤压地图）                │ 草稿或行动说明      │
│    │                                                       │ AP/耗时/痕迹/执行   │
└────┴───────────────────────────────────────────────────────┴─────────────────────┘
  56px                自适应，最小 480px                         384–400px
```

关键约束：

- 顶栏始终保留 **AP 与怀疑**；其他全局指标可折叠，不能因响应式把执行成本相关状态藏掉。
- 左侧 56px 只切换地图、派系、世界、记录，不再承担五个行动域。
- 右侧 Dock 的“行动/详情”共用同一位置。点地图对象进入详情，点“在此行动”回到行动；两种状态都保留对象 breadcrumb，避免打开两个互相遮挡的面板。
- 行动列表、详情和草稿不会同时向下无限增长。选中行动后，列表区域切换成草稿，返回到原域时保留滚动位置。
- 地图始终留在屏幕上，目标选择不弹 modal；这样行动草稿和地图反馈可以同时看见。

### 4.2 行动 Dock 线框

```text
┌──────────────────────────────────────────┐
│ 行动 | 详情                       [全堡▾] │
├─────────┬────────────────────────────────┤
│ 情报  2 │ [搜索行动…]                    │
│ 关系  2 │ ┌────────────────────────────┐ │
│ 影响  2 │ │ Gather Info          10 AP │ │
│ 职务  2 │ │ 即时 · 怀疑 +0.005      › │ │
│ 秘密行动  6 │ ├────────────────────────────┤ │
│         │ │ Share Info           20 AP │ │
│ 当前域  │ │ 即时 · 选择后显示碎片代价 › │ │
│ 可执行数│ └────────────────────────────┘ │
│         │ ▾ 暂不可用（原因）             │
├─────────┴────────────────────────────────┤
│ 选中后此处变为固定草稿，不再把页面撑长    │
└──────────────────────────────────────────┘
```

每个行动行固定回答四件事：**名称、AP、时间、主要痕迹**。完整效果、条件和其他代价进入草稿，不要求玩家逐个 hover 才能比较。

## 5. 五个行动域的精确定义

### 5.1 分类表

| 域 | 当前装什么 | 边界 | 增长时怎么收 |
| --- | --- | --- | --- |
| **情报** | Gather Info、Share Info；未来以获取、验证、分享、删除信息为主要用途的通用行动 | 行动的主要交换物是 fragment、报告或信息可信度；不因“造成一点恐慌”就改归影响 | 内部用“收集 / 分享 / 处理”次级标签；搜索与当前目标筛选 |
| **关系** | Build Network、Publicize Faction；未来 `faction_member` / `faction_leader` 的人脉、庇护、组织行动 | 主要改变 agent–department、agent–faction 或组织公开状态；不直接以传播思潮为主要效果 | 内部用“人脉 / 庇护 / 组织”标签；派系身份是解锁条件，不另开顶级域 |
| **影响** | Propaganda、Incite Rebellion；未来通用的宣传、思潮、恐慌、秩序和叛乱行动 | 主要改变人群态度或社会计量；单纯收集信息、单纯生产资源不放入 | 内部用“宣传 / 鼓动 / 秩序”标签；高痕迹不等于秘密行动 |
| **职务** | 当前职业的两个行动、`profession_group` 行动 | **按执行权限归类**：必须依赖当前职业/职业阶层的行动都在这里，即使效果也涉及情报或影响；行动行显示“情报”“关系”等效果标签 | 职业只展示当前可用项，不列其他职业；未来职业树在域内按“岗位 / 专业 / 阶层”分组，不增加顶级域 |
| **秘密行动** | 当前驻地的 intel / infiltrate / deceive / trade，以及 Silo 1 的 intel / deceive | **按执行通道归类**：具有独立 operation track、在途生效、Exposure 与反间谍的行动都在这里；普通即时部门行动即使增加怀疑也不进入 | 域内第一行固定目标 chip：`当前驻地 / Silo 1`；每个目标只显示后端实际开放的行动，不为 Silo 18 或未来目标预造入口 |

### 5.2 为什么是五个，而不是四个或六个

**四个不够**：若把“情报、关系、影响”合并成“驻地”，当前最常见的三种决策仍会在同一个长列表里混杂；若把“职务”并入“影响”，玩家无法按职业身份找到专属能力；若把“秘密行动”并入“情报/影响”，普通即时行动与 delayed operation 的 Exposure、反间谍和在途反馈会混成一团。

**六个太多**：当前驻地与 Silo 1 是同一类 operation 的不同目标，不应各占一个顶级域；Silo 18 不可操作，更不应只为观察设行动域。派系领袖、职业阶层是解锁条件，不应各占一个顶级域；“资源”目前主要是职务行动的目标类型，也不足以成为与五者并列的日常域。

因此五域对应五个稳定问题：

1. 情报——**能知道或告诉别人什么？**
2. 关系——**能依靠谁、组织谁？**
3. 影响——**能让人群相信或做什么？**
4. 职务——**当前身份能动用哪些专业权力？**
5. 秘密行动——**能动用哪条带延迟与追查的行动通道？**

### 5.3 `group` 能否直接复用

**结论：继续复用，但不能单独承担新分类。**

| 现有 `group` | 新域映射 | 是否还需其他字段 |
| --- | --- | --- |
| `common` | 按 `action_type` 拆为情报/关系/影响 | 需要 `ui_category`；不能只改标题 |
| `profession`、`profession_group` | 职务 | `group` 可继续表达解锁来源 |
| `faction_member`、`faction_leader` | 关系 | 仍需 `ui_category`；领袖只是可见性条件 |
| `station`、`silo1` | 秘密行动 | 需要结构化 `target_silo_id`，不能靠组名显示目标 |

**【设计提案】**新增行动必须声明 `ui_category`；缺失时在内容校验/测试中显式失败，不静默塞入一个“其他”域。为兼容旧快照，前端临时 adapter 可以只对已知的 `group + action_type` 做映射；遇到未知旧行动时显示一个明确的“未分类行动”错误条目并允许查看，但不能悄悄伪装成分类成功。

### 5.4 行动数量增长规则

- 五个顶级域永久固定；新增目标、职业、派系身份只增加域内筛选或行。
- 每域默认单行列表，不再两列按钮；排序沿用“可执行优先 → AP → 名称”，并提供名称/效果标签搜索。
- 默认显示当前可执行项；不可用项保留在底部折叠组，展开后必须显示具体原因。`UnavailableBehavior=hide` 的行动仍按后端规则隐藏。
- 域 badge 显示“当前目标下可执行数 / 当前域总数”，例如 `3 / 7`，避免玩家误以为其他行动被删除。
- 每域超过约 8 项时只增加列表滚动，不增加 drawer 层级；不使用第三级嵌套菜单。
- “最近使用”不是新域；如后续确有需要，只在搜索框下增加一行最近行动 chip。

## 6. 三级剖面的线框与导航

### 6.1 第一级：全堡功能区

**目的**：一眼看懂 Upper / Mid / Lower 与职能分布，找到有事件或行动目标的楼层段，而不是阅读 144 行数据。

```text
驻地 / 全堡功能区                                  [态势:关闭▾]
┌───────────────────────────────────────────────────────────────┐
│ UPPER │ 1–10    Administrative               [IT][司法]   ✓   │
│       │ 11–20   Public Facilities                       !   │
│       │ 21–30   Residential A                           ··· │
│ MID   │ 31–60   Residential B                           ◐   │
│       │ 61–75   Hydroponics              [农业]        ≋ !  │
│       │ 76–80   Medical                  [医疗]           ✓   │
│       │ 81–90   Cafeteria & Supply       [补给]           !   │
│ LOWER │ 91–120  Residential C                           ◐   │
│       │ 121–135 Mechanical               [机械]           ≋   │
│       │ 136–140 Maintenance                               !   │
│       │ 141–144 Mines                  [矿业]             !   │
└───────────────────────────────────────────────────────────────┘
```

- 功能区由**连续且相同 `Zone + Function` 的 Floor**自动归并；同名但不连续的区段不能合并。
- 每行常显：楼层范围、职能、最多两个已知部门锚点、最多三个状态符号。人口、精确恐慌、派系比例不常显。
- 行高按楼层数做有限比例映射，并设最小/最大高度；小窗口改为统一行高，但仍保持楼层顺序。
- 点击功能区：展开其下的楼层 strip，并选中该区段；再次点击收起。展开不是第四级，只是第一级的就地展开。
- 点击具体楼层：进入第二级。功能区和楼层 strip 都保留在同一滚动上下文，返回时不跳回顶部。

### 6.2 第二级：单层

**目的**：把一层变成行动锚点，同时明确“这里有什么已知对象”；不假装是精确建筑平面图。

```text
驻地 / 中区 / 61–75 水培 / 68层          [返回功能区] [详情]
┌───────────────────────────────────────────────────────────────┐
│ 楼梯井 │ 主功能空间                    │ 已知对象 / 状态        │
│  ↑ 67  │ ┌──────────────────────────┐ │ 农业部门            › │
│  F68   │ │ 水培设施                 │ │ 已知人群摘要        › │
│  ↓ 69  │ │ 稳定 [■■■■■■■□□□]       │ │ 谣言/事件（若有）   › │
│        │ └──────────────────────────┘ │                       │
│        │ ┌────────────┬─────────────┐ │                       │
│        │ │ 部门锚点   │ 人群/派系   │ │                       │
│        │ │ 农业       │ 仅显示已知  │ │                       │
│        │ └────────────┴─────────────┘ │                       │
└───────────────────────────────────────────────────────────────┘
```

- 左侧上下楼层按钮等价于上下导航，不表示主角移动。
- 中央只画 1 个主设施与最多 2 个辅助 slot；更多对象进入右侧“已知对象”列表，不堆小图标。
- 常显：楼层号、Function、稳定度的分段条、已确认部门/设施、最多一条最高优先级事件。
- 折叠：精确人口、cohort 清单、派系细节、fragment 清单、历史记录，全部进 Inspector。
- 没有权威对象时显示“暂无已确认情报”，不是空白房间，也不生成装饰 NPC。
- 点击部门、设施、人群摘要或事件：进入第三级 Inspector；点击楼层空白处只选楼层，不强行选部门。

### 6.3 第三级：Inspector

Inspector 复用右侧 Dock 的“详情”页，不在地图上再叠浮窗。

```text
┌──────────────────────────────────────────┐
│ 行动 | 详情                              │
├──────────────────────────────────────────┤
│ 农业部门                        F61–75   │
│ 中区 · 平民 · 已确认                     │
│ [在此行动]                               │
│                                          │
│ 概况                                     │
│ 恐慌 [■■■□□□□□]  人脉 [■■□□□]           │
│ 倾向：亲外 / 中立忠诚                    │
│                                          │
│ 人群与派系（最多2项，余下展开）          │
│ 情报碎片（已知 1 / 未确认若干）          │
│ 最近痕迹与事件                           │
│                                          │
│ 精确数值默认折叠在“查看计量”下           │
└──────────────────────────────────────────┘
```

- 顶部始终显示对象 breadcrumb 与“在此行动”。
- “未知”必须写成未知，不用 0、空字符串或灰色数字冒充。
- 派系倾向显示标签与相对存在，不默认展示全部小数。
- “查看计量”保留当前调试/精确定量需求，但不占据第一屏。

### 6.4 下钻、返回与不迷路规则

| 操作 | 结果 | 返回方式 |
| --- | --- | --- |
| 点功能区 | 就地展开楼层 strip，选中区段 | 再点区段或点 breadcrumb“全堡” |
| 点楼层 | 中央切换为单层，右侧打开该层详情摘要 | breadcrumb“61–75 水培”、返回按钮 |
| 点部门/人群/事件 | 右侧 Dock 切到详情，中央保持单层 | Dock 关闭详情或点楼层空白 |
| 点“在此行动” | Dock 切到行动，启用该对象上下文筛选 | 点“详情”返回同一对象 |
| 选中行动 | Dock 切到草稿，地图显示合法目标 | “返回行动列表”不清空空间选择 |
| Esc | 优先取消目标挑选/草稿，其次关闭 Inspector，最后返回上一级 | 与可见返回按钮完全一致 |

必须保存：`selectedSection`、`selectedFloor`、`selectedAnchor`、每域滚动位置和最后打开的行动域。玩家在剖面与行动之间往返时，不得重置到全堡顶部或默认第一域。

## 7. 剖面与行动的联动规则

### 7.1 点楼层是否过滤行动

**【设计提案】采用“情境筛选”，不是静默硬过滤。**

- 选中楼层/部门后，行动 Dock 顶部出现可移除 chip：`目标：F68 · 农业 ×`。
- 对 `DEPT` 行动，匹配该楼层权威部门锚点的行动排在“当前目标”区；其他部门行动进入“其他目标”，默认折叠但仍可搜索。
- `NONE` 的全堡行动不因选楼层消失，单独显示“全堡”标签。
- `RESOURCE` 行动仍要求资源目标；只有该设施有权威资源锚点时才预填，否则在草稿内选择。
- 秘密行动不受驻地楼层硬过滤，保留在秘密行动域；驻地 operation 作用于全堡，Silo 1 operation 作用于外部目标，都不应被某个楼层上下文错误筛掉。
- 当前目标无可执行行动时显示空态：“此目标暂无可执行行动 · 查看全部”，不能只留下一个空抽屉。

楼层本身不是伪造的新执行参数。若选中 F68 映射到农业部门，最终请求仍提交 `target_dept=Agricultural`；草稿必须把这个转换明确写出来。

### 7.2 路径 A：先选空间，再选行动

1. 在全堡功能区点“61–75 Hydroponics”。
2. 在展开的楼层 strip 点 F68；中央进入单层。
3. 点“农业部门”；右侧出现 Inspector。
4. 点“在此行动”；Dock 显示 `目标：F68 · 农业`，相关部门行动排在前面。
5. 选 Gather Info；草稿显示 `Gather Info → Agricultural`、10 AP、即时、怀疑增量。
6. 确认执行；地图目标保留，结果显示在草稿位置，最近事件在楼层和记录中出现。
7. 返回行动列表时仍是 F68 与农业，不清空上下文。

### 7.3 路径 B：先选行动，再选地图目标

1. 玩家先打开情报域并选择 Gather Info。
2. Dock 进入草稿，顶部显示“选择部门目标”；地图进入 target picking。
3. 合法部门所在功能区/楼层使用实线高亮；当前已选目标用双层边框。无合法锚点的楼层变暗但仍可查看，不清空原草稿。
4. 点合法楼层后，若一层只有一个部门锚点则直接填入；有多个锚点时在 Inspector 要求再选部门，不能猜。
5. 草稿显示目标与全部代价后，执行按钮才可用。
6. 取消 target picking 只退出本次草稿，不改变此前空间选择。

### 7.4 不同目标类型的地图反馈

| 行动目标 | 地图反馈 | 草稿要求 |
| --- | --- | --- |
| `DEPT` | 高亮含合法部门锚点的功能区与楼层；选中后标出具体对象 | 明确显示最终 `target_dept` |
| 驻地 `NONE` | 不高亮某个随机楼层；用全堡外框或实际受影响的 class/zone 范围 | 显示“全堡”或已知影响范围 |
| `RESOURCE` | 若有权威设施锚点则高亮，否则不做空间高亮 | 资源选择器保留在草稿 |
| 秘密行动 | 目标为当前驻地时只高亮全堡外框；目标为 Silo 1 时高亮世界目标卡，不假装出现其内部楼层 | 显示目标地堡、生效月、对应 Exposure、即时/结算怀疑 |
| 多部门/阶层效果 | 高亮已知受影响部门集合；不能只因 `NONE` 就点亮全 144 层 | 显示数量与范围摘要 |

### 7.5 草稿、痕迹与执行反馈

秘密行动草稿示例使用当前真实字段语义：

```text
投放伪造档案 → Silo 1
AP 50 → 30                生效：2 个月后
即时怀疑 +0.006           Silo 1 Exposure +10%
结算时怀疑 +0.04          本地耗时：0 月
[返回]                                  [确认下达]
```

- 即时怀疑、结算怀疑、目标 Exposure 分开列，不能合并成一个“风险”。
- 对 variable cost 显示范围或公式说明；未知就写未知，不能生成虚假精确值。
- 执行后不做持续脉冲动画。用一次性结果卡、楼层静态事件标记和记录条目形成反馈。
- pending operation 显示在世界目标条和记录中；本地楼层只显示确有空间来源的事件，不为没有 floor anchor 的 operation 捏造楼层位置。

## 8. 状态可视化：空间符号优先，精确数字折叠

### 8.1 统一语义

| 状态 | 正确作用域 | 空间表达【设计提案】 | 不采用 |
| --- | --- | --- | --- |
| **谣言/未确认情报** | 有明确 zone/floor 来源时才落在空间；否则属于世界或全堡记录 | 琥珀色虚线框 + 对话符号；选中后显示来源、最后确认时间、相关 fragment。旧情报降低饱和度但保留文字日期 | 不给无位置谣言随机安排楼层；不显示伪造百分比 |
| **怀疑 Suspicion** | 特工本人，全局唯一 | 顶栏眼睛图标 + 10 格 meter，标出 0.5 与 1.0 语义点；行动草稿用斜纹增量格预览 | 不把每一层染红；不在各部门复制同一个怀疑值 |
| **曝光 Exposure** | 当前驻地与 Silo 1 各自独立的 operation track | 对应目标条用 4 段 meter，对应 25% 阈值；旁边显示反间谍阶段符号，精确值进详情 | 不拆成每层属性；不与 Suspicion 合并 |
| **派系倾向** | faction/cohort；只有权威空间关联时才聚合到 zone/floor | Inspector 显示“亲外/中立/排外”等标签；空间上最多两个派系短带 + `+N`，未知派系用灰色网纹 | 不用多个饼图、雷达图和小数覆盖楼层 |
| **人脉** | agent ↔ department | 部门锚点旁使用链节符号与 5 格短条；详情显示精确值 | 不用每张卡重复 `%` 大数字 |
| **恐慌** | department/cohort，按已知空间聚合 | 功能区右侧 5 格橙色条；超过现有高风险语义时加 `!`，详情给精确值 | 不用全屏红色滤镜或常驻闪烁 |
| **结构稳定** | Floor/Zone | 单层主设施旁的 8 格条；功能区只显示最低/平均摘要符号 | 不绘制持续裂纹粒子或动画 |
| **痕迹** | 行动与目标之间的关系 | 草稿中显示预览；执行后在相关对象 Inspector 的“最近痕迹与事件”显示静态条目 | 不把全局怀疑伪装成楼层永久污染值 |

### 8.2 Overlay 与常显界线

- **常显**：当前选择、合法目标、高优先级事件、已知部门锚点、AP/怀疑、当前秘密行动目标的对应 Exposure。
- **按需 overlay**：`恐慌`、`结构`、`派系`、`情报`，一次只能启用一个；图例固定在地图右上，关闭后回到基础态。
- **只在 Inspector**：完整 ideology 数值、全部 faction/cohort、fragment 清单、历史事件、精确人脉与人口。
- **只在草稿**：本次行动的 AP、时间、怀疑、Exposure、资源代价与目标范围。
- 所有颜色同时配图形/文字；灰网纹表示未知，斜纹增量表示预览，虚线表示未确认，实线表示已确认。

### 8.3 数据不足时的诚实降级

**【数据缺口】**当前 `RumorView` 主要是外部地堡级迷雾，事件历史也没有通用 floor anchor；因此谣言默认只能出现在世界目标或全堡记录。只有后续事件 DTO 明确提供 `zone` / `floor` / `department` 时，才允许落到剖面。

**【数据缺口】**普通居民完整列表未下发，cohort 只有 `HomeZone`，不能据此伪造精确楼层派系分布。第一级可做 zone 级派系摘要；单层派系带必须等 map projection 提供真实聚合结果。

## 9. React 落地设计

### 9.1 渲染技术选择

| 技术 | 用途 | 结论 |
| --- | --- | --- |
| **DOM + CSS Grid** | 功能区、楼层 strip、行动列表、Inspector、草稿、响应式 | 主方案；原生点击/焦点/文本/滚动，最容易保证可访问性 |
| **静态 SVG** | 楼梯井、上下连接、范围括线、少量状态 glyph | 辅助；使用固定 `viewBox`，只在选择/数据变化时更新 |
| **Canvas** | 大量逐帧实体或复杂粒子 | 本方案不使用；当前没有该需求，且会增加命中测试、可访问性和重绘成本 |

不引入 D3、Pixi、地图引擎或新状态库。第一级约 11 个功能区、第二级只显示一层，本身就控制了 DOM 数量，不需要为了 144 层预先上 Canvas 或虚拟化。

### 9.2 组件拆分

```text
GameShell
├─ HeaderStats
├─ PrimaryNav
├─ SiloWorkspace
│  ├─ MapBreadcrumb
│  ├─ SiloSectionView
│  │  ├─ ZoneGroup
│  │  └─ FunctionBand
│  │     └─ FloorStrip
│  ├─ FloorView
│  │  ├─ StairwellNav
│  │  ├─ FacilitySlot
│  │  └─ KnownEntityList
│  └─ MapOverlayLegend
└─ ContextDock
   ├─ DockTabs
   ├─ OperationsDock
   │  ├─ DomainRail
   │  ├─ ActionContextBar
   │  ├─ ActionList
   │  │  └─ ActionRow
   │  └─ ActionDraft
   │     └─ CostTraceSummary
   └─ SelectionInspector
```

纯函数与交互状态分开：

```text
selectActionDomain(action)       -> domain
buildActionRows(actions, ctx)    -> sorted/filtered rows
buildSiloSections(floors)        -> contiguous zone/function sections
buildFloorView(floor, anchors)   -> floor slots and known entities
getValidTargets(action, mapVM)   -> target anchors
getMapFeedback(draft, mapVM)     -> highlight scope
```

`GameShell` 或一个局部 `useReducer` 持有以下 UI 状态即可，不必引入全局 store：

```ts
type WorkspaceState = {
  panel: 'actions' | 'details'
  sectionKey?: string
  floorId?: number
  anchorId?: string
  actionDomain: 'intel' | 'network' | 'influence' | 'profession' | 'covert'
  actionKey?: string
  target?: { kind: string; id: string }
  contextFilter: 'relevant' | 'all'
  overlay: 'none' | 'panic' | 'stability' | 'factions' | 'intel'
}
```

游戏状态仍由 Go 快照权威提供；reducer 只保存选择与草稿，不复制引擎计量。

### 9.3 行动元数据：增量而不是解析描述

建议后续契约新增以下只读字段；现有字段保留：

| 字段 | 用途 |
| --- | --- |
| `ui_category` | 五域唯一归属 |
| `intent_tags` | 域内搜索与效果标签，如 `gather`、`share`、`network` |
| `target_silo_id` | 秘密行动的“当前驻地 / Silo 1”chip 与地图/世界反馈 |
| `effect_delay_months` | 秘密行动或在途效果的生效延迟；不覆盖现有本地 `duration_months` |
| `trace_preview` | 当前特工上下文中的即时怀疑、结算怀疑、Exposure；支持范围或未知说明 |
| `disabled_reason` | 灰置原因，不从 description 猜 |
| `sort_order` | 同域稳定排序，未提供时沿用现有排序 |

`trace_preview` 必须由后端按当前职业、特质、碎片选择等权威上下文计算。前端不能从中文/英文 description 中提取 `+0.05`，也不能复制公式。

### 9.4 空间数据驱动

第一级不需要新布局配置：

1. 按 `Level` 升序排列 floors。
2. 仅在 `Zone + Function` 相同且楼层连续时归并为一个 section。
3. `sectionKey` 使用 `siloId:startLevel:endLevel:zone:function`，不能只用 Function 防止重名。
4. 新增楼层会扩大 section 或生成新 section；组件代码不变。

对象落图采用显式 anchor：

```ts
type MapAnchor = {
  id: string
  kind: 'department' | 'facility' | 'cohort' | 'faction' | 'event' | 'resource'
  refId: string
  floorId?: number
  zone?: string
  label: string
  visibility: 'confirmed' | 'rumor' | 'unknown'
}
```

- 部门若有权威 `primary_floor`，落到单层；只有 `Zone` 时停留在功能区，不随机选一层。
- 新部门/新事件只增加 anchor，不修改 `FunctionBand` 或 `FloorView`。
- 对普通地堡的真实逐层人口，建议后端提供裁剪后的 `SiloMapView`：只含每层的已知数量、可见具名人物、派系聚合和事件锚点，不回传完整 residents。
- `Floor.Population` 在语义未补清前不作为真实展示来源；map projection 应有 `population_known`，以区分“0 人”和“未知”。

### 9.5 性能约束

- 首屏只渲染功能区；只展开一个楼层 strip；单层只渲染当前层。
- `buildSiloSections`、行动筛选等纯计算用 `useMemo` 按快照引用与选择条件重算；静态行组件可 `memo`，但不要先建复杂缓存体系。
- 不做 NPC 位置插值、自动轮播、持续 pulse、视差、blur、粒子或每帧 React state 更新。
- 不使用 `transition: all`；如确有反馈，仅对 border/color 做短定向 transition，低性能模式全部关闭。
- SVG 不每帧重绘；图标复用现有 `lucide-react`。
- 延续现有性能审计口径，在真实 WebKitGTK 路径做滚动与目标选择 A/B，而不是只看 Chromium FPS。

## 10. 小窗口与低分辨率降级

### 10.1 响应式规则

| 可用尺寸 | 布局 | 保留 | 降级 |
| --- | --- | --- | --- |
| **≥1280px 宽且高 ≥720px** | 左 nav + 中央剖面 + 384–400px 右 Dock | 完整三级剖面、五域纵栏、地图与草稿同屏 | 次要顶栏指标可折叠 |
| **1024–1279px 宽** | 右 Dock 缩至 340–360px；功能区减少装饰文字 | 地图与行动同屏、AP/怀疑常显 | 行动描述只在草稿；状态符号减为两个 |
| **宽 <1024px 或高 <700px** | 中央上 55% 地图、下 45% bottom sheet；行动/详情仍两页签 | 目标与地图可同屏；五域变横向五个等宽按钮 | 不显示右域说明栏；功能区行高统一 |
| **宽 <768px** | 单视图：地图 / 行动 / 记录底部切换 | 选择 chip、草稿、返回路径、AP/怀疑 | 选行动后全宽草稿，通过“选择地图目标”往返 |
| **高度 ≤600px** | 顶栏压至 44px；底部 sheet 可全高 | 执行按钮和成本固定可见 | 隐藏装饰性楼梯井，楼层与 slot 改垂直列表 |

### 10.2 800 × 600 线框

```text
┌────────────────────────────────────────────────────┐
│ Y122 M3 │ AP 50 │ 怀疑 [••□□□□□□□□] │ 状态▾       │ 44
├────────────────────────────────────────────────────┤
│ 驻地 / 中区 / F68                    [态势▾]       │
│ ┌────────────────────────────────────────────────┐ │
│ │ F66  Hydroponics                          ✓   │ │
│ │ F67  Hydroponics                          !   │ │
│ │ F68  Hydroponics · Agricultural         ›   │ │
│ │ F69  Hydroponics                          ·   │ │
│ └────────────────────────────────────────────────┘ │
├────────────────────────────────────────────────────┤
│ [情报] [关系] [影响] [职务] [秘密行动]     行动 | 详情 │
│ 目标：F68 · 农业 ×                                 │
│ Gather Info → Agricultural             10 AP       │
│ 即时 · 怀疑 +0.005                                 │
│ [返回]                              [确认执行]     │
└────────────────────────────────────────────────────┘
```

- 小窗口不依赖 hover；所有点击目标至少 40px 高，主要按钮建议 44px。
- 名称过长时列表允许两行，不能截掉 AP、时间和执行按钮。
- 第一级在空间比例布局无法读时改为等高楼层段列表，但 Upper/Mid/Lower 顺序和范围保留。
- 第二级房间 slot 改垂直卡片列表，不通过横向缩放把文字缩到不可读。
- 任何尺寸下都能通过“查看全部”撤销情境筛选，不能让某类行动永久不可达。

## 11. 推荐施工顺序与验收标准

### 11.1 原子阶段

1. **五域 Dock**：先以现有 `group + action_type` adapter 分类，替换平铺按钮；保持执行请求与行动语义不变。
2. **只读三级剖面**：由 floors 生成全堡功能区、单层与 Inspector；没有权威位置的数据只到 zone，不伪造居民。
3. **选择联动**：实现空间优先、行动优先两条路径，统一 reducer、草稿、返回和筛选。
4. **结构化代价与状态 overlay**：契约增量登记并升版本后，再接 `ui_category`、`trace_preview`、`effect_delay_months` 与 map projection。
5. **响应式与性能复核**：覆盖小窗口、键盘、空数据、未知情报、长期列表，按现有 WebKitGTK 口径回归。

### 11.2 可测试验收

- 当前每个可用行动恰好出现在一个域；当前驻地与 Silo 1 共用秘密行动域，目标 chip 可切换，Silo 18 不出现在行动目标中。
- 新增一个带 `ui_category` 的行动只增加行动行，不修改任何布局组件；缺少分类时契约校验显式失败。
- 新增连续楼层自动扩大功能区；新增 Function 自动生成新 band；非连续同名 Function 不错误合并。
- 选中 F68 → 农业 → 在此行动 → Gather Info 后，草稿明确提交 Agricultural；返回后仍保持原空间上下文。
- 先选 DEPT 行动时只有合法部门锚点被标为可执行目标； NONE、RESOURCE、秘密行动不错误点亮驻地楼层。
- 选择楼层后，其他行动仍可通过“查看全部”到达；不可用原因可见。
- Suspicion 只出现在特工全局状态与草稿；Exposure 只出现在对应的驻地或 Silo 1 operation track；二者不混画。
- 无位置谣言只进世界/全堡记录；普通居民数据缺失时不显示虚构人口或派系楼层分布。
- 800 × 600 下无需横向滚动即可完成“选目标 → 看代价 → 执行 → 返回”；AP、怀疑、目标和执行按钮始终可达。
- Linux/WebKitGTK 同口径交互无持续高 CPU；无大阴影、`transition: all`、常驻 Canvas 循环或逐帧阴影。

## 12. 明确不做

- 不做 HOI4 全界面复刻，不设置国家/科研/生产等无关入口。
- 不做 Frostpunk 环形城市、主机 radial wheel。
- 不做 Beholder/TWOM 式主角走楼、潜入、多人派工。
- 不在没有权威数据时生成“每层住着谁”的装饰性假信息。
- 不用统一 Risk% 取代 Suspicion、Exposure、延迟和具体禁用原因。
- 不为新增楼层、部门、目标地堡或行动继续增加硬编码布局分支。
- 不改变现有游戏结算与行动语义；本轮只规定信息如何组织、选择和反馈。

## Sources

- [cite:1] [HOI4 Official Wiki — User interface](https://hoi4.paradoxwikis.com/UI)
- [cite:2] [HOI4 Official Wiki — Decision modding](https://hoi4.paradoxwikis.com/Decision_modding)
- [cite:3] [HOI4 Official Wiki — Intelligence agency](https://hoi4.paradoxwikis.com/Intelligence_agency)
- [cite:4] [HOI4 Official Wiki — Operations](https://hoi4.paradoxwikis.com/Operations)
- [cite:5] [Paradox Dev Diary — Christmas & Modding](https://forum.paradoxplaza.com/forum/developer-diary/hoi4-dev-diary-christmas-modding.1061526/)
- [cite:6] [Beholder — Official Steam page and gameplay screenshots](https://store.steampowered.com/app/475550/Beholder/)
- [cite:7] [Deutschland-Spielt — Beholder 操作指南](https://www.deutschland-spielt.de/spiele/abenteuer/beholder/loesungshilfe/)
- [cite:8] [PlayStation Country — Beholder Complete Edition review](https://www.playstationcountry.com/beholder-complete-edition-ps4-review/)
- [cite:9] [This War of Mine — Official Steam page and gameplay screenshots](https://store.steampowered.com/app/282070/This_War_of_Mine/)
- [cite:10] [Geekscape — This War of Mine hands-on](https://www.geekscape.net/pax-prime-2014-my-hands-on-with-this-war-of-mine-the-most-important-game-of-our-generation)
- [cite:11] [Cheat Code Central — This War of Mine PC review](https://www.cheatcc.com/articles/this-war-of-mine-review-for-pc-pc/)
- [cite:12] [Xbox Wire — Frostpunk: Radial-Driven Design of Gameplay and Controls](https://news.xbox.com/en-us/2019/09/20/frostpunk-console-edition-radial-driven-design-of-gameplay-and-controls/)
- [cite:13] [PwrDown — Frostpunk heat map](https://www.pwrdown.com/gaming/frostpunk-how-to-heat-homes/)
- [cite:14] [RimWorld Wiki — User interface](https://rimworldwiki.com/wiki/User_interface)
- [cite:15] [RimWorld Wiki — Architect](https://rimworldwiki.com/wiki/Architect)
- [cite:16] [RimWorld Wiki — Work](https://rimworldwiki.com/wiki/Work)

# Silo40 界面信息架构重构：五个行动域与三级地堡剖面 · React 落地设计

> 拆分自 [Silo40 界面信息架构重构：五个行动域与三级地堡剖面](../09-ui-information-architecture.md)；目录、章节映射与出处见该索引。

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


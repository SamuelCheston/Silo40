# Silo40 机制研究索引

本目录只保留抽象机制、工程经验与项目实现资料。直接剧情、人物、具体设定出处和事件母题研究已移出公开仓库，不在此处建立替身链接。

## 内容实现与工程

| 文件 | 内容 | 使用场景 |
| --- | --- | --- |
| [07-content-authoring-guide.md](./07-content-authoring-guide.md) | 内容作者手册：目录分组、加载校验、触发器/效果/变异封闭集、冷却与延迟、思潮真源、验证流程与历史缺陷 | 新增或修改内容事件、机制脚本前查阅 |
| [08-linux-frontend-rendering-perf.md](./08-linux-frontend-rendering-perf.md) | Linux/WebKitGTK 渲染性能审计、修复记录与复核口径 | 排查桌面渲染与动画性能问题 |

## 界面与信息架构

| 文件 | 内容 | 使用场景 |
| --- | --- | --- |
| [09-ui-information-architecture.md](./09-ui-information-architecture.md) | 五个行动域、三级地堡剖面、情境 Dock、空间与行动联动、响应式和性能约束 | 界面信息架构与交互设计 |
| [11-information-fragments-design.md](./11-information-fragments-design.md) | Fact/Version/Observation 三层信息模型、共享视角模板、证据组合、有限 LLM 表层变体 | 信息碎片、证据呈现与文本生成边界设计 |

## 权力与组织机制

| 文件 | 内容 | 使用场景 |
| --- | --- | --- |
| [10-it-control-credibility.md](./10-it-control-credibility.md) | 幕后威势、二元关键联盟、处置规程、三方附着、分层确信、检测分类与可打穿平衡目标 | IT/Judicial/Mayor、信息、隐蔽维护与介入机制设计 |

## 同类机制对照

| 文件 | 内容 | 使用场景 |
| --- | --- | --- |
| [game-mechanics-breakdown.md](./game-mechanics-breakdown.md) | 同类策略与模拟游戏的机制拆解，区分参考事实和项目推断 | 比较核心循环、状态、时间、随机性、后果与反馈 |
| [mechanics-comparison.md](./mechanics-comparison.md) | 横向机制对照、项目现状与可借鉴范围 | 新机制立项前的成本与适配性评估 |

## 使用约定

1. 当前行为以代码、测试和 `references/` 项目源参考为准；本目录研究文档不自动代表现状。
2. 新内容只描述抽象机制与原创实现，不复制外部作品的人物、具体剧情、对白或可识别事件链。
3. 历史基线必须注明适用代码版本；旧轨迹和旧设计结论不得直接套到当前版本。

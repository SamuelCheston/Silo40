# Silo40 机制研究索引

本目录只保留抽象机制、工程经验与项目实现资料。涉及外部作品的文档采用“抽象化而非删除”的口径：保留设定结构、数值关系与机制价值，不写具体人物名、组织名、集数、纪年与可指认剧情。

## 世界观与历史机制

| 文件 | 内容 | 使用场景 |
| --- | --- | --- |
| [01-worldbuilding.md](./01-worldbuilding.md) | 地堡结构、楼层分区、部门分工、资源循环、外部与清洁机制、遗物与记忆管制、地堡网络与终止机制、Seed（已拆分为同名子目录，本文件为索引） | 世界观设定、事件触发条件与禁忌规则 |
| [02-timeline.md](./02-timeline.md) | 小说纪年总览、关键跨度、改编侧阶段结构、两套纪年的使用边界 | history 事件、大革命/重置/陷落节点、500 年倒计时 |
| [03-characters-and-factions.md](./03-characters-and-factions.md) | 部门权力图谱、小说人物功能、改编侧人物功能位、暗线组织与派系动力学 | NPC 设计、faction、部门行动与政权更迭 |

## 改编与内容计划

| 文件 | 内容 | 使用场景 |
| --- | --- | --- |
| [04-books-vs-show.md](./04-books-vs-show.md) | 原著与改编的结构差异、人物功能改写、同类节点处理差异、机制层差异、可借鉴增量 | 避免混用两套设定；识别可复用机制 |
| [05-game-adaptation-motifs.md](./05-game-adaptation-motifs.md) | 可改编桥段、事件母题、结局模板与现有事件的映射（已拆分为同名子目录，本文件为索引） | 新事件与结局设计的选题库 |
| [06-silo40-event-plan.md](./06-silo40-event-plan.md) | Silo 40 事件清单、量化校准、验证回归、互调结果与待决项（已拆分为同名子目录，本文件为索引） | 事件分批实现、数值校准与 sim 对照 |
| [../story/silo40-good-endings.md](../story/silo40-good-endings.md) | 《共证》《越岭》《无人裁决》的剧情台阶、玩家抉择、终局画面与三线交叉拆台 | 好结局事件链写作、原创边界与结局资格叙事对接 |

## 内容实现与工程

| 文件 | 内容 | 使用场景 |
| --- | --- | --- |
| [07-content-authoring-guide.md](./07-content-authoring-guide.md) | 内容作者手册：目录分组、加载校验、触发器/效果/变异封闭集、冷却与延迟、思潮真源、验证流程与历史缺陷（已拆分为同名子目录，本文件为索引） | 新增或修改内容事件、机制脚本前查阅 |
| [08-linux-frontend-rendering-perf.md](./08-linux-frontend-rendering-perf.md) | Linux/WebKitGTK 渲染性能审计、修复记录与复核口径 | 排查桌面渲染与动画性能问题 |

## 界面与信息架构

| 文件 | 内容 | 使用场景 |
| --- | --- | --- |
| [09-ui-information-architecture.md](./09-ui-information-architecture.md) | 五个行动域、三级地堡剖面、情境 Dock、空间与行动联动、响应式和性能约束（已拆分为同名子目录，本文件为索引） | 界面信息架构与交互设计 |
| [11-information-fragments-design.md](./11-information-fragments-design.md) | Fact/Version/Observation 三层信息模型、共享视角模板、证据组合、有限 LLM 表层变体（已拆分为同名子目录，本文件为索引） | 信息碎片、证据呈现与文本生成边界设计 |
| [13-action-exposure-incidental-intel.md](./13-action-exposure-incidental-intel.md) | 行动常驻/解锁划分、九部门职务伴随 Observation、既有势力接触与事件门控任务链 | 行动露出、信息获取与第三方委托设计 |

## 权力与组织机制

| 文件 | 内容 | 使用场景 |
| --- | --- | --- |
| [10-it-control-credibility.md](./10-it-control-credibility.md) | 幕后威势、二元关键联盟、处置规程、三方附着、分层确信、检测分类与可打穿平衡目标（已拆分为同名子目录，本文件为索引） | IT/Judicial/Mayor、信息、隐蔽维护与介入机制设计 |
| [12-ending-system-design.md](./12-ending-system-design.md) | 三条胜利路线、好／中性／坏结局谱系、资格闸、互斥排序与四类主体仲裁 | 结局定名、结局判定与多实体加权设计 |
| [13-ideology-axes-design.md](./13-ideology-axes-design.md) | 五条二元思潮轴、复合族命名、开放联盟、玩家推动与自然漂移、TIME 资格映射 | 思潮系统、派系聚类、行动影响与结局资格设计 |
| [14-faction-power-design.md](./14-faction-power-design.md) | 群星派系机制拆解、无代议机构的影响形态、本作派系生成规则与七个权力挂载节点 | 派系生成、满意/积怨计量、议程与街头压力设计 |

## 同类机制对照

| 文件 | 内容 | 使用场景 |
| --- | --- | --- |
| [game-mechanics-breakdown.md](./game-mechanics-breakdown.md) | 同类策略与模拟游戏的机制拆解，区分参考事实和项目推断（已按游戏拆分为同名子目录，本文件为索引） | 比较核心循环、状态、时间、随机性、后果与反馈 |
| [mechanics-comparison.md](./mechanics-comparison.md) | 横向机制对照、项目现状与可借鉴范围（已拆分为同名子目录，本文件为索引） | 新机制立项前的成本与适配性评估 |

## 使用约定

1. 当前行为以代码、测试和 `references/` 项目源参考为准；本目录研究文档不自动代表现状。
2. 新内容只描述抽象机制与原创实现，不复制外部作品的人物、具体剧情、对白或可识别事件链。
3. 历史基线必须注明适用代码版本；旧轨迹和旧设计结论不得直接套到当前版本。
4. 01–06 的“改编侧”内容只保留功能位与机制，不写入具体人名、组织名、集数或纪年。
5. 标注“已拆分”的文档按主题拆成了多个小文件：原文件名是索引，正文在同名子目录。引用具体章节时先查索引里的“原章节 → 子文件映射”表，再进子文件；旧文中以 `§` 编号引用的章节号保持原编号不变。

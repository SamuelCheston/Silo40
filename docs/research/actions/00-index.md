# 情境化动作设计索引

> 本目录只讨论微观、人物驱动的动作设计。每份文件只处理一个问题，可独立阅读。
>
> 全系列共同硬判据：**动作参数必须参与结算，不能只参与文案。**按钮名称、对象图标和剧情描述都不能替代状态差异。

| 文件 | 单一主题 |
| --- | --- |
| [01-reference-spatial-observation.md](./01-reference-spatial-observation.md) | Beholder、This War of Mine、Do Not Feed the Monkeys、Shadows of Doubt 各自可借鉴的对象与信息机制 |
| [02-reference-dialogue-constraints.md](./02-reference-dialogue-constraints.md) | Disco Elysium、Pentiment、Citizen Sleeper、The Occupation 各自可借鉴的对话、身份和机会成本机制 |
| [03-object-model.md](./03-object-model.md) | 人、部门联系人、设施、物件、材料、当前事件的定义、选择与失效 |
| [04-action-tuple.md](./04-action-tuple.md) | 执行者、对象、地点、时机、输入、即时结果、代价与痕迹如何逐项参与结算 |
| [05-button-replacements.md](./05-button-replacements.md) | 八个抽象按钮逐个替换为情境动作，以及后台保留结果 |
| [06-information-production.md](./06-information-production.md) | Observation 的来源链、Fact/Version 对接、披露、组合、去重与回放 |
| [07-relationship-memory.md](./07-relationship-memory.md) | 关系数值与“帮过、交付、隐瞒、泄露”等具体记忆如何共同驱动后续 |
| [08-contextual-ui.md](./08-contextual-ui.md) | 为什么主列表是当前地点的人与物，以及基础职务、解锁动作和往返路径 |
| [09-opportunity-cost.md](./09-opportunity-cost.md) | 时间、地点、身份、NPC 日程和替代路径如何约束动作 |
| [10-risks-antipatterns.md](./10-risks-antipatterns.md) | 重复打工、不可回退、耗时、菜单爆炸、伪记忆和全局函数换皮等否决项 |
| [11-resolution-and-acceptance.md](./11-resolution-and-acceptance.md) | 后端权威契约、结算顺序、确定性、逐项测试与垂直切片 |

## 推荐阅读顺序

1. 先读 03、04，统一对象与动作实例。
2. 再读 05，确认旧按钮如何替换。
3. 按实现主题读 06–09。
4. 施工与验收前必读 10、11。
5. 01、02 仅用于核查外部案例出处，不作为需要复刻的界面模板。

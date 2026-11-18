# ADR-0018 · 后端无玩家/NPC 之分：共用 Actor 接口，玩家即代表人物

| 状态 | 日期 | 依据 |
| --- | --- | --- |
| 已接受 | 2026-11-17 | 项目所有者裁决（会话指令下达），本 ADR 为首次落地；[actor.go](../../internal/engine/actor.go) Actor 抽象 |

## 背景与问题

派系算法重设计（领袖发起事件 + 感知传播 + 加权随机加入）需要定案玩家在后端的身份：是派系体系之外的特殊存在，还是与 NPC 同构的个体。代码已有统一 ActorView（执行引擎只认识视图、不区分来源），但"玩家能否作为派系代表人物"从未裁决；[14 派系权力设计](../../docs/research/14-faction-power-design.md) §6.6 曾把玩家排除在派系成员身份之外，与同篇 §4"玩家与当局共用同一套通道"自相矛盾。

## 决策

- 从游戏后端层面，**没有玩家和 NPC 的区别**：所有个体（Agent / Profession / Resident）共用同一套 Actor 控制接口——同一状态视图、同一动作集、同一资格判定、同一 mutation 结算管线。
- **玩家本身被视为一个代表人物**：可以发起、领导、加入派系，哪怕其圈子里一个人都没有。零成员派系合法存在，与普通派系一样受筹备期人数规则约束。
- 玩家与 NPC 的唯一区别在**决策来源**：NPC 走带权重的随机决策（[ADR-0007](0007-weighted-random-stance.md)），玩家由 UI 直接选择。玩家不掷骰，但发起/加入同样消耗 AP、同样走资格与结算。

## 被否掉的替代方案

- 玩家专属派系接口 / 特殊分支：制造两套控制路径，违反统一 Actor 抽象，派系算法对玩家行为不可解释、不可审计。
- 玩家排除在派系身份之外（14 篇 §6.6 原条款）：把玩家变成派系之上的超然存在，与 14 篇 §4 统一原则冲突；本裁决取代该条款的后端部分。

## 后果与代价

- 取代 [14 派系权力设计](../../docs/research/14-faction-power-design.md) §6.6"玩家不能'加入'派系获得成员身份"的后端排除条款；掩护身份等叙事约束降为内容层的表达问题，不再是后端限制。
- 派系代表引用必须能指向玩家（Agent），不再限定 Resident；派系代表字段、领袖接任规则对玩家与 NPC 同构。
- Agent 补齐 Ambition / Influence 字段（政治威望改为全层由二者派生的独立裁决，随派系新算法设计文档落地）；玩家不消耗加权随机 RNG，确定性契约不变（[08 确定性](../08-determinism-known-issues.md)）。
- 与 [ADR-0010](0010-player-as-intervener.md) 不冲突：代表人物是驻地内的事实身份，不是批准历史的权力；结局仲裁仍只读事实，不得加"玩家必须做过某事"的硬闸。
- 上线前无兼容义务（[ADR-0017](0017-no-backward-compat-before-release.md)）：代表字段形状、存档版本可直接破坏性变更，不写迁移。

## 参考

- [actor.go](../../internal/engine/actor.go)（ActorView 统一状态视图）
- [ADR-0007 带权重的随机决策](0007-weighted-random-stance.md)、[ADR-0010 玩家是干涉者](0010-player-as-intervener.md)、[ADR-0017 上线前无兼容义务](0017-no-backward-compat-before-release.md)
- [14 派系权力设计](../../docs/research/14-faction-power-design.md) §4、§6.6
- [02 架构](../02-architecture.md)（统一 Actor 管线）
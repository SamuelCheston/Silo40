# IT 幕后权力：威势、Protocol 能动性与 Judicial 三方附着

> 日期：2026-11-14（机制边界、平衡与碎片生成器修订）
> 性质：机制设计提案，不包含代码改动。
> 目标：实现一个可被熟练玩家打穿、又会按 Protocol 主动自保的 IT； Judicial 可在 IT / Mayor / covenant 间迁移；碎片、检测、公开证据与 Safeguard 各自分层计算。

本文档已按主题拆分为多个小文件，一份只讲一件事；本文件只做索引，原有信息一条不丢，只改变了分布方式。拆分文件位于 [10-it-control-credibility/](./10-it-control-credibility/) 目录。

## 子文件目录

| 文件 | 内容 |
| --- | --- |
| [01-boundary-and-summary.md](./10-it-control-credibility/01-boundary-and-summary.md) | §0 阅读约定与结论摘要、§1 机制边界、设计裁决与现状差距 |
| [02-naming-and-dimension.md](./10-it-control-credibility/02-naming-and-dimension.md) | §2 命名与量纲（幕后威势候选与选定） |
| [03-prestige-state-and-information.md](./10-it-control-credibility/03-prestige-state-and-information.md) | §3 威势状态与信息系统（状态变量、月度更新、分层事实与版本、碎片获取、公开证据） |
| [04-prestige-sources-and-leverage.md](./10-it-control-credibility/04-prestige-sources-and-leverage.md) | §4 威势来源与玩家有效杠杆 |
| [05-alliance-and-maintenance-actions.md](./10-it-control-credibility/05-alliance-and-maintenance-actions.md) | §5 IT 的二元 selectorate 与 Protocol 能动性（有限资源、隐蔽维护动作、双动机 utility） |
| [06-protocol-assets-and-deviation.md](./10-it-control-credibility/06-protocol-assets-and-deviation.md) | §6.1–6.3 Protocol 内容资产、三类偏离与上层控制方介入链 |
| [07-cognition-gradient.md](./10-it-control-credibility/07-cognition-gradient.md) | §6.4 IT 主任对上层控制方的认知梯度 |
| [08-fragment-generator-overview.md](./10-it-control-credibility/08-fragment-generator-overview.md) | §6.5 上层控制方信息碎片的三类生成器（总览与路径表） |
| [09-relic-generator.md](./10-it-control-credibility/09-relic-generator.md) | §6.5 A Relic 遗物生成器 |
| [10-protocol-reflection-generator.md](./10-it-control-credibility/10-protocol-reflection-generator.md) | §6.5 B IT 主任参悟 Protocol 生成器 |
| [11-control-channel-generator.md](./10-it-control-credibility/11-control-channel-generator.md) | §6.5 C 控制通道对话生成器与三条路径衔接 |
| [12-judicial-adhesion.md](./10-it-control-credibility/12-judicial-adhesion.md) | §7 Judicial 三方附着与可打穿平衡 |
| [13-detection-and-ap.md](./10-it-control-credibility/13-detection-and-ap.md) | §8 检测、AP 与其他机制自洽 |
| [14-parameters-and-audit.md](./10-it-control-credibility/14-parameters-and-audit.md) | §9 参数集中与审计 |
| [15-roadmap-and-acceptance.md](./10-it-control-credibility/15-roadmap-and-acceptance.md) | §10 落地 roadmap（第 1–7 步）、§11 总验收与冲突结论 |

## 原章节 → 子文件映射

| 原章节 | 所在文件 |
| --- | --- |
| §0 + §1（1.1–1.4） | 01-boundary-and-summary.md |
| §2（2.1） | 02-naming-and-dimension.md |
| §3（3.1–3.7） | 03-prestige-state-and-information.md |
| §4 | 04-prestige-sources-and-leverage.md |
| §5（5.1–5.5） | 05-alliance-and-maintenance-actions.md |
| §6.1–6.3 | 06-protocol-assets-and-deviation.md |
| §6.4 | 07-cognition-gradient.md |
| §6.5 总览 | 08-fragment-generator-overview.md |
| §6.5 A | 09-relic-generator.md |
| §6.5 B | 10-protocol-reflection-generator.md |
| §6.5 C + 衔接 | 11-control-channel-generator.md |
| §7（7.1–7.4） | 12-judicial-adhesion.md |
| §8（8.1–8.4） | 13-detection-and-ap.md |
| §9 | 14-parameters-and-audit.md |
| §10 + §11 | 15-roadmap-and-acceptance.md |

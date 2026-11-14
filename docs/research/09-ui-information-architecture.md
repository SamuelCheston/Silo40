# Silo40 界面信息架构重构：五个行动域与三级地堡剖面

> 日期：2026-11-13  
> 性质：研究与施工设计，不包含代码改动。  
> 目标：让玩家在有限空间内回答“此刻能做什么、代价是什么、会留下什么痕迹”，并把楼层、部门、人群变成可点击的行动锚点。

本文档已按主题拆分为多个小文件，一份只讲一件事；本文件只做索引，原有信息一条不丢，只改变了分布方式。拆分文件位于 [09-ui-information-architecture/](./09-ui-information-architecture/) 目录。

## 子文件目录

| 文件 | 内容 |
| --- | --- |
| [01-conventions-and-summary.md](./09-ui-information-architecture/01-conventions-and-summary.md) | §0 阅读约定（四种标记）、§1 结论摘要 |
| [02-current-facts.md](./09-ui-information-architecture/02-current-facts.md) | §2 当前界面与数据事实、设计边界 |
| [03-reference-cases.md](./09-ui-information-architecture/03-reference-cases.md) | §3 参考案例核查与借用边界（HOI4 / Beholder / TWoM / Frostpunk / RimWorld） |
| [04-overall-layout.md](./09-ui-information-architecture/04-overall-layout.md) | §4 总体布局：稳定外壳与单一情境 Dock（含线框） |
| [05-five-action-domains.md](./09-ui-information-architecture/05-five-action-domains.md) | §5 五个行动域的精确定义与增长规则 |
| [06-three-level-sections.md](./09-ui-information-architecture/06-three-level-sections.md) | §6 三级剖面（全堡功能区 / 单层 / Inspector）的线框与导航 |
| [07-section-action-linkage.md](./09-ui-information-architecture/07-section-action-linkage.md) | §7 剖面与行动的联动规则（情境筛选、两条路径、草稿与痕迹） |
| [08-state-visualization.md](./09-ui-information-architecture/08-state-visualization.md) | §8 状态可视化：空间符号优先、Overlay 界线、诚实降级 |
| [09-react-implementation.md](./09-ui-information-architecture/09-react-implementation.md) | §9 React 落地设计（渲染技术、组件拆分、元数据、空间数据、性能约束） |
| [10-small-window-construction-acceptance.md](./09-ui-information-architecture/10-small-window-construction-acceptance.md) | §10 小窗口与低分辨率降级、§11 施工顺序与验收标准、§12 明确不做 |

## 原章节 → 子文件映射

| 原章节 | 所在文件 |
| --- | --- |
| §0 + §1 | 01-conventions-and-summary.md |
| §2（2.1–2.2） | 02-current-facts.md |
| §3（3.1–3.5） | 03-reference-cases.md |
| §4（4.1–4.2） | 04-overall-layout.md |
| §5（5.1–5.4） | 05-five-action-domains.md |
| §6（6.1–6.4） | 06-three-level-sections.md |
| §7（7.1–7.5） | 07-section-action-linkage.md |
| §8（8.1–8.3） | 08-state-visualization.md |
| §9（9.1–9.5） | 09-react-implementation.md |
| §10 + §11 + §12 | 10-small-window-construction-acceptance.md |

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

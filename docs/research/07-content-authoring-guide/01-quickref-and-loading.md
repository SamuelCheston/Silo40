# Silo40 内容作者手册 · 决策速查与目录分组加载链路

> 拆分自 [Silo40 内容作者手册](../07-content-authoring-guide.md)；目录、章节映射与出处见该索引。

## 0. 决策速查

| 我想… | 结论 | 动哪里 |
| --- | --- | --- |
| 加一个条件触发的剧情/危机事件 | 写内容 | `events/special/` 或 `events/crisis/` 新增 `.js` |
| 加一个按时间推进的里程碑 | 写内容 | `events/histories/`，`timestampAtOrAfter(year, month)` |
| 加一个常驻状态规则（恐慌→暴乱这类） | 写内容 | `events/rules/`，`script.canTrigger` + `script.apply` |
| 加一个玩家按钮动作 | 写内容 | `events/player_actions/`（目录未建，建了即生效） |
| 改每回合结算公式（资源/思潮/指标/胜负） | 写内容 | `events/mechanics/*.js`（mechanic 定义，不是内容事件） |
| 发/收信息碎片、改楼层稳定度 | 写内容 | 封闭集效果 `profession_fragment_add/remove`、`floor_stability_delta`（§4） |
| 加新的效果类型/触发器类型/变异类型/flag/结局 | **必须动引擎** | 见 §10，先立项 |

## 1. 目录分组与加载链路

| 分组 | 职责 | 加载方 |
| --- | --- | --- |
| `events/histories/` | 脚本化里程碑（按游戏历触发） | 内容加载（白名单） |
| `events/special/` | 条件性特殊事件 | 内容加载（白名单） |
| `events/crisis/` | special/histories 的后续升级或延迟代价 | 内容加载（白名单） |
| `events/rules/` | 常驻规则（带冷却，脚本判定） | 内容加载（白名单，`8eec461` 起） |
| `events/player_actions/` | 文件驱动玩家动作（尚未建目录） | 内容加载（白名单） |
| `events/mechanics/` | 每回合核心公式与动作定义（`defineMechanics`） | **MechanicEngine 单独加载**，不走内容白名单 |

加载链路：启动 `BootstrapContent` → `resolveContentDirectories`（优先二进制同目录
`events/`，缺失时从内嵌 FS 释放）→ `contentEventDirectories` **白名单过滤**
（`histories/history/special/crisis/rules/player_actions/player_action/events`，
`internal/service/game_service.go`）→ 逐文件 `ValidateContentEventDefinition` →
upsert 进库并禁用已删除者。**任何文件校验失败 = 整次同步失败 = 全部内容事件静默不加载**
（历史事故，见 §12.1）。

**防呆闸**（`internal/service/content_guard_test.go`，`d69cdcb`）：`go test ./internal/service`
时自动拦截——磁盘上有事件文件却被白名单漏掉的分组（死内容）、加载/校验失败、缺可用触发器
或效果、事件 ID 重复，都会红并报出分组/文件/事件名。新加事件分组后先跑它。


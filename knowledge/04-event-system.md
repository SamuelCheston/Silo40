# 内容事件与跨堡信号

## 1. 三种内容配置各司其职

| 形态 | 位置 | 职责 |
| --- | --- | --- |
| `defineEvent` | `events/histories|special|crisis|rules|player_actions/` | 条件、日历、行动和后续事件 |
| `defineMechanics` | `events/mechanics/` | 月度公式、通用/职业行动、stats 和评分 |
| action gate JSON | `events/action_gates.json` | 阶段与行动可见/可用，不定义效果 |

Silo 18 N1–N9 不是普通 `defineEvent`，而是 Go 中的强制正典节点表；跨堡 operation 也是 Go 固定定义。不要用新增 JS 事件冒充这两类能力。

## 2. 加载与发布约束

启动时优先使用二进制同目录 events；缺失才释放嵌入副本。`syncActionGateConfig` 还会把内嵌注册表同步到已有 events 目录，所以修改源 `action_gates.json` 后要重新构建。

内容定义经校验 upsert 到 SQLite，失效定义禁用，启用项按分组/文件/ID 排序。普通内容白名单不含 mechanics；action_gates.json 作为配置单独读取，不被当作剧情文件。

生产缺 action gate 配置会 fail-closed；试玩脚本也拒绝缺文件的旧包。普通内容 Bootstrap 错误在 GUI 启动层记录日志，因此开发与 CI 必须运行内容守卫，不能只看窗口是否出现。

## 3. 事件定义与运行名

一个事件文件包含唯一 `defineEvent({...})`，核心字段为 id/title/type/fire_mode/trigger/effects，可选 cooldown、player_action 和 script hooks。推荐 snake_case 且文件名与 ID 一致。

运行时名为 `category:event_id`。结构化 trigger/effects 优先；`script.canTrigger` 只增加判断，`script.apply` 返回额外 effects/emit，不能直接改 Go 状态。

基础 trigger 包括时间、Silo 指标、资源、人口、flag、event_triggered、player_action_is、部门计数和 all/any。基础 effects 包括指标/资源/人口 delta、部门 panic/productivity/思潮、碎片、楼层稳定度和延迟事件。

Cohort 是思潮真源，所以 `profession_ideology_delta_all` 不是长期改写 cohort 的替代品；部门宏观指标也可能在下一本地相被派生公式重算。

## 4. 多堡 scope、世界事实与跨堡队列

多堡内容效果增加目标 scope：self、world、silo:<ref>、neighbors、all 等，在施加时解析。额外能力包括：

- `cross_signal`：把事件投递到 World.CrossQueue。
- `world_flag_set`：修改白名单世界事实。
- `world_flag_true/false`：读取世界事实。
- `signal_received`：检查本堡本月收到的信号。

当前世界 flag 白名单为 `silo1_destroyed` 和 `network_jammed`；`silo1_destroyed` 与 `!World.Silo1.Alive` 同步，旧 silo flag 读写保留兼容别名。未声明的新 flag 会在加载时失败。

CrossEvent 包含 SourceSilo、TargetScope、EventID、DueYear/DueMonth 和单调 Seq。派发发生在所有堡本地相之后、时钟推进之前，严格按 `(到期时间, Seq)`；目标 scope 再解析为按 SiloID 升序的集合。ONCE 事件按目标堡内容状态防重复。

## 5. 延迟、冷却与正典时间的区别

- ONCE/REPEATABLE 和 cooldown 使用每堡 ContentEventState。
- `schedule_event` 创建 service delayTask，到期后不再检查目标 trigger；纯调度事件常用恒假 canTrigger 防月度提前触发。
- 普通 delayTask 按创建序号稳定排序，多堡时先按目标堡 staging，再在该堡本地相前发布。
- CrossQueue 是堡间消息，不等同于普通 delayTask。
- Silo 18 节点按世界绝对时间在阶段 0c 强制生效，不评估普通条件，不消耗 RNG。

N1 从 Y122 M2 开始，N8 在 Y128 M6 终止 18 号，N9 在 Y130 M1 设置 Silo 1 覆灭事实。N8 的 DEATH 和具名人物状态由 Go 权威覆写；玩家行动和加权结局不能改写 18 号正典。

## 6. 行动完成节点如何连接门控

普通机制行动成功后，service 写入 `player_action:<id>` 内容状态。action_gates.json 的 `event_completed` 再读取这些状态推进阶段或解锁行动。

因此：

- 点击按钮但执行失败，不算完成。
- 只推进月份，不会伪造完成节点。
- 修改 mechanic ID 会同时影响行动路由、完成节点、注册表和存档中的事件 key，必须一起审查。

## 7. 新增内容后的验证

```bash
go test ./internal/service -run 'TestContentGuard|TestRulesGroupIsLoaded' -v
go test ./internal/engine ./internal/service ./internal/api
```

多堡内容还应验证目标 scope、信号到期月、堡序、Silo 18 节点不重复和读档恢复。机制或时序变化使用同 seed parity 前后 diff；生产门控与 parity fixture 的门控加载方式不同，不能拿 ungated 基线宣称生产行动树分布已验收。

完整 DSL 字段以 `events/README.md` 和 Go 校验器为准，历史设计文章只作为背景。

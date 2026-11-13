# 存档版本与兼容规则

## 1. 当前版本与保存对象

当前 `model.CurrentSaveVersion = 7`。一个存档是单活跃 World 的完整对象图：驻地、Silo 1、Silo 18、各堡 Agent、事件历史、内容状态，以及 World 中的跨堡队列、operation、历史影响、ProgressStage、UnlockedActions 和结局 Decision。

普通开局必须满足驻地身份不变量：玩家、Agent.SiloID、Agent.StationSiloID 指向同一驻地；Silo 40 不得出现在 silo 表中；驻地不能带 Silo 1/Silo 18 特质。

## 2. 版本链的真实含义

| 版本 | 引入内容 | 当前启动处理 |
| --- | --- | --- |
| 无标记 legacy | 早期单堡 | 备份后补 v1 标记，进入后续链 |
| v1 | 单堡 Silo 根聚合 | 升级 v2 |
| v2 | World 骨架 | 升级 v3，补跨堡队列/世界 flag/Agent 堡归属 |
| v3 | 双堡原型 | 升级 v4，回填派系持久身份字段 |
| v4 | 派系持久身份 | 升级 v5，初始化 playable operation/历史字段 |
| v5 | 旧可试玩多堡 | **显式拒绝**：旧身份模型不能续玩 |
| v6 | 外派驻地三堡 | **显式拒绝**：缺行动门控和多实体结局状态 |
| v7 | 门控、永久解锁、加权仲裁 | 当前唯一可正常 Resume 的持久化版本 |

所谓 legacy 升级链是 `legacy → v1 → v2 → v3 → v4 → v5`，**不会把旧档伪迁移成可玩的 v7**。到 v5 后，Resume 按身份语义断裂拒绝；v6 也没有伪造 v7 迁移。高于 v7 或其他异常版本同样拒绝。

## 3. 升级安全语义

每一步真实迁移都采用：

1. SQLite `VACUUM INTO` 一致性备份，文件名带时间戳与阶段后缀。
2. 单事务写结构/字段和固定目标版本。
3. 回读校验关键记录与 SaveMeta。
4. 失败回滚并保留备份，成功才进入下一步。

后缀包括 `-v2world`、`-v3twosilo`、`-v4faction`、`-v5playable`。每步写自己的固定版本，不能直接跳到 CurrentSaveVersion 而绕过后续校验。

AutoMigrate 在迁移前创建/补表，但不负责业务语义。拒绝 v5/v6 不等于可以把版本字段改成 7 继续；应保留数据库与备份并开新局。

## 4. 自动与手动保存

- 新开局立即 `persist()`。
- UI 的 Begin/ProcessNext 队列在完整月份/耗时操作结束以及 game over 时落 SQLite；部分即时动作完成路径只更新 FreeCache，并非每次点击都写库。
- Wails `SaveGame` 可保存最近完整状态，但 pending operation 或 EventBus 未排空时明确拒绝。
- 同步 debug `PassTime` / `ExecuteAction` 保留旧的关键节点语义：game over 自动落库，普通进行中主要更新 FreeCache；不要把它误写成每步保存。

FreeCache 是进程内快照，不是跨重启保障。桌面关闭后应从 SQLite 恢复最近完整结算点。

## 5. 三堡对象图的落库方式

保存采用固定 World ID 的整图替换事务。Silo 按内部 ID 1/2/3 写入，Agent 与所属堡关联。

professions/cohorts/factions/residents 在内存中使用堡内 ID；落库时对第 N 堡的主键与相关外键统一加 `(N-1)×10000`，读档减回。resources/floors 不需要偏移。这样避免不同堡子表主键冲突，也保持驻地 ID 1 的历史布局。

事件历史与 ContentEventState 带 SiloID；Silo 18 的 `silo18_chain:N*` 状态防止节点读档重入。World.Silo1.Alive 与完整 Silo 1 的 DEATH 同步，N9 覆灭事实优先。

## 6. Resume 校验与恢复范围

SQLite Resume 会：

- 检查 SaveMeta 精确版本和 World 存在性。
- 按 Silo.ID 升序加载三堡及关联对象，反向偏移子表 ID。
- 加载每堡 Agent、事件历史和内容状态。
- Normalize World.Pacing。
- 验证驻地身份与 40 号不存在。
- 重建引擎、内容 handler、receivedSignals、LLM 回放状态。

v7 能恢复三堡、阶段、永久解锁、operation、历史影响和结局仲裁记录。以下内容仍不是完整续跑状态：

- `delayTasks/delayCounter` 重建为空，未到期普通延迟任务会丢失。
- `pendingOperation` 清空，未完成的 UI 操作不会续跑。
- EventBus 中未处理事件不作为对象图保存。
- gameplay RNG 只从原 seed 重建，未保存当前流位置，详见 [确定性](08-determinism-known-issues.md)。

手动保存要求完整结算点正是为了避免把 pending 操作写入半状态，但它不能替代延迟/RNG 状态的完整持久化。

## 7. 内容版本不等于存档版本

ContentEventDefinition 启动时从当前 events 同步，删除的定义禁用；已有局不会冻结旧脚本。ContentEventState 以 SiloID + EventKey 关联触发历史。

修改事件 ID、mechanic ID、完成节点或分组可能切断旧状态与门控条件。即使 SaveVersion 不变，也要审查兼容性；涉及 World 阶段、永久解锁或仲裁语义断裂时应提升版本并明确拒绝或迁移。

## 8. 修改存档时的检查清单

1. 破坏性模型/语义变化提升 CurrentSaveVersion 并写版本历史。
2. 明确选择“真实迁移”或“显式拒绝”，不伪造兼容。
3. 真实迁移保留备份、事务、回读和失败回滚。
4. 覆盖三堡对象图、ID 偏移、operation、门控、Decision 与正典状态。
5. 测试空档、v7、v5/v6、高版本、缺 World、身份错误、迁移失败。
6. 做存读档与同 seed 对照，同时披露 RNG/延迟任务仍未完整恢复的事实。

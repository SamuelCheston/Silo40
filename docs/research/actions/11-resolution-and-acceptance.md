# 情境动作结算与验收规范

> 目的：把对象模型、七元组、Observation 和关系记忆收敛为可实施的结算契约与测试清单。
>
> 硬判据：**动作参数必须参与结算，不能只参与文案。**验收以状态差异和回放证据为准，不以按钮截图或文案审读代替。

## 1. 权威边界

后端负责：

- 对象存在性、可见性、权限和日程；
- 从上下文生成合法 ActionInstance；
- 七元组校验、成本、主结果、Observation、关系记忆和事件；
- RNG、完成事件键、存档与回放。

前端负责：

- 展示当前地点的人与物；
- 绑定用户选择并请求后端生成实例；
- 展示不可用原因、主结果、材料、记忆与痕迹；
- 不自行推断成功率、材料池、权限、曝光或全局结果。

前端不能只把旧按钮包进对象弹窗，也不能提交自造的 `enabled=true` 或结果数值。

## 2. 推荐数据契约

### ActionInstanceRequest

```text
template_id
actor_id
primary_target: ObjectRef
secondary_targets: ObjectRef[]
location: LocationRef
timing_choice: WindowRef
input_refs: InputRef[]
choice_id?
```

### ActionPreview

```text
instance_id / binding_hash
localized_action_label
resolved_actor/target/location/timing summaries
required_inputs and missing_inputs
immediate_result_preview
cost_preview
trace_preview
available + disabled_reason[]
```

预览只能来自后端当前上下文。`binding_hash` 用于防止玩家在预览后偷换对象、材料或窗口。

### ActionResult

```text
instance_id
primary_outcomes[]
observation_results[]
relationship_memory_results[]
disclosure_results[]
costs_applied[]
traces_applied[]
completed_event_keys[]
scheduled_events[]
newly_available_actions[]
```

若某项为空，返回空结果，不生成假卡。

## 3. 结算管线与原子性

```text
1. 根据当前选择重新绑定对象
2. 校验七元组、对象存在性、权限、输入和窗口
3. 校验并预留 AP、物品、事件防重入
4. 执行主结果
5. 固定顺序处理 Observation、关系记忆、披露和痕迹
6. 推进时间与事件，写完成键
7. 调度后续事件并刷新门控
8. 保存完整结算点
9. 返回结果
```

- 任何校验失败都不得扣 AP 或产生部分结果。
- 主行动失败不进入 Observation 阶段。
- 材料为空不影响主行动成功。
- 重复完成键不能重复写结果。
- 读档、重复结果卡和重复 UI 点击不能重复结算。

## 4. 事件与门控接口

### 完成键

```text
player_action:<action_instance/template>
observation:<observation_id>
contact:<offer_id>:accepted
contact:<offer_id>:declined
contact:<task_id>:completed
contact:<task_id>:expired
```

键的设计必须支持防重入和条件查询，不能只用于日志标题。

### 门控

- 基础职务：空 `unlock`，但 `available` 仍校验对象、AP、地点和时机。
- 进阶职务：按 ID 的履历、事件、关系或阶段 `unlock`。
- 新解锁只前进；实时条件回落只转 UNLOCKED。
- 执行 API 必须再次校验，不能仅靠前端隐藏。

## 5. 确定性与 RNG

1. 主结果、Observation、关系记忆和事件按固定阶段顺序执行。
2. 多对象按稳定 ID 排序，禁止 map 顺序改变结果。
3. 只有材料池非空且确需选择时消费 Observation RNG。
4. 首个里程碑 Observation 由内容保证，不靠随机撞出。
5. LLM 不选择 Fact、Version、对象、记忆、门控或结局，不消费 gameplay RNG。
6. 新动作体系建立受控 sim 新基线，不能只以“UI 改动”声明 parity 不变。

## 6. 七元组逐项测试

| 字段 | 必测差异 |
| --- | --- |
| 执行者 | 无职业/权限者不能执行；不同职业使用不同材料与结果 |
| 对象 | 两名联系人/两座设施得到不同主结果或记忆 |
| 地点 | 不同 zone 的材料来源、传播或设施结果不同；无楼层不伪造楼层 |
| 时机 | 窗口前不可用、窗口内可执行、窗口后进入替代路径；时长真实推进 clocks |
| 输入 | 换 Observation、工具或许可会改变可用性和结果；缺失时给具体原因 |
| 即时结果 | 按钮承诺的设施、对象、材料或事件主结果真实写入 |
| 代价与痕迹 | AP、时间、见证、披露、Suspicion、Exposure 各自归属正确 |

任何字段只改 UI 文本、不改状态，测试应失败。

## 7. 八按钮替换测试

### Build Network

- 两名联系人需要不同帮助；完成后产生不同记忆和后续选项。
- 同一帮助重复执行不重复提供完整收益。

### Share Info

- 同一 Observation 向 Judicial 与 Supply 出示产生不同核验与披露结果。
- 重复出示给同一人返回已读，不重复接受或思潮收益。

### Gather Info

- 动作目录中不存在独立 Gather。
- 主职务无材料仍成功，有材料时来源与实际行为一致。

### Conduct Propaganda

- 不同材料 Version、受众和渠道产生不同覆盖与结果。
- 不能无受众直接增加全局传播效果。

### Incite Rebellion

- 无既有主体和具体诉求时不可执行。
- 结果范围由实际参与者决定，不默认覆盖全部平民，不新建专用反抗组织。

### Publicize Faction

- 只有真实派系、具体主张和公开受众才可执行。
- 公开范围与受众有记录，不等于全堡自动支持。

### STATION_DECEIVE

- 必须选择伪造材料、链路和读取窗口。
- 延迟、Exposure、反间谍和失效状态正确，不能即时直接改目标。

### STATION_TRADE

- 必须存在联系人、物品和承诺；交付后才改变资源与关系。
- 不同联系人不会提供完全相同的兑换结果。

## 8. Observation 与关系记忆测试

- 每份材料都能回溯 observer、location、action、channel、Fact 和 Version。
- 同一来源转述不能刷独立证据；副本保留 independence key。
- 材料为空不消耗 RNG，重复获取不重复计数。
- 关系数值与记忆原子写入，读档不重复。
- 每条记忆至少有一个后续消费方；个人记忆不自动同步到部门。
- 隐瞒与公开留下不同披露路径，不是同一 Share 结果的正负号。

## 9. 界面验收

- 首屏展示当前地点的人与物，无全局 Build/Share 按钮。
- 对象上只出现当前相关动作；基础和进阶有明确区分。
- 不可用原因具体到 AP、对象、材料、权限或窗口。
- 材料、对象、事件之间往返不丢失上下文。
- 小窗口可完成绑定、预览、执行和返回。
- 已解决动作退出主列表，不鼓励全点一遍。

## 10. 推荐垂直切片

### 切片 A：对象与七元组

实现 Supply 仓库、仓管、异常批次和申领单，完成“核对库存→形成 Observation→归还单据→关系记忆”。

### 切片 B：同一材料多对象

把同一申领单分别出示给 Judicial 调查员和 Supply 仓管，证明核验、披露、记忆与后续不同。

### 切片 C：Medical 多地点观察

在不同 zone 巡诊，得到同一 Fact 的不同 Version，验证来源、盲点和去重。

### 切片 D：机会成本与替代路径

设置一次配给交接窗口，验证到场、转交、错过后的材料路径，以及时钟真实推进。

### 切片 E：一个既有派系公开事件

只使用真实派系和现有诉求，验证公开主体、场合、受众和后续，不建立新组织。

## 11. 完成定义

只有同时满足以下条件，才能宣称情境动作设计落地：

- [ ] 八个旧抽象按钮已从玩家入口移除或按清单替换。
- [ ] 所有执行动作可还原完整七元组。
- [ ] 至少两个同类对象在同一模板上产生可证明的不同结果。
- [ ] Observation 和关系记忆都有后续消费方。
- [ ] 时间、地点、身份和日程各至少约束一条真实路径。
- [ ] 无“不同文案、同一全局结果”的换皮动作。
- [ ] 同种子双跑一致，读档和重复点击不重复结算。
- [ ] 关键路径无无预告唯一窗口和硬性死锁。

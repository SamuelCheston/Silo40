# Silo40 行动露出、职务伴随信息与第三方接触设计

> 日期：2026-11-14
> 基线：`127abdc`
> 性质：研究与机制设计，不包含代码改动。
> 范围：行动的三态露出、职务行动附带 Observation、既有势力向玩家递任务。不得新造专用反抗组织，不改动私库。

## 0. 阅读约定与结论

- **【现状事实】**：已在基线代码、配置或项目文档中核实。
- **【参考事实】**：已在外部游戏资料中核实，只用于机制对照。
- **【设计提案】**：本文建议的新行为，不声称已经实现。
- **【能力缺口】**：现有系统不能直接表达，实施时必须增量补齐或缩小首版范围。

核心结论：

1. **常驻与解锁是“目录露出”问题，可用与不可用是“实时条件”问题，不能混谈。**常驻行动也可能因 AP、目标或材料不足而灰置；解锁行动一旦露出便永久保留，之后条件回落只会变成灰置，不重新隐藏。
2. **每个职业有一项基础职务常驻，另一项进阶职务按阶段、事件或关系解锁。**通用 Build Network、Share Info 作为基础能力常驻；高影响、跨堡和进阶秘密行动继续走三态门控。
3. **不再保留任何“唯一主奖励是信息”的按钮。**通用 Gather Info、两个 Intel operation 移除；四个直接取碎片的职业行动改造成具有独立职务结果的行动。信息是主行动成功后的伴随 Observation，不另收 AP、不另开调查按钮。
4. **首个本地 Observation 取代 Gather Info 成为 `first_signal` 的来源。**九职业的基础职务都必须有合法的首份材料池，不能出现某职业开局无法推进的硬死锁。
5. **第三方任务采用“既有实体接触 → 固定选择 → 事件完成键 → 门控/后续事件”的链。**接触方只能引用当前仍存在的 faction、department 或 external channel；不建立专用反抗组织，不复制外部作品的组织与剧情。

## 1. 基线现状与能力边界

### 1.1 三态门控已经把两类条件分开

【现状事实】`events/action_gates.json` 与后端判定器已经实现：

```text
LOCKED：未解锁，不下发行动
  │ unlock 首次满足，key 永久写入 World.UnlockedActions
  ▼
UNLOCKED：已解锁但当前不可执行，下发 DisabledReason
  │ available、目标条件、既有 Enabled 与 AP 全部满足
  ▼
AVAILABLE：可执行
```

`unlock` 只向前，`available` 每次快照重算；执行 API 会再次校验，不能靠 UI 绕过。[action_gates.go](../../internal/service/action_gates.go#L199-L229) [applyActionGate](../../internal/service/action_gates.go#L347-L397) [validateActionGate](../../internal/service/action_gates.go#L417-L443)

【现状事实】当前配置只有 `action_gather_info` 是空 `unlock`、空 `available`；`first_signal` 又由它的完成事件驱动。全部职业行动目前通过 `action_type=PROFESSION_ACTION` 在第一信号后统一解锁，尚未按职业行动 ID 分化。[action_gates.json](../../events/action_gates.json)

【现状事实】判定器优先按行动 ID、再按 action type 找门控，因此可以在不推翻现有结构的情况下，把基础职务与进阶职务拆开。[gateForAction](../../internal/service/action_gates.go#L328-L335)

### 1.2 当前情报专用入口不止一个

【现状事实】以下行动的主描述或唯一主要奖励直接是 fragment：

| 当前入口 | 当前直接信息效果 | 本文处置 |
| --- | --- | --- |
| `GATHER_INFO` | 从目标部门未知 ID 中随机取一份 | **移除按钮与完成事件依赖** |
| `JUDICIAL_SEARCH_WARRANT` | 取一份 fragment，目标恐慌增加 | 改为合规搜查，执法结果是主效果，材料是伴随 Observation |
| `POLICE_INTERROGATE` | 取一份 fragment，目标恐慌增加 | 改为案件处置，处置结果是主效果，证词是伴随 Observation |
| `MECHANICAL_PIPE_TAP` | 通过管线偷听并取一份 fragment | 改为管网巡检，设备/稳定结果是主效果，声音与日志是伴随 Observation |
| `AGRICULTURAL_FIELD_GOSSIP` | 取一份 fragment并改变思潮 | 改为田间轮班，生产与人群结果是主效果，谈话是伴随 Observation |
| `STATION_INTEL` | 一个月后得到驻地精确报告 | **移除独立 operation**；报告改由驻地行动和联系人伴随产生 |
| `SILO1_INTEL` | 一个月后得到上层控制方精确报告 | **移除独立 operation**；报告改由既有信道和实际 operation 伴随产生 |

这些不是改名即可解决的问题。只把按钮从“收集情报”换成“打听消息”，而主奖励仍只有 fragment，仍属于本文否决的专用情报按钮。[common_actions.js](../../events/mechanics/common_actions.js#L1-L28) [profession_actions.js](../../events/mechanics/profession_actions.js)

### 1.3 信息模型已有设计基础，但当前运行模型仍较薄

【现状事实】当前 `KnownFragments` 主要是字符串 ID 集合，通用行动按目标前缀从未知 ID 中抽取。既定信息设计已经区分 `Fact / Version / Observation`，并要求记录 `source_role / source_location / channel / acquisition_event_id`；这为“同一件事被不同职务看到不同版本”提供了直接接口。[11-information-fragments-design.md](./11-information-fragments-design.md)

【设计提案】实施伴随信息时，`KnownFragments` 继续作为兼容计数投影，但新增材料必须先形成一份 Observation，再写入其 legacy alias。不能先随机抽 ID、再补一个看似合理的来源；来源和版本必须先于卡片实例化确定。此前 09/11 文档中保留 Gather Info 的旧提案由本文取代；沿用的是三态界面与 Observation 模型，而不是旧的信息获取入口。

### 1.4 事件系统能承载任务链，但不是完整 quest 系统

【现状事实】现有能力包括：

- `event_triggered` 与 `player_action_is` 触发器。[content_events.go](../../internal/engine/content_events.go#L418-L427)
- `events/player_actions/` 的内容行动，支持 common/profession/faction 等 scope、目标类型、AP 与显隐行为。[game_service.go](../../internal/service/game_service.go#L2810-L2851)
- 行动完成后写入 `player_action:<id>`，可供 `event_completed` 门控消费。[completePlayerAction](../../internal/service/game_service.go#L1330-L1345)
- `schedule_event`、受限脚本 `emit`、事件完成状态和延迟事件。[content_events.go](../../internal/engine/content_events.go#L470-L485) [game_service.go](../../internal/service/game_service.go#L2389-L2488)
- faction 具有持久 ID、representative、stance、memory、demands，脚本上下文可以只读访问。

【能力缺口】当前没有通用 quest 实例、自由对话、任务目标追踪、可取消延迟任务，以及修改 faction memory 的封闭效果。`faction_member` scope 当前仍固定返回 false，不能把接触任务假装成已经可用的派系专属行动。[matchesPlayerActionScope](../../internal/service/game_service.go#L2854-L2871)

因此首版应使用**一次性事件链和固定选择**，不先造任务系统。 repeatable 实例、复杂目标计数、自由对话和动态 faction 关系后置。

## 2. 外部参考核查

### 2.1 HOI4：常驻与 focus 解锁不是两套互斥系统

【参考事实】常规政治 Decisions 也可能有可见与执行门槛。例如部分通用政治行动不要求 focus，但仍要求执政条件、支持率、稳定度或 Political Power，因此“常驻可见”不等于“永远可点”。[cite:1][cite:2]

【参考事实】具体国家可以通过 focus 的 completion reward 露出新 Decisions。土耳其相关 focus 明确写出解锁一组法律 Decisions，解锁后的各项行动仍有各自 PP 成本与条件。[cite:3][cite:4]

【参考事实】Decision 的 `visible` 决定是否出现在面板，`available` 决定能否执行，不满足时可保留灰置；focus 的奖励在 `completion_reward` 中执行，具体 Decision 也可检查 focus 完成状态，不能概括成一个统一的 `unlock` 字段。[cite:5][cite:6]

**借用边界【设计提案】**：

- 借“基础目录长期存在”和“新能力由明确来源解锁”的认知差异。
- 灰置只解释当前缺项，必须同时显示它已解锁，避免玩家误以为还要完成某个剧情节点。
- 不复制国策树、PP 或国家层级；Silo40 的解锁来源应是职务履历、地方事件、关系和既有势力接触。

### 2.2 Beholder：发现、入档、使用是三个环节

【参考事实】官方介绍包含趁租客外出搜查物品、安装摄像头、建立档案和提交报告。实际搜查后，还需把物品描述或监视行为记入住客档案；对话信息也能入档。[cite:7][cite:9]

【参考事实】普通物件可以形成习惯或倾向信息，违禁物与目击违法行为可用于报告、逮捕或其他处置。信息不是单纯的 collectible，后续用途与物件性质、任务和玩家选择相关。[cite:8][cite:9][cite:11]

【参考事实】“偶然发现”不宜理解为纯随机掉落。教程和任务会明确要求主动搜查，准确表述是**玩家在执行空间/职务行动时发现材料**，而不是点击一个自动生成情报的全局按钮。[cite:9][cite:10]

**借用边界【设计提案】**：

- 借“本职行动 → 发现材料 → 材料入档 → 决定用途”的链，不借入室偷窃与主角移动。
- 普通材料与违法材料都可以是 Observation；事实、解释和法律用途不能合并成“拿到违禁物=知道全部真相”。
- 职务行动必须有独立主结果，信息不能只是换皮后的唯一奖励。

### 2.3 Beholder 第三方运动：以人物接触串起任务，而非开启阵营菜单

【参考事实】第三方运动先通过具体人物在住所附近接触玩家，再以对话交付物件或委托；后续通过电话、安置、监视和既有物件系统推进。玩家可接受、拒绝、私自处理或向不同主体报告，影响报酬、人物安危和后续路线。[cite:10][cite:11][cite:12]

【参考事实】后续任务与结局取决于一系列实际选择，而非仅仅“接触过该势力”。[cite:11][cite:14]

【参考事实】资料对首个箱子委托是否构成后续请求的硬前置存在不一致：后续请求页面与首个委托页面/流程攻略的表述并不完全相同。[cite:10][cite:12][cite:13] 因此本项目不复制固定前置链，只借“人物接触—固定选择—复用日常行动—延迟后果”的形态。

## 3. 常驻与解锁的划分

### 3.1 判定顺序

【设计提案】每个行动按以下顺序决定露出方式：

1. **是否属于当前身份的最低履职能力？**是则基础常驻。
2. **是否只是同一能力的材料、AP 或目标限制？**是则不设解锁，只设 `available`。
3. **是否引入新的权限、风险等级、对象或剧情后果？**是则设 `unlock`。
4. **解锁后是否仍受动态条件约束？**是则保留 `available + reason`，不重新锁定。

禁止把“玩家还没用过”“AP 不够”“不知道该选哪个目标”当成剧情解锁理由。

### 3.2 当前通用与秘密行动的处置

| 行动 | 露出结论 | 实时可用条件 | 玩家看到的来源说明 |
| --- | --- | --- | --- |
| Build Network | **常驻** | AP、合法部门目标 | 基础社交能力，无锁标 |
| Share Info | **常驻** | 至少一份已知材料、AP、目标 | 基础信息处置能力；不是收集按钮 |
| Propaganda | 关系解锁 | `first_signal`、AP | “Mayor 关系打开公开表达渠道” |
| Incite Rebellion | 威望解锁 | Mayor 关系、AP | “地方号召力达到新层级” |
| Publicize Faction | 阶段与代表身份解锁 | 威望及既有代表条件 | “档案阶段与既有派系身份共同开放” |
| 基础职务（每职业一项） | **常驻** | AP、目标、既有目标条件 | “当前职业的基础职责” |
| 进阶职务（每职业一项） | 阶段/履历/关系解锁 | 各行动实时条件 | 显示首次解锁来源 |
| STATION_INFILTRATE / DECEIVE / TRADE | 保持解锁制 | 关系、宣传、阶段、AP | 属于新权限和高风险通道，不改为常驻 |
| SILO1_DECEIVE | 保持解锁制 | 阶段、宣传、AP | 上层通道开放后才露出 |
| Gather Info、两个 Intel operation | **移除** | 不适用 | 不以另一个名字复活 |

### 3.3 三态在界面上的区别

| 状态 | 主行动列表 | 标记与文案 | 解锁来源 |
| --- | --- | --- | --- |
| 常驻但当前不可用 | 保留 | 无锁；灰置并写“缺 AP / 缺材料 / 缺目标” | 不需要 |
| `LOCKED` | 默认不下发、不占列表 | 只在路线/接触页显示作者允许预告的“能力线索”，不提供执行入口 | 写事件、职务履历或关系来源，不写模糊“继续游戏” |
| 刚进入 `UNLOCKED` | 出现并带“新解锁” | 开锁图标 + “已解锁，暂不可用”；缺项用当前 `DisabledReason` | 明示由哪个事件/履历解锁 |
| `UNLOCKED → AVAILABLE` | 保留，锁标消失 | 正常行动行；仍可带“职务进阶/事件”来源 tag | 可在详情回看 |
| 解锁后条件回落 | 保留灰置，不重新隐藏 | “已解锁，当前条件未满足” | 不再次播解锁提示 |

【能力缺口】当前 LOCKED 完全不下发，且 `UnlockedActions` 只保存 key，没有解锁时间与来源。若需要路线预告、跨读档“新”标和解锁履历，应增量提供 `unlock_preview` 或 `UnlockRecord`；不能要求前端从 action ID 猜来源。首版可只做当前快照内的解锁 toast 与灰项原因，不强行补全历史。

## 4. 职务伴随 Observation

### 4.1 基础与进阶职务及信息渠道总表

下表同时给出常驻/进阶划分和每个部门天然看到的东西。基础职务不是“低概率调查”，而是开局即可履行的本职工作；进阶职务带来新的权限或处置方式。

| 部门 | 基础职务（常驻） | 进阶职务（建议解锁来源） | 伴随信息渠道 | 天然盲点 |
| --- | --- | --- | --- | --- |
| **Mayor** | Public Address：公开讲话与人群反馈 | Direct Order：首次讲话后，获得正式调度入口 | 现场提问、请愿、掌声/沉默差异、部门回执、命令被改写的环节 | 看到官方口径和公开反应，不能直接证明私下执行细节 |
| **Judicial** | 合规搜查：完成法定检查与物品处置 | Arrest：已有合规搜查履历或案件事件后开放 | 物件及法律状态、封存记录、证词、搜查令与现场事实不一致处 | 能证明合法性/程序，不自动证明动机；合法物件也能提供习惯信息 |
| **IT** | Surveillance：维持监控与链路权限 | Cover-Up：档案阶段后开放 | 访问日志、异常在线时段、行为模式、删除队列、索引缺口 | 看到系统记录，不知道离线谈话；日志本身也可能是被清理过的版本 |
| **Police** | 案件处置：完成拘押/释放/秩序处理 | Crackdown：第一信号或正式秩序事件后开放 | 嫌疑人陈述、目击者矛盾、查扣物、出警记录、同一事件的不同说法 | 口供受恐惧影响，不能把供述当客观事实 |
| **Medical** | Community Treatment：逐层巡诊与处置 | Quarantine：完成首次巡诊或公共卫生事件后开放 | 症状分布、药物使用、就诊者叙述、不同楼层对同一事件的不同版本、库存异常 | 知道身体和病史，不知道完整命令链；患者转述可能受立场影响 |
| **Supply** | Ration Allocation：配给与库存调度 | Shelter：第一信号后开放 | 申领单、缺斤短两、替代物、封存箱、流向异常、未登记库存 | 能核验物流，不知道物品最终用途和决策者动机 |
| **Mechanical** | Overhaul：设备检修与恢复 | 管网巡检：首次检修后开放 | 维修单、非授权改动、振动与压力异常、管线传来的谈话、旧改造痕迹 | 能证明设备状态，听到的对话没有完整上下文 |
| **Mines** | Deep Excavation：掘进与轮班 | Tunnel Network：下层关系或掘进履历开放 | 岩层中的旧物、未登记通道、班次传闻、爆破记录、图纸与实际空间差异 | 看到物理痕迹，难以确认地上部门的正式解释 |
| **Agricultural** | Intensive Harvest：生产与田间轮班 | Field Rotation：首次生产周期后开放 | 作物异常、配给质量、田间谈话、班次记录、不同生产区的说法 | 熟悉生产和基层舆情，看不到高层原始命令 |

### 4.2 为什么这些渠道不是“换皮 Gather Info”

一份伴随材料必须同时满足四条：

1. **主行动独立成立**：没有新材料时，职务行动仍有资源、秩序、关系、设施或人群结果。
2. **来源由工作过程产生**：维修看到设备，配给看到库存，行医听到患者叙述；不能从全局未知 ID 池随机抽一张。
3. **材料只覆盖该渠道的观察范围**：同一 Fact 可以有多个 Version，但不同来源不能互相补齐自己不可能知道的内容。
4. **主行动失败则不产出**：不能用失败行动刷材料；没有合格材料也不让主行动失败。

【设计提案】行动结果 UI 先显示职务结果，再显示次级的“顺带留下了一份材料”。没有材料时只显示职务结果，不出现“调查失败”或空情报卡。

### 4.3 结算管线

```text
玩家执行职务行动
  → 既有门控/目标/AP 校验
  → 主行动成功并完成既有 mutations
  → Observation hook 在固定阶段求值
      1. 读取 action_id、实际 target、zone/floor anchor、已发生 Fact
      2. 过滤 channel access、source role、可见性和去重键
      3. 从该职务的合法材料池确定 Fact + Version + channel
      4. 写入 Observation 与 acquisition_event_id
      5. 投影 legacy fragment alias / KnownFragments
  → 写入 player_action:<action_id> 与 observation 完成键
  → 刷新 action gates
  → 返回“职务结果 + 可选材料卡”
```

【设计提案】建议的声明字段：

| 字段 | 作用 |
| --- | --- |
| `action_id` | 绑定职务行动，不单独形成行动 |
| `channel` | rounds / inspection / inventory / maintenance / testimony / public_work 等 |
| `fact_pool` | 允许观察的事实及版本，不含全局无关 ID |
| `eligibility` | 已发生事件、目标、职业、关系、区域和已知状态 |
| `location_policy` | 只使用权威 floor/zone；缺数据时停在 zone |
| `guarantee_key` | 首份本地 Observation 等里程碑保证 |
| `dedupe_key` | observer × fact × version × channel × source scope |
| `acquisition_event_id` | 把材料回溯到实际行动和事件，供任务/门控使用 |

RNG 只在主行动成功后、材料池非空且需要选择时按固定顺序消费现有流；里程碑首份材料应直接确定，不额外抽取。新增 hook 会改变行动轨迹，实施时必须建立受控新基线和同种子前后对照，不能声称行为中性。

### 4.4 开局阶段如何摆脱 Gather Info

【设计提案】把阶段改为：

```text
arrival
  │ 任一基础职务产生首份有效本地 Observation
  ▼
first_signal
  │ 完成一次 Share Info
  ▼
archive_awakened
```

- 每个职业的基础职务都配置 `guarantee_key=first_local_observation`，内容加载时校验其初始合法材料池非空。
- 首份 Observation 同时写 `observation:first_local`，`first_signal.when` 消费该键；不再消费 `player_action:action_gather_info`。
- 重复材料不推进阶段，空转月份不推进阶段，主行动失败不推进阶段。
- 九职业都要跑“开局 → 首次基础职务 → 首份材料 → first_signal”的测试，不能只测当前推荐职业。

### 4.5 Medical 示例：职务路线天然产生信息差

【设计提案】Medical 巡诊不需要一个“调查楼层”按钮。玩家选择 Community Treatment 后，系统先完成治疗与恐慌/生产力结果，再根据实际目标部门、权威楼层/zone、已发生 Fact 和巡诊渠道生成材料。

同一“配给变更”事件可以留下三份 Observation：

| 巡诊来源 | 玩家看到的材料 | 支持与盲点 |
| --- | --- | --- |
| Upper 就诊者 | 将不适解释为新的健康管理安排，能提供正式通知和就诊时间 | 支持“官方如何解释”，不能证明配给实际短缺 |
| Mid 就诊者 | 多名患者描述替代食物与不适，能提供症状和班次关联 | 支持“影响正在发生”，不能直接证明责任部门 |
| Lower 就诊者 | 患者拿出与上层通知不同的配给说法，并提及某次临时调换 | 提供另一个 Version 与物流线索，仍只是患者经历 |

三份材料指向同一 Fact，却有不同 `source_location / channel / version / blind_spot`。信息差来自行医路线和接触人群，不来自 Medical 拥有更高的“调查成功率”。没有权威楼层时只标 zone，不能为了文案随机分配楼层。

### 4.6 去重、失败与空结果

- 同一 `dedupe_key` 已存在时，主行动照常结算，不重复增加 KnownFragments、不重复推进门控、不弹新卡。
- 没有合法材料时不消费 Observation RNG，不生成空卡，不影响主行动成功。
- 首份里程碑材料必须存在，属于内容完整性校验，而不是靠玩家反复刷行动撞概率。
- 材料获取与 LLM 文本变体分离；模型失败不能影响主行动、Observation、门控或 RNG。
- 证人死亡、设施损毁、目标失联等权威状态必须先过滤材料池，不能让死者继续提供新证词。

## 5. 既有势力接触与递任务

### 5.1 接触方只能是现有实体

【设计提案】使用统一 `source_ref`，但不新增组织类型：

```text
faction:<persistent_id>       当前仍存在的派系及其代表
department:<profession_id>    当前部门或其已有联系人
channel:<existing_channel>    已建立的外部信道，不伪造 40 号总部
```

- 接触必须附着于当前真实实体；派系消失、联系人死亡或信道中断时，后续事件按显式失败/失效处理，不能继续收到“幽灵任务”。
- 未公开 faction 只有在玩家已经通过关系、材料或事件确认其存在后才能递任务；不能突然显示完整派系资料。
- 既有事件中的抽象“地下网络”不是自动可用的联系人，除非它已对应真实 faction/department/contact。
- 不新造专用反抗组织，不把流散后代识别为组织，不恢复 40 号指挥链。

### 5.2 接触形态

**推荐形态：一张有来源的接触卡 + 固定选择，不新增常驻阵营菜单。**

```text
[Supply 联系人 · 工作接触]
对方在你完成配给后留下了一个异常申领单，希望你看完再决定。

[接下核对] [拒绝并退还] [暂不回应]
```

- 接触发生在与来源相符的位置：department 出现在工作/楼层上下文，faction 出现在已知联系人或 Inspector，external channel 出现在世界目标卡。
- 选择是 `PLAYER_EVENT` 内容行动，不是自由聊天；接受、拒绝、暂缓各有明确事件键。
- 未决定前只保留一张接触卡，不反复在每月日志刷屏。
- 接触卡与行动列表使用同一门控真源，但可用 `presentation=contact` 让 UI 不把它当普通职务按钮。该字段属于增量能力。

### 5.3 挂到既有事件与门控的标准链

```text
条件满足（关系 / 阶段 / Observation / 既有事件）
  → action_gates 解锁 contact_offer
  → 玩家打开接触并完成 offer player_action
  → player_action:contact_offer 完成
  → accept / decline / defer 三个内容行动解锁
  → 选择写入各自 player_action 完成键
  → 接受键解锁任务步骤，或触发既有职务监听事件
  → 任务步骤完成，写 player_action:<step> / special:<result>
  → schedule_event 产生后果；action_gates 解锁下一选择
```

可直接复用：

| 需求 | 现有接口 |
| --- | --- |
| 关系/阶段/材料/事件使接触露出 | `action_gates.json` 的 `unlock` 与 `available` |
| 接受/拒绝作为固定选择 | `events/player_actions/*.js` + `PLAYER_EVENT` |
| 选择后永久露出后续行动 | `event_completed=player_action:<choice>` |
| 用日常职务完成委托 | `player_action_is` + accepted event 的 `all` |
| 延迟后果 | `schedule_event` |
| 多分支后续 | 不同选择对应不同完成键、事件与门控 |
| 超时 | 接受时 schedule expiry；到期事件先检查完成/失败状态，已结束则不生效 |

首版不需要 quest 实例。每个一次性任务以唯一事件键表示 offer/accepted/step/result/expired，ContentEventState 就是状态记录。 repeatable 任务、多实例同 ID 委托、复杂目标计数和自由对话才需要正式任务实例。

### 5.4 原创接入示例：Supply 工作委托

以下示例展示如何完全落在既有部门上，不建立新组织。

1. **接触露出**：玩家达到 `first_signal`，且 Supply 关系达到配置阈值；`action_gates` 解锁 `contact_supply_audit_offer`。
2. **接触内容**：Supply 联系人出示一张异常申领单，请玩家核对后续配给。玩家可接、拒绝或暂缓。
3. **接受**：`player_action:contact_supply_audit_accept` 完成，解锁委托步骤，并 schedule 一个 guarded expiry。
4. **履行**：玩家后续完成指定的 Supply 职务。一个 `special` 事件用 `all(accepted, playerActionIs(...))` 监听，不新增“完成委托”万能按钮。
5. **选择用途**：核对后，玩家把材料交给 Mayor、退回 Supply 联系人或留作私人档案。三个选择分别写不同事件键。
6. **后果**：首版使用既有资源、恐慌、碎片、调度与门控改变路线；关系变化仍由职务行动产生。若需要事件直接改关系或写 faction memory，另增封闭效果，不改写固定 Stance，不新增结局类型。

这个例子中，异常申领单既是职务 Observation，也是接触与后续任务的输入。信息不再是收集终点，而是既有势力开始相信玩家“能办事”的依据。

### 5.5 选择与后果的规则

- 每个选择必须写清**立即结果、何时结算、影响哪个既有实体、留下什么事件键**。
- 首版后果优先使用现有 silo/profession/resource/fragment/schedule/gate 效果；需要关系或 faction memory 时增加封闭、可校验效果，不在脚本里自由改对象。
- `Stance` 是派系创建时固定的身份字段，不因一次委托被随意重写；任务记忆应追加，不应篡改派系身份。
- 任务结果通过真实计量、派系状态、已解锁行动和后续事件影响既有结局仲裁，不设置“完成某势力任务=直接进入某结局”的捷径。
- 固定时间线场景的具名人物生死只服从权威链，第三方委托不能覆盖。
- 拒绝和放弃是有效分支，不应自动生成更坏结局；后果必须来自可解释的实体反应。

## 6. 信息呈现与文案规则

### 6.1 行动行

```text
Community Treatment                         15 AP
基础职务 · 巡诊与人群结果
```

```text
Quarantine                                  20 AP
已解锁，暂不可用 · 需要完成首次巡诊
来源：Medical 职务履历
```

### 6.2 行动结果

```text
巡诊完成：目标恐慌下降，生产力恢复。

顺带归档一份材料
《不同版本的配给通知》
Medical · Lower 巡诊 · 患者叙述
这份材料支持“通知内容不一致”，尚不能证明责任来源。
```

行动列表不显示“情报概率”或材料掉落预告；没有材料时不显示第二块，也不写“本次没有情报”，避免把职务行动重新包装成情报判定。

### 6.3 解锁提示

```text
新职务行动解锁：Direct Order
来源：完成首次公开讲话
当前可用；AP 与目标仍在执行前校验。
```

```text
已解锁，暂不可用：Incite Rebellion
当前缺少：Mayor 关系 0.25
不是剧情未解锁，不会再次隐藏。
```

## 7. 实施切片与验收

### 7.1 推荐切片

1. **门控拆分**：每职业一项基础职务常驻，进阶职务按 ID 配置；Build/Share 常驻；保持其他三态规则与服务端校验。
2. **Observation hook 垂直切片**：先完成 Medical、Supply、Mechanical 三个部门，证明治疗、库存、维修三种渠道不同。
3. **移除专用信息入口**：删除 Gather 与两个 Intel operation，改造四个直接取碎片职业行动；把 `first_signal` 切到首份 Observation。
4. **覆盖其余六部门**：每职业基础/进阶与合法材料池齐全，跑九职业开局推进测试。
5. **首个第三方接触**：只做 Supply 一次性委托，使用固定选择、事件键、门控和 guarded expiry，不先建 quest 系统。
6. **扩展既有 faction 接触**：在真实代表、已知状态和消失处理齐备后，再接 faction memory 等增量效果。

### 7.2 验收标准

- 开局不存在 Gather Info 或任何唯一主奖励为信息的按钮；九职业都能通过基础职务到达首份 Observation 和 `first_signal`。
- 常驻行动灰置时显示实时原因；已解锁行动条件回落仍保留；LOCKED 不进入可执行列表。
- 没有材料时职务主效果仍完整；主行动失败没有材料；重复材料不重复计数、不重复推进、不额外消耗 RNG。
- 同一 Fact 的 Medical/Supply/Mechanical 材料在渠道、直接观察、版本和盲点上可区分，不能只是替换部门名称。
- 缺权威楼层时材料只标 zone；没有死者/失联者继续提供新材料的入口。
- 首个 Supply 委托能接受、拒绝、暂缓、完成和超时；各分支有不同事件键，门控不能绕过，完成后续不会重复触发。
- 接触方不存在或失效时任务显式结束；未确认派系不暴露完整身份；没有新增专用反抗组织。
- 任务选择通过真实事件、计量和门控影响后续，不直接指定结局，不改固定时间线人物生死。
- 新 Observation hook 建立受控 sim 新基线；同种子双跑一致，LLM off/on 不影响 gameplay 轨迹。

## 8. 明确不做

- 不把 Gather Info 改名后继续保留同样的随机 fragment 主奖励。
- 不让所有职业共享一个“调查”按钮或全局未知 ID 池。
- 不用纯随机掉落替代职务来源，不让玩家靠失败或空行动刷材料。
- 不复制 HOI4 国策树、PP，也不把灰置等同于未解锁。
- 不复制 Beholder 的具体人物、组织、物件链或固定剧情前置。
- 不新造专用反抗组织，不让流散后代成为可识别、可招募的任务网络。
- 不在首版建设自由对话、通用 quest 实例或复杂任务编辑器。
- 不用任务选择直接改写派系固定 Stance、正典人物生死或结局类型。

## Sources

- [cite:1] [HOI4 Official Wiki — Political decisions](https://hoi4.paradoxwikis.com/Political_decisions)
- [cite:2] [HOI4 Official Wiki — Generic decisions](https://hoi4.paradoxwikis.com/Generic_decisions)
- [cite:3] [HOI4 Official Wiki — Turkish national focus tree](https://hoi4.paradoxwikis.com/Turkish_national_focus_tree)
- [cite:4] [HOI4 Official Wiki — Turkish decisions](https://hoi4.paradoxwikis.com/Turkish_decisions)
- [cite:5] [HOI4 Official Wiki — Decision modding](https://hoi4.paradoxwikis.com/Decision_modding)
- [cite:6] [HOI4 Official Wiki — National focus modding](https://hoi4.paradoxwikis.com/National_focus_modding)
- [cite:7] [Alawar — Beholder official game introduction](https://alawar.com/games/beholder)
- [cite:8] [Prima Games — Beholder game overview](https://primagames.com/eguides/mobile-gaming-eguide-tips-tricks-and-strategies-for-todays-hottest-mobile-games/beholder/game-overview)
- [cite:9] [Beholder Wiki — 搜查、记录与处置](https://wikiwiki.jp/beholder/%E7%BD%AA%E3%81%A8%E7%BD%B0)
- [cite:10] [Showgamer — Beholder walkthrough](https://showgamer.com/en/proxozhdenie-slozhnyx-i-neprostyx-situacij-v-igre-beholder)
- [cite:11] [PlayStationTrophies — Beholder complete walkthrough](https://www.playstationtrophies.org/forum/topic/301117-beholder-complete-edition-100-platinum-trophy-walkthrough/)
- [cite:12] [Beholder Wiki — 首个箱子委托](https://beholder.fandom.com/wiki/Mystery_box)
- [cite:13] [Beholder Wiki — 后续人物请求](https://beholder.fandom.com/wiki/Danton%27s_request)
- [cite:14] [PlayStationTrophies — 第三方运动结局条件](https://www.playstationtrophies.org/game/beholder-complete-edition/trophy/206448-brave-new-world.html)

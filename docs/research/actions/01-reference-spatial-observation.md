# 参考案例：空间、物件与观察链

> 范围：Beholder、This War of Mine、Do Not Feed the Monkeys、Shadows of Doubt。只讨论玩家动作如何出现，不复制人物、组织或剧情。
>
> 硬判据：**动作参数必须参与结算，不能只参与文案。**本文判断一个案例是否可借，也以其对象、地点、时机和输入是否改变实际结果为准。

## 1. Beholder：借“人—房—物—证据—档案”，不借走楼

### 实际做法【参考事实】

玩家先点具体房门，再选择敲门、窥视或开锁；进入房间后，在烟感器上安装摄像头，在家具中搜查或拿取物品。官方玩法强调趁租客不在时行动。看到违法行为或特殊物品后，还要把相应说明记入住客档案；写 Profile/Report 时，仍需填写租客、房号、法令并放入证据。深谈开启委托，帮助、交付、隐瞒和举报改变后续分支。[cite:1][cite:2][cite:3]

这里存在 Search、Report、Profile、说服等抽象词，但它们都不是全局动作：

| 表面动词 | 真正参与结算的参数 |
| --- | --- |
| 搜查 | 哪个房间、哪件家具、租客是否在场、发现什么物件 |
| 记录 | 哪个租客、哪条物件说明或目击行为 |
| Report | 哪个租客、哪条法令、哪份证据、报告是否属实 |
| 说服 | 当前对话、此前互动、可用声望、对方诉求 |

### 可借鉴层

1. **对象模型**：动作入口是租客、房门、设施和物件，不是“情报”分类。
2. **材料用途**：普通物件可说明生活习惯，违法物件可进入处置；信息价值不只是一种。
3. **二级确认**：发现材料与正式入档分开，避免玩家捡起物品就自动理解全部意义。
4. **关系记忆**：帮过谁、交付过什么、举报过谁，比抽象好感度更能解释后续变化。

### 不借的部分

- 不把主角移动、偷窃和潜入当作本作必须新增的玩法。
- 不把每次职务行动都设计成“找违禁品”，否则 Observation 会退化成另一套收集清单。
- 不复制办公桌表单的全部操作步骤；本作保留“选择材料与用途”，去掉无结算价值的拖拽。

### 对本作的映射

```text
点 Medical 病区 → 巡诊
点异常申领单 → 查看来源与关联事件
点 Supply 联系人 → 出示申领单 / 询问物流 / 归还单据
```

这些动作必须使用病区、联系人、单据的真实 ID。若换成另一病区或另一联系人结果完全一样，就没有借到这一层。

## 2. This War of Mine：借“执行者—设施—工具—时段”，不借生存经营

### 实际做法【参考事实】

玩家先选幸存者，再点柜子、瓦砾、门或设施上的情境图标。工具决定能否开锁或清障，动作耗时；白天与夜间提供不同活动，夜间派遣还受地点、背包、技能、健康和危险影响。Trade/Craft 虽属抽象类别，仍须选择执行者、商人或设施、具体物品和材料。[cite:4][cite:5][cite:6][cite:7]

### 可借鉴层

1. **执行者差异**：同一设施对不同身份、健康和工具状态的人提供不同结果。
2. **设施差异**：工作台、门、瓦砾不是统一的“使用”按钮，而是各自决定动作和消耗。
3. **时段差异**：时段不是视觉滤镜，而是改变可去地点、可执行动作和风险。
4. **具体交换**：交易必须落到“谁拿什么换什么”，不是点击后增加通用关系。

### 不借的部分

- 不引入多人小队、背包经营或昼夜生存循环。
- 不把耗时本身当作深度；工具不足应显示具体原因，并提供其他工作路径。
- 不让每个动作都变成采集资源，避免职务结果只剩经营数值。

### 对本作的映射

Medical 巡诊与 Mechanical 检修可以共用“设施作业”模板，但输入和结果必须不同：

| 参数 | Medical | Mechanical |
| --- | --- | --- |
| 设施 | 病区/诊疗点 | 泵房/管线/机械层 |
| 工具与权限 | 药品、诊疗身份、患者准入 | 工具、维修权限、停机窗口 |
| 主结果 | 患者与人群状态 | 设备、稳定与资源状态 |
| 伴随材料 | 症状、病史、患者叙述 | 维修单、改动痕迹、设备异常 |

若二者最终都只增加同一部门的 connection，而没有上述差异，就是公式化按钮。

## 3. Do Not Feed the Monkeys：借“机位—作息—关键词—干预”，不复制监视墙

### 实际做法【参考事实】

玩家通过已购“笼子”的固定机位观察特定房间，人物按作息出现。点击对话或物件得到黄色关键词，在笔记中整理、组合后上网检索，再利用查得的联系方式打电话、寄送物品，或回答针对某个笼子的调查问题。买笼、租金、饮食、健康、睡眠和兼职共同消耗金钱与时间。俱乐部禁止接触对象是规则而非绝对技术锁，玩家仍可干预，但需承担后果。[cite:8][cite:9][cite:10]

检索框、拨号、寄件和观察报告是抽象接口，但其前置不同：

```text
具体机位 + 已出现的人物/物件
→ 已知关键词
→ 可检索词组
→ 得到具体联系方式或地址
→ 选择通话、寄件或报告
```

玩家不能只因记得攻略就跳过当前存档的关键词门槛。[cite:9]

### 可借鉴层

1. **信息逐步变形**：原话→关键词→检索结果→可行动联系人，而不是一次生成完整答案。
2. **时间窗口**：人物作息决定能听到什么，错过窗口会改变可用信息。
3. **对象化解锁**：电话号码、地址和姓名解锁的是特定动作，不是全局“情报+1”。
4. **规则风险**：禁止接触可以成为有后果的违规选择，不一定只做硬禁用。

### 不借的部分

- 不复制多屏监控墙、买笼经济或生存消耗。
- 不让检索成为自由文本猜词游戏；本作用已发现对象生成可选词组。
- 不让关键进度依赖唯一且不可恢复的短暂窗口。

### 对本作的映射

IT 的链路维护可以得到某个终端的访问日志，日志先产生“设备编号/时段”材料；玩家再将设备编号与维修单组合，找到对应设施或联系人。后续动作只能是“询问该终端的值班人”“核查该时段的维修记录”，不能升级成全局 Build Network。

## 4. Shadows of Doubt：借“证据参数与多条侦查路径”，不复制全城

### 实际做法【参考事实】

市民有住址、工作和日程；CCTV 只有现场装有设备时可用，收据要按地点、时间对照录像。指纹、通话记录、邮件和证件须到场检查，再由玩家钉板、连线，并用具体嫌疑人和证据填写结案表。可进入不等于获准：门锁、安保、私闯处罚及屋主意愿构成限制；金钱可用于装备和贿赂，社会信用影响入户机会。[cite:11][cite:12][cite:13]

通用提问菜单、钉板和结案表并未消失，但答案仍取决于具体 NPC、证据、权限和玩家已经掌握的材料。

### 可借鉴层

1. **证据带参数**：收据的商户与时间、录像的地点与时段、指纹的样本来源共同决定能否互证。
2. **替代路径**：可用询问、现场物件、记录、权限或贿赂达到同一目标，避免单一唯一解。
3. **准入与发现分离**：玩家能进入某处，不等于有权搜查，更不等于能正确解释物件。
4. **结论需要对象**：提交结论时必须选择具体主体和证据，不能按“完成调查”自动结案。

### 不借的部分

- 不生成全城级 NPC 日程、自由钉板或复杂 3D 现场。
- 不把贿赂变成所有关系的通用替代路径。
- 不让证据数量自动等于结论正确；来源独立性和适用范围仍需校验。

### 对本作的映射

异常申领单与维修单可以指向同一 Fact，但只有满足“时间相邻、设施一致、物流批次相同”等明确条件才能形成组合证据。两份材料即使主题相同，参数不合也不能组合。

## 5. 四个案例的共同结论

1. 动作首先属于一个对象，不属于一张全局动作表。
2. 通用动词可以存在，但对象、输入和权限必须改变结果。
3. 信息获得是多步行为链，关系变化是具体行为的后果。
4. 时间应改变对象与材料，而不只是增加 AP 消耗。
5. 正确借鉴是缩小对象模型，而不是把全城、整栋楼或监控墙完整搬进本作。

## 6. 本文反例

```text
按钮 A：帮 Supply 核对配给
按钮 B：帮 Mechanical 修泵

若二者只更换 label，最后都执行：
applyGlobalEffect("connection", +X)
```

且部门、设施、物品、职务身份、材料池和后续记忆均不参与结算，那么它们仍是同一个公式化按钮。共享底层 mutation 并不是问题；问题是上游不识别对象、下游没有不同结果。

## Sources

- [cite:1] [Beholder 官方 Press Kit](https://beholder-game.com/en/presskit)
- [cite:2] [Prima Games — Beholder: What You Need to Know](https://primagames.com/eguides/mobile-gaming-eguide-tips-tricks-and-strategies-for-todays-hottest-mobile-games/beholder/what-you-need-to-know)
- [cite:3] [Invision Community — Beholder review](https://invisioncommunity.co.uk/beholder-review/)
- [cite:4] [This War of Mine — Steam](https://store.steampowered.com/app/282070/This_War_of_Mine/)
- [cite:5] [GamePressure — This War of Mine: Essence of the Game](https://www.gamepressure.com/thiswarofmine/essence-of-the-game/z46c9b)
- [cite:6] [GamePressure — This War of Mine: Night Assignments](https://www.gamepressure.com/thiswarofmine/basic-information/zf6d3b)
- [cite:7] [GamePressure — This War of Mine: Trading](https://www.gamepressure.com/thiswarofmine/trading/z16d43)
- [cite:8] [Do Not Feed the Monkeys — Steam](https://store.steampowered.com/app/658850/Do_Not_Feed_the_Monkeys/)
- [cite:9] [Rapid Reviews UK — Do Not Feed the Monkeys](https://www.rapidreviewsuk.com/do-not-feed-the-monkeys-review/)
- [cite:10] [Gamereactor — Do Not Feed the Monkeys](https://www.gamereactor.eu/do-not-feed-the-monkeys-review/)
- [cite:11] [Shadows of Doubt — Steam](https://store.steampowered.com/app/986130/Shadows_of_Doubt/)
- [cite:12] [PlayStation Blog — Shadows of Doubt gameplay](https://blog.playstation.com/2024/08/12/shadows-of-doubt-hits-ps5-on-september-26-new-private-detective-gameplay-revealed/)
- [cite:13] [Rock Paper Shotgun — Shadows of Doubt review](https://www.rockpapershotgun.com/shadows-of-doubt-review)

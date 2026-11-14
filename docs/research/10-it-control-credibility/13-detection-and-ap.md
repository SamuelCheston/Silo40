# IT 幕后权力：威势、Protocol 能动性与 Judicial 三方附着 · 检测、AP 与其他机制自洽

> 拆分自 [IT 幕后权力：威势、Protocol 能动性与 Judicial 三方附着](../10-it-control-credibility.md)；目录、章节映射与出处见该索引。

## 8. 检测、AP 与其他机制自洽

### 8.1 行为—部门规范冲突与 AP

令 `a∈[-1,1]` 为行为偏离部门规范的方向和强度，`d∈[-1,1]` 为部门规范倾向：

```text
mismatch = |a|·max(0,-sign(a)·d)
AP_cost  = ceil(base_AP·(1+0.5·mismatch))
```

AP 表示组织行动、找渠道和承受社交阻力的成本，只看行动本身与规范冲突；目击、记录和身份监控不重复塞进 AP。

### 8.2 三类检测流程

| 类别 | 判定 | 后果 |
| --- | --- | --- |
| `UNSAMPLED` | 身份×位置不在监控采样框；矿工普通行为默认属于此类 | **不进入检测流程**：无采样 RNG、无普通怀疑增量、无“很小系数”。AP 仍独立计算 |
| `SAMPLED` | 在监控范围内的普通身份与行为 | 使用 §8.3 数值公式 |
| `DIRECT_PROTOCOL` | 行为触及 Protocol 禁止/必需/目标，IT 的相关行为优先进入此类 | 不生成普通怀疑惩罚，直接按 §6.2 计算 `R_s/W/J` |

矿工若主动制造正式 Judicial 案件或触及 Protocol，不是把系数调大，而是新建相应案件/Protocol 路径；普通监控采样仍保持豁免。

### 8.3 `SAMPLED` 的总怀疑公式

```text
Δsuspicion = base_action·V_identity·F_fit·W_witness·R_record·(1+mismatch)
```

| 因子 | 范围 | 依据 |
| --- | --- | --- |
| `base_action` | 0.01–0.05 | 行为本身的敏感性，不含身份和传播 |
| `V_identity` | 0.5–2.0 | 执行者在该部门/楼层的可见度；Mayor/上层精英高，普通中层低 |
| `F_fit` | 0.6–2.0 | 行为与身份职责的符合度；利用职务正常办事低，身份自相矛盾高 |
| `W_witness` | 私下 0.35 / 小圈子 1.0 / 部门 1.6 / 公开 2.2 | 同时在场目击者与二次传播半径 |
| `R_record` | 无记录 0.7 / 匿名日志 1.0 / 个人档案 1.35 / 监控或告示 1.6 | 可复核记录延长可检测性 |
| `1+mismatch` | 1–2 | 排外规范越强，逆行越显眼；极端时翻倍 |

这些因子只乘普通怀疑，不乘 AP，也不替代 `DIRECT_PROTOCOL`。IT 触及 Protocol 时不是“怀疑度很大”，而是直接发生偏离后果。

### 8.4 信息、门控、结局与确定性

- 玩家 `B` 只决定行动和输出证据强度；`E/Q` 只读相应受众层，无重复曝光。
- LOCKED 行动不能产生观察、Protocol、维护或附着效果；目标校验同时拒绝公开亮相与上层控制方。
- 现有玩家 IT 行动统一加 Safeguard 风险与新三类偏离存在实现冲突；落地时必须改为条款判定，不能把新维护动作继续挂到统一正惩罚上。
- `W/K/A` 不成为第五类结局主体，只经实际状态进入 [weighted_endings.go](../../../internal/service/weighted_endings.go)。
- 当前仲裁没有法治候选，需新增 `RULE_OF_LAW`：`covenant_locked=true` 只是该候选的资格条件，不是自动胜利；最终仍由玩家、驻地派系、具名人物和上层控制方 加权选择，Safeguard 等硬失败优先。
- 观察、条款、事实、动作和附着按稳定 ID 排序；获取与暴露判定使用既有单 RNG 流，其余公式不消耗 RNG。
- 存档包含 `B/π`、Protocol believed/true 版本、偏离与 review、`RelicItem/H_channel/reflection_key/dialogue_id/utterance_id`、`G/J/E/Q/W/K/R_op/R_s/A/regime/mayor_id/covenant_locked/计数器/配置版本`。


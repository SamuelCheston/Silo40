# 结局事件链 · `EXCLUSIONIST`《深闭》

> [返回总索引](../silo40-endings.md) · [机制底座](./10-chain-foundation.md) · [剧情台阶](./03-exclusionist.md)

链条形态：**默认收敛**。《深闭》没有独立的"推进"节点群——它是其他线全部未成立时的落点。本线事件链只做两件事：给惯性滑入以可读节点，给死后复制以明确分岔。

## 1. 节点表

| 节点 | 分组 | 触发 | 分岔与含义 | 效果（封闭集） | 后续 |
| --- | --- | --- | --- | --- | --- |
| `normalcy_hardens` | special，REPEATABLE cd12 | `all(timestampAtOrAfter(125,1), professionIdeologyGteCount("loyalty", 0.6, 4))` | 惯性滑入的可读节点：封闭忠诚取得公共主导；每次点火说明"又一年没有改变" | `history_burden −0.02`、`rebellion −0.02`（sim 校准起点；历史正在被常态化） | 计数即叙事 |
| `post_fall_takeover` | special，ONCE | `all(siloFlagTrue("silo1_destroyed"), professionIdeologyGteCount("loyalty", 0.6, 4))` | 死后复制分岔：覆灭事实抵达时封闭仍主导，本地权力接管封闭逻辑 | `history_burden +0.03`、`panic_value +0.03`（sim 校准起点） | 终局收束 |

## 2. 分支与改道（本线是其他线的影子）

| 上游事实 | 汇入本线的路径 |
| --- | --- |
| 《共证》宿主暴露后未被处置，驻地继续封闭 | 传播链材料被归档为怪人遗物；`normalcy_hardens` 继续计数 |
| 《无人裁决》思潮闸失败（纯堡主导） | `post_fall_takeover` 直接承接；若 `mayor_public_stand` 已成立则改判《人治》 |
| 《公议》《人治》双双资格失败 | 按实际权力状态落入本线 |
| 奉外式外部崇拜破灭 | 反弹使 `loyalty` 广度上升，加速 `normalcy_hardens` |

## 3. 超时与失败结算

- 本线没有超时：它就是"时间流逝本身"的结局。`normalcy_hardens` 的 cd12 是叙事节拍，不是进度条。
- 爬出窗口：在 `post_fall_takeover` 之前，任何一条其他线的节点群重新激活（新的分享、新的接触卡、新的路线调查）都算爬出；判定不读本线节点，读实际成立的事实。

## 4. 与结局判定的对接

- `EXCLUSIONIST` 已在仲裁候选（现状四个历史候选之一），无 G1 缺口。
- 资格读法：封闭派系立场与忠诚思潮广度经派系贡献进入加权；`profession_ideology_gte_count` 读的是部门派生视图，可放心用作条件（作者手册 §8）。
- 同分顺序中 `EXCLUSIONIST` 垫底：只要任何其他候选合格，本线就让位——这与"默认落点"的叙事身份一致。

## 5. 能力缺口

G8（终局画面展示：两个变体需要结局文案分支）。其余无缺口——本线是十条线中机制负担最轻的一条。

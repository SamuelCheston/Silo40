defineEvent({
  id: "negotiation_open_channel",
  title: "谈判：搭上市长办公室的秘密线路",
  description: "中间人换了三条线路、两个假班次，才把第一句话递进市长办公室。对面没有挂断——这就是开始。谈判不是一次通牒：你们可以一直通气、反复试探；市长可能一直拖延，也可能在最后一刻妥协。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 谈判层·开信道（21 设计卷 §5）：建立市长与叛乱方之间的秘密信道。
  // 开信道时一次性抽取市长性情（权重输入的性格/特质归因），此后月度
  // 自动低强度通气；信道崩塌不可重开（失败是合法结果）。
  player_action: {
    id: "NEGOTIATION_OPEN_CHANNEL",
    label: "建立谈判信道",
    description: "通过中间人与市长办公室建立秘密线路，开始谈判。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("NEGOTIATION_OPEN_CHANNEL"),
  effects: [
    negotiationOpen(),
  ],
});

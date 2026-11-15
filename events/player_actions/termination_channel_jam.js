defineEvent({
  id: "termination_channel_jam",
  title: "值夜里的一次检修",
  description: "你借着一次例行的夜间检修留在配线间：把一段本该空置的接线柱重新压上，又在交接簿上补了一行谁也看不懂的编号。没有人被说服，也没有人被告发——只是当那条远程通道再次点亮时，本地这一侧的应答会晚半拍，然后散成杂讯。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 远程终止渠道的反制 (《寂堡》链 21、20 分册 §4.2/§5)：藏在日常小动作里，
  // 有对象 (配线架)、地点 (值夜时的配线间)、时机 (终止自检之后)、材料
  // (备用接线柱与交接簿) 与痕迹 (历史包袱 +0.03)。共享情报不是反制，
  // 所以本动作与 Share Info 无关；它只对通信硬件动手。
  //
  // 触发条件由 action_gates.json 的 TERMINATION_CHANNEL_JAM 负责 (自检已点亮，
  // 且与 Mechanical 或 IT 有最低信任)。效果：一个月后信道干扰成立。
  player_action: {
    id: "TERMINATION_CHANNEL_JAM",
    label: "值夜时对配线架做手脚",
    description: "借例行检修在配线间留下一个不该有的接点：终止信道的应答会被杂讯吃掉。只免疫这一条远程渠道，挡不住物理手段。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("TERMINATION_CHANNEL_JAM"),
  effects: [
    scheduleEvent("termination_channel_disrupted", 1), // TODO(balance)
  ],
});

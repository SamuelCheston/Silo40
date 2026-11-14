defineEvent({
  id: "escape_window_orderly",
  title: "有序占用撤离通道",
  description: "轮班表贴出来的那一刻，撤离就不再是一场逃亡，而是一次搬迁：谁维护出口、谁按批次配给、谁照顾走不快的人、谁在最后一批关闭设备。旧权力仍然在骂这是叛乱，但楼层是稳的，通道是通的。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 抉择 R-D（02-rebellion §R-4）：有序占用＝低结构损耗、撤离秩序系数高。
  // 本键供出发结算读撤离秩序；与全面暴乱互斥呈现（门控无否定谓词，共用缺口）。
  player_action: {
    id: "ESCAPE_WINDOW_ORDERLY",
    label: "有序占用撤离通道",
    description: "以公开停工与有序占用打开窗口，保留基础设施与撤离秩序。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 1, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_WINDOW_ORDERLY"),
  effects: [
    siloMetricDelta("rebellion", 0.02),  // TODO(balance)
    siloMetricDelta("history_burden", 0.02), // TODO(balance)
  ],
});

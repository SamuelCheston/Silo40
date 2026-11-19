defineEvent({
  id: "mayor_bypass_drift",
  title: "无人承接的秩序",
  description: "你没有把那个位置给他。六个月过去，驻地确实没有一个能公开承接秩序的本地人——材料留在你手里，秩序也留在暗处。会上没有人再提那件事，走廊里的旧话题有了固定的官方版本。没有谁在镇压；只是从你绕过他的那一天起，常态就开始自己硬化。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 延迟回访（绕过键 +6 月，TODO(balance)）：世界明确引用这次绕过。
  // 纯秘密路线的漂移代价——历史包袱与叛乱边缘被常态磨钝，驻地滑向
  // 排斥主义惯性（与 normalcy_hardens 同向，但这一笔归因于选择本身）。
  trigger: all(
    eventTriggered("player_action", "MAYOR_ALLY_BYPASS"),
    monthsSinceEventGte("player_action", "MAYOR_ALLY_BYPASS", 6)
  ),
  effects: [
    siloMetricDelta("history_burden", -0.02), // initial calibration, TODO(balance)
    siloMetricDelta("rebellion", -0.01),      // initial calibration, TODO(balance)
  ],
});
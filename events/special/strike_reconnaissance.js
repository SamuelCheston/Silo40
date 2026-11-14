defineEvent({
  id: "strike_reconnaissance",
  title: "外部投送侦察",
  description: "外围哨位几次报告高空有不明物体缓慢掠过，不发光、不应答，像在反复确认坐标。没人知道那是什么，也没人敢把猜测说出口。只有读过旧档案的人明白：当投送开始反复确认，留给地面的时间就不多了。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Remote-physical-strike channel omen (《寂堡》链 21)：通牒与终止信道之后局势
  // 仍不收敛，上层控制方开始为物理投送做侦察校位。确定性触发，无随机。
  trigger: all(
    eventTriggered("history", "silo1_ultimatum"),
    timestampAtOrAfter(128, 1),
    siloMetricGte("rebellion", 0.8)
  ),
  effects: [
    professionMetricDeltaAll("panic_value", 0.06),
    siloMetricDelta("rebellion", 0.02),
    scheduleEvent("bombing_terminal", 3),
  ],
});

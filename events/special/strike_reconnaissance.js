defineEvent({
  id: "strike_reconnaissance",
  title: "外部投送侦察",
  description: "外围哨位几次报告高空有不明物体缓慢掠过，不发光、不应答，像在反复确认坐标。没人知道那是什么，也没人敢把猜测说出口。只有读过旧档案的人明白：当投送开始反复确认，留给地面的时间就不多了。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Remote-physical-strike channel omen (《寂堡》链 21)：终止信道自检之后局势仍
  // 不收敛，上层控制方开始为物理投送做侦察校位。确定性触发，无随机。阶梯串联
  // 在终止自检之后 (＋6 月，TODO(balance))：压力在任一级回落都会停住阶梯，
  // 每一级都重新检查复合门槛。
  // 与终止信道同口径：物理投送只对仍在运转的驻地开启 (Energy ≥ 1)。
  trigger: all(
    eventTriggered("special", "termination_channel_test"),
    monthsSinceEventGte("special", "termination_channel_test", 6),
    siloMetricGte("order_pressure", 0.45),
    actorMetricGte("propaganda_level", 2),
    { type: "silo_resource_gte", metric: "Energy", value: 1 }
  ),
  effects: [
    professionMetricDeltaAll("panic_value", 0.06),
    siloMetricDelta("rebellion", 0.02),
    scheduleEvent("bombing_terminal", 3),
  ],
});

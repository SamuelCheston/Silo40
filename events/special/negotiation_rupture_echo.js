defineEvent({
  id: "negotiation_rupture_echo",
  title: "断线的回声",
  description: "信道断了三个月，中间人没有再回来。谈判结束了，猜忌没有结束：被堵死的出路在走廊里发酵，封门的声音还在耳朵里，'既然谈不成'的念头在各部门之间传。断线不是句号——它留下的恐慌与不合作，还在继续收利息。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 延迟回访（信道破裂键 +3 月，TODO(balance)）：镇压打断或 IT 曝光都落
  // 同一个破裂键；三个月后回声一次性计账：全部门恐慌 +0.02、叛乱 +0.02。
  // 破裂不是死胡同：世界明确引用这次断线，恐慌与叛乱进入后续权重。
  trigger: all(
    eventTriggered("negotiation", "channel_rupture"),
    monthsSinceEventGte("negotiation", "channel_rupture", 3)
  ),
  effects: [
    professionMetricDeltaAll("panic_value", 0.02), // initial calibration, TODO(balance)
    siloMetricDelta("rebellion", 0.02), // initial calibration, TODO(balance)
  ],
});

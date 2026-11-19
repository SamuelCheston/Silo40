defineEvent({
  id: "route_second_window",
  title: "第二扇窗",
  description: "他又来了，图纸比上次更旧，带来的消息比上次更坏。'我说过我还在这儿。'他把图纸摊开——这段时间，想走的人仍在暗处互相打量，每一次打量都是一次暴露，账本上已经记了一笔。窗口还开着，但比上次窄了：重新丈量，得挑更险的时辰。",
  type: "TECHNICAL",
  fire_mode: "ONCE",
  // 延迟回访（拒绝键 +8 月 / 暂缓键 +4 月，TODO(balance)）：兑现拒绝文案
  // "我还在这儿"与暂缓文案"时间也是成本"。拖延本身付出真实代价——
  // prep_exposure 记入准备账本，出发时进追踪基数；点亮同时经 action_gates
  // 解锁 ROUTE_SURVEY_REACCEPT（换条件再来一次，AP 更贵），不是死胡同。
  trigger: any(
    all(
      eventTriggered("player_action", "ROUTE_SURVEY_DECLINE"),
      monthsSinceEventGte("player_action", "ROUTE_SURVEY_DECLINE", 8)
    ),
    all(
      eventTriggered("player_action", "ROUTE_SURVEY_DEFER"),
      monthsSinceEventGte("player_action", "ROUTE_SURVEY_DEFER", 4)
    )
  ),
  effects: [
    preparationDelta("prep_exposure", 0.03), // initial calibration, TODO(balance)
  ],
});
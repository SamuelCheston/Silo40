defineEvent({
  id: "route_confirmed",
  title: "路线成形",
  description: "丈量的结果摊在一张手绘图上：哪一段能走人、哪一段只能走货、哪一段永远不能停留。这还不是'可以离开'的证明——先遣还没有出发——但驻地第一次拥有了一条被实测过的路。",
  type: "TECHNICAL",
  fire_mode: "ONCE",
  // 先遣事实的前置（13 分册 §1）：勘察步骤完成后路线事实成立。
  trigger: eventTriggered("special", "route_survey_step"),
  effects: [
    siloMetricDelta("rebellion", 0.04), // TODO(balance)
  ],
});

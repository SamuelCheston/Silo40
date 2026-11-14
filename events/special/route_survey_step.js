defineEvent({
  id: "route_survey_step",
  title: "勘察：丈量大山的心脏",
  description: "借着机械与矿业职务的正当理由，你们把检修出口一段一段地丈量：闸门能不能重复开启、传感在哪几米失效、哪条夹层能避开主通道。每丈一段，'外面'就从一个词变成一段路。",
  type: "TECHNICAL",
  fire_mode: "ONCE",
  // 路线勘察步骤（13 分册 §1）：接受键之后，由机械/矿业职务动作承载实地
  // 勘察。订阅既有职务动作的节点放 special/（不拦截宿主动作）。
  trigger: all(
    eventTriggered("player_actions", "route_survey_accept"),
    any(
      playerActionIs("MECHANICAL_OVERHAUL"),
      playerActionIs("MECHANICAL_PIPE_TAP"),
      playerActionIs("MINES_DEEP_EXCAVATION"),
      playerActionIs("MINES_TUNNEL_NETWORK"),
    ),
  ),
  effects: [
    // 接应档位：路线已勘察（16 §2.1）。先遣确认前只到 surveyed。
    preparationStage("surveyed"),
    siloMetricDelta("rebellion", 0.02), // TODO(balance)
  ],
});

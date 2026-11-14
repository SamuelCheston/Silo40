defineEvent({
  id: "silo1_purge_omen",
  title: "本地整肃征兆",
  description: "IT 与医务开始不寻常地协同：深夜清点人数、调取名册、请领远超平常的镇静与麻醉类物资。问起来，答复都是例行的设备检修。老练的人看得出，这不是检修，是在为一场自上而下的整肃做准备。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Local-execution channel omen (本地执行链)：通牒之后叛乱仍居高，上层控制方
  // 转而指令堡内 IT/医务执行一场本地整肃（投药、清档、减口）。确定性触发。
  trigger: all(
    eventTriggered("history", "silo1_ultimatum"),
    timestampAtOrAfter(126, 9),
    siloMetricGte("rebellion", 0.8)
  ),
  effects: [
    professionMetricDeltaAll("panic_value", 0.04),
    siloMetricDelta("rebellion", 0.02),
    scheduleEvent("silo1_purge_execution", 3),
  ],
});

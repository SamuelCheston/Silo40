defineEvent({
  id: "silo1_purge_omen",
  title: "本地整肃征兆",
  description: "IT 与医务开始不寻常地协同：深夜清点人数、调取名册、请领远超平常的镇静与麻醉类物资。问起来，答复都是例行的设备检修。老练的人看得出，这不是检修，是在为一场自上而下的整肃做准备。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Local-execution channel omen (本地执行链)：带压力条件的通牒签发后压力仍
  // 不回落，上层控制方转而指令堡内 IT/医务执行一场本地整肃（投药、清档、
  // 减口）。确定性触发。时间门相对通牒 (＋4 月，TODO(balance))，门槛读叛乱
  // 与恐慌的复合 order_pressure (与通牒同阈值)——单读叛乱 ≥0.8 不可达。
  // 阶梯时点：整肃前兆 U+4 (执行 U+7)、终止自检 U+10 (投放 U+14)、投送侦察
  // U+16 (打击 U+19)——《断层》的快速维生崩塌 (~U+10) 仍有机会先于远程结算。
  trigger: all(
    eventTriggered("special", "silo1_ultimatum_cross"),
    monthsSinceEventGte("special", "silo1_ultimatum_cross", 4),
    siloMetricGte("order_pressure", 0.45),
    actorMetricGte("propaganda_level", 2)
  ),
  effects: [
    professionMetricDeltaAll("panic_value", 0.04),
    siloMetricDelta("rebellion", 0.02),
    scheduleEvent("silo1_purge_execution", 3),
  ],
});

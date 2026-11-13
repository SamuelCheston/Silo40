defineEvent({
  id: "neighbor_silo_silence",
  title: "邻堡静默",
  description: "旧日断联同盟中的一座邻堡彻底停止了应答。当年四十号地堡带头切断联络、随后在轰炸中毁灭，如今这张网络正在它派出的特工眼前逐格熄灭。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  trigger: timestampAtOrAfter(124, 1),
  effects: [
    siloMetricDelta("history_burden", 0.05),
    siloMetricDelta("rebellion", 0.03),
    professionMetricDeltaAll("panic_value", 0.04),
  ],
});

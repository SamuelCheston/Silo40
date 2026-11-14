defineEvent({
  id: "termination_channel_test",
  title: "终止链路自检",
  description: "IT 值夜记录里多了一段无法归档的链路自检：一条久未启用的远程投放通道被点亮，短暂握手后又归于沉寂。居民只当是又一次设备误报；只有极少数人意识到，那条通道的另一端连着活人。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Remote-termination channel omen (《寂堡》链 21)：通牒之后叛乱仍居高，上层
  // 控制方点亮终止信道做投放前自检。确定性触发，无随机。
  trigger: all(
    eventTriggered("history", "silo1_ultimatum"),
    timestampAtOrAfter(127, 1),
    siloMetricGte("rebellion", 0.8)
  ),
  effects: [
    professionMetricDeltaAll("panic_value", 0.05),
    siloMetricDelta("rebellion", 0.03),
    scheduleEvent("termination_release", 4),
  ],
});

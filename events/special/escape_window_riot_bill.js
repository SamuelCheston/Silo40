defineEvent({
  id: "escape_window_riot_bill",
  title: "账单的第二期",
  description: "火光熄了三个月，账没有熄。撞开的出口还开着，可每一层被撞松的结构还在继续掉渣：修一处，裂两处。夜里有人听见承重墙里的响声就再也睡不着——他们记得那晚楼层是怎么连成一片的，也记得代价是怎么一层一层摊下来的。账单的第二期到了，付账的还是留下的人。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 延迟回访（暴乱键 +3 月，TODO(balance)）：兑现 escape_window_riot 文案
  // 「账单分两次到期」的承诺——开窗当天付人口与资源，三个月后结构损耗
  // 继续到期（全楼层稳定再降），恐慌余波未平。
  trigger: all(
    eventTriggered("player_action", "ESCAPE_WINDOW_RIOT"),
    monthsSinceEventGte("player_action", "ESCAPE_WINDOW_RIOT", 3)
  ),
  effects: [
    floorStabilityDelta("", -0.04),                 // 全部楼层，TODO(balance)
    professionMetricDeltaAll("panic_value", 0.02),  // TODO(balance)
  ],
});

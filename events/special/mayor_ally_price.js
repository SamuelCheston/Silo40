defineEvent({
  id: "mayor_ally_price",
  title: "盟友的价码",
  description: "你扶他站台的那只手，四个月后还在被账单追着。你喂的材料变成了他的声望，他的声望变成了别的部门的不安：例会上绕过他的议程越来越多，走廊里'他凭什么'的声音越来越密。盟友不是免费的——你替他付的那份，记在部门之间的裂缝里。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 延迟回访（接受键 +4 月，TODO(balance)）：同盟的代价记在部门张力上。
  // 喂证心跳的合法性拉力是收益，这一笔是收益的影子。
  trigger: all(
    eventTriggered("player_action", "MAYOR_ALLY_ACCEPT"),
    monthsSinceEventGte("player_action", "MAYOR_ALLY_ACCEPT", 4)
  ),
  effects: [
    siloMetricDelta("dept_tension", 0.02), // initial calibration, TODO(balance)
  ],
});
defineEvent({
  id: "pre_fall_echo",
  title: "那一夜的余波",
  description: "三个月了。那一夜的选择没有变成过去：队列里的人还记得口号，质询席上的人还记得自己举过手，被砸开的仓库门还没有修好。秩序、质询、火光——无论那一夜选中了哪一个，它都在继续收利息。驻地没有忘记那一夜，正如那一夜没有放过驻地。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 延迟回访（三选一任一完成键 +3 月，TODO(balance)）：覆灭前夕的抉择
  // 不是快照投影完就结束——三个月后按所选分支回来算账：
  //   忠诚动员 → 队列的惯性继续凝成忠诚（cohort loyalty +0.05），叛乱再压一档；
  //   公开质询 → 为公共讨论承担的风险继续到期（history_burden +0.03）；
  //   全面暴乱 → 维存损耗的第二期（全楼层稳定 −0.05）。
  // 时序纪律（33-nodes §3）：T-1 兜底 Y7M10、结算 Y8M1——晚于 Y7M7 做出的
  // 选择，+3 月回声落在结算之后，按「覆灭后不留内容窗口」口径不会到期；
  // 这是时钟硬墙，不为此放宽延迟。
  trigger: any(
    all(
      eventTriggered("player_action", "PRE_FALL_LOYALTY_MOBILIZATION"),
      monthsSinceEventGte("player_action", "PRE_FALL_LOYALTY_MOBILIZATION", 3)
    ),
    all(
      eventTriggered("player_action", "PRE_FALL_OPEN_INQUIRY"),
      monthsSinceEventGte("player_action", "PRE_FALL_OPEN_INQUIRY", 3)
    ),
    all(
      eventTriggered("player_action", "PRE_FALL_TOTAL_RIOT"),
      monthsSinceEventGte("player_action", "PRE_FALL_TOTAL_RIOT", 3)
    )
  ),
  script: {
    apply: function(ctx) {
      if (ctx.states["player_action:PRE_FALL_LOYALTY_MOBILIZATION"] &&
          ctx.states["player_action:PRE_FALL_LOYALTY_MOBILIZATION"].triggered) {
        // 忠诚动员：口号停了，队列的惯性没停 (initial calibration, TODO(balance))。
        return {
          effects: [
            cohortIdeologyDelta("loyalty", 0.05),
            siloMetricDelta("rebellion", -0.02),
          ],
        };
      }
      if (ctx.states["player_action:PRE_FALL_OPEN_INQUIRY"] &&
          ctx.states["player_action:PRE_FALL_OPEN_INQUIRY"].triggered) {
        // 公开质询：为不完全受控的公共讨论继续担风险 (initial calibration, TODO(balance))。
        return {
          effects: [siloMetricDelta("history_burden", 0.03)],
        };
      }
      // 全面暴乱：维存损耗的第二期 (initial calibration, TODO(balance))。
      return {
        effects: [floorStabilityDelta("", -0.05)],
      };
    },
  },
});

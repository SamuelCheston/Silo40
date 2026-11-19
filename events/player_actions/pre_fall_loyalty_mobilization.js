defineEvent({
  id: "pre_fall_loyalty_mobilization",
  title: "以全面忠诚动员换取秩序",
  description: "你支持封闭派系：暂停全部外部讨论，全面动员忠诚，等待下一道命令。驻地换来了表面的平静——队列整齐，口号一致，讨论停止。覆灭前夕的思潮正在凝成纯种地堡的形状：当外面再也没有回音时，这里的人只会说'以后只能靠自己严守旧规'。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // T-2 loyalty mobilization (33-nodes §3, 34-spec §1): order bought with
  // total loyalty. The cohort loyalty write tilts the pre-fall snapshot
  // toward the pure-silo shape — TIME absolute disqualification is then a
  // snapshot fact, not a verdict of this key.
  // 延迟回声见 special/pre_fall_echo.js（完成键 +3 月：忠诚惯性继续凝聚）。
  player_action: {
    id: "PRE_FALL_LOYALTY_MOBILIZATION",
    label: "忠诚动员换秩序",
    description: "暂停外部讨论，全面动员忠诚；驻地平静，快照倾向纯种。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 0,            // initial calibration, TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("PRE_FALL_LOYALTY_MOBILIZATION"),
  effects: [
    siloMetricDelta("rebellion", -0.05),            // initial calibration, TODO(balance)
    professionMetricDeltaAll("panic_value", -0.04), // initial calibration, TODO(balance)
    cohortIdeologyDelta("loyalty", 0.10),           // all cohorts; initial calibration, TODO(balance)
  ],
});

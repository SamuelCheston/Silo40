defineEvent({
  id: "silo1_purge_plot_leak",
  title: "把清点的时间讲出去",
  description: "你没有动那两页纸。你只是把请领清单、名册调阅与深夜清点的时间对在一起，告诉了夜班走廊里几个信得过的人：'他们要在换班最静的时候动手。'消息没有署名，却比通知走得快。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 本地整肃的反制·信息侧 (33/34 分册、20 分册 §4.2/§5)：藏在日常小动作里，
  // 有对象 (请领清单与名册调阅的时间)、地点 (夜班交接的走廊)、时机 (整肃征兆
  // 之后、执行之前)、材料 (对得上的时间)、痕迹 (历史包袱 +0.02、司法侧开始
  // 留意是谁在打听)。它不动纸面——名册篡改是另一条动作、另一档后果。
  //
  // 触发由 action_gates.json 的 PURGE_PLOT_LEAK 负责 (整肃征兆已点亮，且与
  // 司法或市长侧有最低信任)。效果：一个月后整肃图谋被公开，执行节点据此把
  // 后果从"重档"降到"中档"——整肃仍会来，只是失去突然性。
  player_action: {
    id: "PURGE_PLOT_LEAK",
    label: "把清点的时间讲出去",
    description: "把请领、调阅与深夜清点的时间对上，讲给夜班的人听：整肃仍会来，但不会那么突然。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("PURGE_PLOT_LEAK"),
  effects: [
    scheduleEvent("silo1_purge_exposed", 1), // TODO(balance)
    siloMetricDelta("history_burden", 0.02), // TODO(balance)
    // 痕迹：司法侧会记住是谁在打听清点时间（月度冷却会把它带回去）。
    { type: "profession_metric_delta", profession: "Judicial", metric: "alertness", value: 0.05 },
  ],
});

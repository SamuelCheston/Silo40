defineEvent({
  id: "silo1_purge_countermeasure",
  title: "复核夜的两页名册",
  description: "你在医务与 IT 交接名册的空档里留了下来：把两页纸换了位置，又照着旧班表的笔迹补了两行。没有人被说服，也没有人被告发——只是下一次点名时，会有人被漏掉。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 本地整肃的反制 (33/34 分册、20 分册 §4.2/§5)：藏在日常小动作里，有对象
  // (名册与请领单)、地点 (医务与 IT 的交接间)、时机 (整肃征兆之后、执行之前)、
  // 材料 (旧班表笔迹)、痕迹 (历史包袱 +0.02)。共享情报不是反制——本动作只
  // 动纸面，不涉及信息流通。
  //
  // 触发由 action_gates.json 的 PURGE_ROSTER_TAMPER 负责 (整肃征兆已点亮，
  // 且与 IT 或 Medical 有最低信任)。效果：一个月后名册篡改生效；整肃执行
  // 节点据此把后果从"重档"降到"轻档"。
  player_action: {
    id: "PURGE_ROSTER_TAMPER",
    label: "在名册复核时换掉两页",
    description: "借交接空档改动名册与请领单：整肃仍会来，但会少带走一些人、少清掉几行记录。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("PURGE_ROSTER_TAMPER"),
  effects: [
    scheduleEvent("silo1_purge_softened", 1), // TODO(balance)
    siloMetricDelta("history_burden", 0.02),  // TODO(balance)
  ],
});

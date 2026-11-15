defineEvent({
  id: "escape_spillover_send_physical",
  title: "投递：实体交接",
  description: "最古老的信道开始移动：人或物，物理地穿过那段黑暗。整包都在，磨损很轻，也没有任何流量可供对岸分析——代价是时间，以及路上每一段无法撤回的暴露。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  // 外溢投递（16 §3.5 投递侧）：实体交接类。整包携带、6 月在途、
  // 轻微磨损；不产生网络流量，暴露成本最低。与逃出群体的顺带携带
  // 耦合（再传递任务）属后话，见 KNOWN_GAPS。
  player_action: {
    id: "ESCAPE_SPILLOVER_SEND_PHYSICAL",
    label: "经实体交接送出包裹",
    description: "以人或物的物理携带投递外溢包裹。最慢，也最不显眼。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_SPILLOVER_SEND_PHYSICAL"),
  effects: [
    spilloverSend("physical_handoff"),
  ],
});

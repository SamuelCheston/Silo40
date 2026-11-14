defineEvent({
  id: "open_discourse_milestone",
  title: "开放表达里程碑",
  description: "开放思潮第一次成为公共表达：机械间里有人讨论那段外部信号的可信度，配给队伍里有人在争一条旧规定是否还应绝对服从。不是所有人都相信外面安全——恰恰因为保留了不确定，这种开放才经得起下一次危机。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 6, // initial calibration, TODO(balance)
  // A-1 (33-nodes §3, 34-spec §1): the readable heartbeat of the long
  // openness build (choices T-A/T-B). Reads the department derived view —
  // the cohort truth-source projection; qualification still reads the
  // engine's pre-fall snapshot, never this node.
  trigger: professionIdeologyGteCount("pro_foreign", 0.55, 2), // initial calibration, TODO(balance)
  effects: [
    siloMetricDelta("history_burden", 0.02), // initial calibration, TODO(balance)
  ],
});

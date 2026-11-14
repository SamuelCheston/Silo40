defineEvent({
  id: "pre_fall_choice",
  title: "覆灭前夕：最后一次先听谁的",
  description: "最后的异常预警过后，驻地面对同一个问题：先听谁的。封闭派系要求暂停全部外部讨论，等待下一道命令；开放派系要求按原计划举行公开核验与质询。双方都真心相信自己在拯救驻地。这是覆灭之前最后一次'先听谁的'——快照会记住这一刻的选择。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // T-1 (33-nodes §3, 34-spec §1): the pre-fall moment. The moment itself is
  // a world fact (ADR-0010: it arrives whether or not the player is ready);
  // the three responses are player actions unlocked by this key. The deadline
  // force-raises the moment at relative Y7M10 (code 129/10) if the event
  // chain never delivered it — the last choice must not be killed by
  // precondition timing (硬伤 #3 fallback).
  trigger: all(
    timestampAtOrAfter(129, 1), // relative Y7, initial calibration, TODO(balance)
    eventTriggered("special", "open_discourse_milestone")
  ),
  deadline_year: 129,  // relative Y7M10, initial calibration, TODO(balance)
  deadline_month: 10,
  script: {
    apply: function(ctx) {
      return { effects: [], emit: [] };
    },
  },
});

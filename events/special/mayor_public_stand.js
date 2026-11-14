defineEvent({
  id: "mayor_public_stand",
  title: "公开担当时刻",
  description: "那道无人能解释来源的旧命令摆在桌上。Mayor 没有向居民解释全部，只以本地责任公开拒绝执行——他拒绝的是'不合法的本地命令'，不是任何来自外部的东西。证人在场，部门能确认，Police 愿意执行。法警没有去执行那道命令：司法部没有站在 IT 一边，哪怕它也没有站到 Mayor 一边。当夜驻地安静。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 9, // initial calibration, TODO(balance)
  // S-2 (30-nodes §3, 31-spec §1): the public stand. Ruling: repeatable —
  // the first stand is the convergence precondition, every later stand
  // defends against order snapping back (W recovery, crisis turnover,
  // bailiff probes). The legitimacy delta is the line's core rate lever:
  // the engine's obedience response converts it into police obedience month
  // by month (30-nodes §5). Setting: the Mayor never learns that other
  // silos exist.
  trigger: all(
    eventTriggered("special", "mayor_crisis_stage"),
    siloMetricGte("legitimacy", 0.5) // initial calibration, TODO(balance)
  ),
  effects: [
    siloMetricDelta("rebellion", -0.05),            // initial calibration, TODO(balance)
    professionMetricDeltaAll("panic_value", -0.05), // initial calibration, TODO(balance)
    siloMetricDelta("legitimacy", 0.05),            // initial calibration, TODO(balance)
  ],
});

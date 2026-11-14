defineEvent({
  id: "bailiff_dispatch_evidence_b",
  title: "法警调度记录：越区执法",
  description: "部门间紧张升温时，有人认出几名法警出现在本不该出现的辖区：没有当地部门的请求，没有备案的协助文书，只有口头转述的'上面的安排'。两名法警事后被调离原岗位，调令理由一栏是空白。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 两份独立法警调度证据链之二：与第一份链条触发条件、事实、证人各自独立
  // (10-it-control-credibility §4/§7.4)。
  // TODO(content): 子项 6 的完整调查链落地后，本节点改由调查动作喂入。
  trigger: all(
    eventTriggered("observation", "first_local"),
    siloMetricGte("dept_tension", 0.25)
  ),
  effects: [
    beliefObserve("player", "Judicial_2", 0.95),
    beliefObserve("Judicial", "Judicial_2", 0.9),
  ],
});

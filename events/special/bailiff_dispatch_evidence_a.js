defineEvent({
  id: "bailiff_dispatch_evidence_a",
  title: "法警调度记录：深夜调令",
  description: "动荡期间，司法档案里浮出一份深夜调令：一队法警被从岗位上调走，去一处无人说得清名目的地点'维持秩序'。调令上的签发栏不属于司法部已知的任何办公室。誊抄这份调令编号的档案员，把副本锁进了自己的抽屉。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 两份独立法警调度证据链之一 (10-it-control-credibility §4/§7.4)。
  // 确定性触发：本地观察成立且驻地进入动荡期。观察只进入信念层
  // (玩家与司法部)，不直接产生曝光——私下确信永远不直接加 E (§3.3)。
  // TODO(content): 子项 6 的完整调查链落地后，本节点改由调查动作喂入。
  trigger: all(
    eventTriggered("observation", "first_local"),
    siloMetricGte("rebellion", 0.15)
  ),
  effects: [
    beliefObserve("player", "Judicial_1", 0.95),
    beliefObserve("Judicial", "Judicial_1", 0.9),
  ],
});

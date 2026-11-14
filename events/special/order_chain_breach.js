defineEvent({
  id: "order_chain_breach",
  title: "规则断点",
  description: "你整理的材料不再指向某个人的罪行，而是指向命令链上的断点：一份缺本地合法来源的执法调度、一栏倒填日期的补签、一条绕过正式渠道的征用记录。司法侧有人收下了这份副本——他不是怀疑体制，只是无法忍受手续不全的命令。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // B-1 (26-nodes §1, 27-spec §1): the line's entry node. Materials point at
  // the rule breach rather than personal guilt (choice L-A projection,
  // 01-rule-of-law L-2). The history_burden bar is reached within the opening
  // months (archive_truth_broadcast +0.08 plus early circulation).
  trigger: all(
    eventTriggered("player_action", "action_share_info"),
    siloMetricGte("history_burden", 0.10) // initial calibration, TODO(balance)
  ),
  effects: [
    // First breach material lands on the Judicial side (legacy fragment layer,
    // same pattern as replication_progress).
    professionFragmentAdd("Judicial"),
    siloMetricDelta("history_burden", 0.02), // initial calibration, TODO(balance)
  ],
});

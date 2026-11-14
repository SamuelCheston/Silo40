defineEvent({
  id: "departure_willingness",
  title: "想走的人",
  description: "一次公开处置之后，维修夹层里开始有人低声谈论同一件事：外面。想逃的是恐惧，想推翻的是愤怒，想趁乱往上爬的是野心——他们还互不相识，也还不是任何一支队伍。但离开意愿已经出现，且可以被联络。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 分支二节点一 · 意愿（the Witness）：想走的人出现且可联络。
  // 13 分册 §1：相对 Y1 起、≥3 个部门恐慌达到阈值 (sim 校准起点)。
  trigger: all(
    timestampAtOrAfter(123, 1), // 相对 Y1，TODO(balance)
    professionMetricGteCount("panic_value", 0.5, 3), // TODO(balance)
  ),
  effects: [
    siloMetricDelta("rebellion", 0.03),      // TODO(balance)
    siloMetricDelta("history_burden", 0.02), // TODO(balance)
  ],
});

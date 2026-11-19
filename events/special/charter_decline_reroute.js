defineEvent({
  id: "charter_decline_reroute",
  title: "材料流向市长圈层",
  description: "你收回的那份断点材料没有在抽屉里落灰。这几个月，抄本在市长圈层里传阅：签发栏的空白、编号的断档、被绕过的正式渠道。他的人不提你的名字，但都知道材料从哪儿来。材料换到的不是约束命令的规则，是换人的筹码——他的声望随着每一份站得住的材料往上走。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 延迟回访（拒绝键 +4 月，TODO(balance)）：世界明确引用这次拒绝。
  // 材料改道市长圈层（30-nodes §6）已在 mayor_evidence_step 即时接线，
  // 本回声是改道之后的延迟合法性拉力。二次拒绝走同一条回声。
  trigger: any(
    all(
      eventTriggered("player_action", "CHARTER_CONTACT_DECLINE"),
      monthsSinceEventGte("player_action", "CHARTER_CONTACT_DECLINE", 4)
    ),
    all(
      eventTriggered("player_action", "CHARTER_SECOND_DECLINE"),
      monthsSinceEventGte("player_action", "CHARTER_SECOND_DECLINE", 4)
    )
  ),
  effects: [
    siloMetricDelta("legitimacy", 0.02), // initial calibration, TODO(balance)
  ],
});
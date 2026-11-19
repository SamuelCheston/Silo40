defineEvent({
  id: "charter_second_decline",
  title: "再拒绝：这一次把门带上",
  description: "你把抄本推了回去。这一次他没说'想清楚再来找我'——他点点头，把抄本揣进怀里，站起身。'规程不等人。这句话我说过两遍了。'门轻轻带上。断点材料还在你手里，但去处已经明确：市长圈层在等它们。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 二次敲门的拒绝分支：终局拒绝。材料改道市长圈层——mayor_evidence_step
  // 订阅本键（喂证心跳照常走 Share Info），charter_decline_reroute 延迟回声。
  player_action: {
    id: "CHARTER_SECOND_DECLINE",
    label: "拒绝第二次接触",
    description: "不把断点材料交出去；约章线彻底关闭，材料流向市长圈层。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 0,            // initial calibration, TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("CHARTER_SECOND_DECLINE"),
  script: {
    apply: function(ctx) {
      return { effects: [], emit: [] };
    },
  },
});
defineEvent({
  id: "route_survey_offer",
  title: "接触：机械侧的路线勘察",
  description: "一位机械侧的联系人把你带到检修出口旁的配电间。'闸门、密封、传感盲区，我都能看，'他说，'但看一次就要冒一次的险。你要不要一条真的路，还是只要一个关于外面的说法？'",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 分支二节点二 · 路线（13 分册 §1）：门控接触卡。接受后开启勘察步骤；
  // 拒绝/暂缓各自留键。接触方是 Mechanical 联系人（既有实体，不新增组织）。
  player_action: {
    id: "ROUTE_SURVEY_OFFER",
    label: "接触机械侧勘察人",
    description: "机械侧联系人愿意勘察检修出口与外部通道，但要先谈清条件。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 1, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ROUTE_SURVEY_OFFER"),
  script: {
    apply: function(ctx) {
      return { effects: [], emit: [] };
    },
  },
});

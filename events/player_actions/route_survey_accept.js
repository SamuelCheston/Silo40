defineEvent({
  id: "route_survey_accept",
  title: "接受：开始勘察路线",
  description: "你们约定了暗号与回程时限。勘察从检修出口开始：哪一段能走、哪一段要绕、哪一段的传感还活着。路线是事实，不是计划——事实在一次次的实地丈量里慢慢成形。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 接受键：解锁路线勘察步骤（special:route_survey_step 读本键）。
  player_action: {
    id: "ROUTE_SURVEY_ACCEPT",
    label: "接受勘察",
    description: "委托机械侧联系人实地勘察撤离路线。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 0, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ROUTE_SURVEY_ACCEPT"),
  effects: [
    siloMetricDelta("rebellion", 0.01), // TODO(balance)
  ],
});

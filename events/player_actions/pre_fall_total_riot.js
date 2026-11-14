defineEvent({
  id: "pre_fall_total_riot",
  title: "以立即全面暴乱证明开放",
  description: "你选择用最响的方式证明开放：全面暴乱。楼层在震动，仓库被打开，旧秩序的每一个符号都被拖到街上。开放确实被看见了——以维生基础为代价。如果社会维存因此不可逆地崩溃，证明开放的那一刻也就没有了可以开放的明天。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // T-2 total riot (33-nodes §3, 34-spec §1): openness proven by force.
  // Sustenance damage is real; whether it crosses into FRACTURE is settled
  // by the existing social-state facts, not by this key.
  player_action: {
    id: "PRE_FALL_TOTAL_RIOT",
    label: "全面暴乱证明开放",
    description: "以全面暴乱证明开放；维存受损，按社会状态接受后果。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 0,            // initial calibration, TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("PRE_FALL_TOTAL_RIOT"),
  effects: [
    siloMetricDelta("rebellion", 0.15), // initial calibration, TODO(balance)
    floorStabilityDelta("", -0.1),      // all floors; initial calibration, TODO(balance)
  ],
});

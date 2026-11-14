defineEvent({
  id: "pre_fall_open_inquiry",
  title: "维持最低秩序，公开质询照常",
  description: "你用已有的关系、威望与部门协作维持最低秩序：今晚的配给与值守照常，同时让公开核验与质询照常进行。居民可以接受秩序，不必承认外部讨论永远是罪。这是最难的一条路——它要求你为一个并不完全受自己控制的公共讨论承担风险。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // T-2 open inquiry (33-nodes §3, 34-spec §1): the expected path. Minimum
  // order plus open inquiry; the snapshot qualification is earned by the
  // whole run's cohort ideology, not granted by this key.
  player_action: {
    id: "PRE_FALL_OPEN_INQUIRY",
    label: "维持秩序，质询照常",
    description: "以既有关系与部门协作维持最低秩序，公开质询照常进行。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 0,            // initial calibration, TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("PRE_FALL_OPEN_INQUIRY"),
  effects: [
    siloMetricDelta("history_burden", 0.03), // initial calibration, TODO(balance)
  ],
});

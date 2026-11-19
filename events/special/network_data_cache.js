defineEvent({
  id: "network_data_cache",
  title: "邻堡数据缓存",
  description: "邻堡失联后，恢复节点中发现了其遗留的数据缓存。打捞出的材料优先流向掌握档案最少的部门，旧事因此再度浮出水面。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Causal followup of the neighbor silo going silent.
  trigger: eventTriggered("history", "neighbor_silo_silence"),
  effects: [
    // Two fragments to the laggard department (empty profession = fewest
    // known fragments; empty fragment = random unknown, seeded RNG).
    professionFragmentAdd(),
    professionFragmentAdd(),
    siloMetricDelta("history_burden", 0.03),
    siloMetricDelta("rebellion", 0.03),
  ],
});

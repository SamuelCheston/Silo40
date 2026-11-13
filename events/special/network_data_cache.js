defineEvent({
  id: "network_data_cache",
  title: "邻堡数据缓存",
  description: "邻堡失联后，恢复节点中发现了其遗留的数据缓存。IT 从里面打捞出两份从未见过的档案碎片。",
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

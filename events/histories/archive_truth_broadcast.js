defineEvent({
  id: "archive_truth_broadcast",
  title: "档案真相广播",
  description: "一批封存档案经公共信息渠道披露，居民开始接触曾被清除的历史。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  trigger: timestampAtOrAfter(122, 1),
  effects: [
    siloMetricDelta("history_burden", 0.08),
    siloMetricDelta("rebellion", 0.03),
  ],
});

defineEvent({
  id: "archive_truth_broadcast",
  title: "档案真相广播",
  description: "一段被封存的历史广播突然在公共频道循环播放，居民第一次意识到建堡初期曾发生过大规模信息清洗。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  trigger: timestampAtOrAfter(122, 1),
  effects: [
    siloMetricDelta("history_burden", 0.08),
    siloMetricDelta("legitimacy", -0.03),
  ],
});

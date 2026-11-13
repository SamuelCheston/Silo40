defineEvent({
  id: "archive_salvage_operation",
  title: "档案打捞",
  description: "IT 借着清洗令的混乱反向操作，从待销毁队列里偷偷打捞档案。每次只敢捞一份，流向最需要它的部门。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 12,
  trigger: all(
    timestampAtOrAfter(123, 6),
    siloMetricGte("history_burden", 0.10),
  ),
  effects: [
    // One fragment to the laggard department per salvage.
    professionFragmentAdd(),
  ],
});

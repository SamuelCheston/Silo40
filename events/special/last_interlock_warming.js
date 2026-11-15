defineEvent({
  id: "last_interlock_warming",
  title: "保障机制升温",
  description: "底层核心逻辑的自检频率变了。IT 的干预动作越来越多、越来越重，而每一次干预都在唤醒那套比所有人都老的保障装置——它不站队，只等阈值。有人开始明白：旧秩序在自己的地基里埋了最后一道装置，它保护驻地的方式，可能是杀死它。",
  type: "INTERNAL",
  fire_mode: "ONCE",
  // DEATH omen, safeguard sub-path (29-failure-endings §2.5): the other two
  // death omens (ultimatum chain, strike reconnaissance) already ship; the
  // safeguard path had none. safeguard_risk is a readable silo metric, so the
  // warning uses the existing monthly channel. Deterministic threshold.
  trigger: { type: "silo_metric_gte", metric: "safeguard_risk", value: 0.6 }, // initial calibration, TODO(balance)
  effects: [
    professionMetricDeltaAll("panic_value", 0.03), // initial calibration, TODO(balance)
  ],
});

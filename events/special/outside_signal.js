defineEvent({
  id: "outside_signal",
  title: "接收到外部信号",
  description: "IT部门截获了一段模糊的无线电信号，似乎来自地表或其他地堡。恐慌暂时退潮，但'外面也许有人'的念头从此扎了根。",
  type: "EXTERNAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 12,
  trigger: timestampAtOrAfter(122, 1),
  effects: [
    professionMetricDeltaAll("panic_value", -0.04),
    siloMetricDelta("history_burden", 0.02),
  ],
});

defineEvent({
  id: "outside_signal",
  title: "接收到外部信号",
  description: "IT部门截获了一段模糊的无线电信号，似乎来自地表或其他地堡。",
  type: "EXTERNAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 12,
  effects: [
    professionIdeologyDeltaAll("pro_foreign", 0.05),
  ],
});

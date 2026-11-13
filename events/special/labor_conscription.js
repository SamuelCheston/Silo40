defineEvent({
  id: "labor_conscription",
  title: "强制轮班",
  description: "人口锐减之后，管理层签发强制轮班令，把丧亲者与伤员编入生产线。产量回来了，别的东西没有。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 9,
  // No JS helper exists for population triggers; raw object matches the
  // engine-supported "silo_population_lte" trigger type.
  trigger: { type: "silo_population_lte", value: 6000 },
  effects: [
    siloResourceDelta("Supplies", 600),
    scheduleEvent("conscription_backlash", 3),
  ],
});

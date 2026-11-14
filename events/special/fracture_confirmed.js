defineEvent({
  id: "fracture_confirmed",
  title: "断层确认",
  description: "维存崩溃已经成为不可撤销的事实。余下的人口或许还能挣扎一时，原有驻地却不再具备恢复为社会的基础。",
  type: "INTERNAL",
  fire_mode: "ONCE",
  script: {
    canTrigger: function(ctx) {
      return false;
    },
  },
  effects: [
    siloMetricDelta("history_burden", 0.03),
  ],
});

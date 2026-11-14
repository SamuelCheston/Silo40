defineEvent({
  id: "reset_execution",
  title: "重置执行令",
  description: "请领单转成了执行令。投药、封档与带人的队伍开始行动；此前是否已经把副本和接收者送出清洗范围，将决定这次处置是完成，还是在抵抗中空转。",
  type: "CRISIS",
  fire_mode: "ONCE",
  script: {
    canTrigger: function(ctx) {
      return false;
    },
    apply: function(ctx) {
      var countermeasure = ctx.states["player_action:reset_countermeasure_done"];
      if (countermeasure && countermeasure.triggered) {
        return { effects: [], emit: [] };
      }
      return {
        effects: [
          siloPopulationDelta(-1500),
          siloMetricDelta("history_burden", -0.15),
          professionMetricDeltaAll("panic_value", 0.10),
          siloMetricDelta("rebellion", 0.06),
        ],
        emit: ["special:reset_completed"],
      };
    },
  },
});

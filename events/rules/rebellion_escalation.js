defineEvent({
  id: "rebellion_escalation_rule",
  title: "叛乱升级规则",
  description: "当叛乱值超过 0.6 时，冲突造成人口伤亡。",
  type: "RULE",
  fire_mode: "REPEATABLE",
  cooldown_months: 1,
  script: {
    canTrigger: function(ctx) {
      return ctx.silo_metrics["rebellion"] > 0.6;
    },
    apply: function(ctx) {
      const extra = Math.floor(ctx.total_population * 0.005);
      return {
        effects: [
          siloPopulationDelta(-extra)
        ]
      };
    }
  }
});

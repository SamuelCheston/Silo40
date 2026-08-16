defineEvent({
  id: "supplies_shortage_rule",
  title: "物资短缺规则",
  description: "当物资低于 200 时，地堡凝聚力下降且亲外思潮上升。",
  type: "RULE",
  fire_mode: "REPEATABLE",
  cooldown_months: 1,
  script: {
    canTrigger: function(ctx) {
      return (ctx.resources["Supplies"] || 0) < 200;
    },
    apply: function(ctx) {
      return {
        effects: [
          siloMetricDelta("cohesion", -0.02),
          professionIdeologyDeltaAll("pro_foreign", 0.02)
        ]
      };
    }
  }
});

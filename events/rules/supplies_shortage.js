defineEvent({
  id: "supplies_shortage_rule",
  title: "物资短缺规则",
  description: "物资储备逐级恶化：配给越来越稀，队伍越来越长，恐慌随档位加深。",
  type: "RULE",
  fire_mode: "REPEATABLE",
  cooldown_months: 4,
  script: {
    canTrigger: function(ctx) {
      return (ctx.resources["Supplies"] || 0) < 1500;
    },
    apply: function(ctx) {
      // Boiling-frog tiers (mutual-adjustment ruler: panic small 0.02-0.04):
      // <200 severe, <800 biting, else mild.
      const supplies = ctx.resources["Supplies"] || 0;
      if (supplies < 200) {
        return { effects: [professionMetricDeltaAll("panic_value", 0.04)] };
      }
      if (supplies < 800) {
        return { effects: [professionMetricDeltaAll("panic_value", 0.03)] };
      }
      return { effects: [professionMetricDeltaAll("panic_value", 0.02)] };
    }
  }
});

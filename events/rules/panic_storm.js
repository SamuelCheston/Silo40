defineEvent({
  id: "panic_storm_delay_rule",
  title: "恐慌风暴规则",
  description: "当恐慌值过高时，调度延时暴乱事件。",
  type: "RULE",
  fire_mode: "REPEATABLE",
  cooldown_months: 6,
  script: {
    canTrigger: function(ctx) {
      // 检查是否有任何职业的恐慌值 > 0.7
      return ctx.professions.some(p => p.panic_value > 0.7);
    },
    apply: function(ctx) {
      return {
        effects: [
          scheduleEvent("delayed_riot", 3)
        ]
      };
    }
  }
});

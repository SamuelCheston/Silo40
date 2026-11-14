defineEvent({
  id: "fracture_point_of_no_return",
  title: "不可逆点",
  description: "能源或人口已经跌破维存恢复线。叛乱仍在继续，但此刻争夺的不再是谁来裁决，而是驻地还能否维持任何社会。",
  type: "CRISIS",
  fire_mode: "ONCE",
  trigger: any(
    all(
      { type: "silo_resource_lte", metric: "Energy", value: 0 },
      siloMetricGte("rebellion", 0.9),
    ),
    all(
      { type: "silo_population_lte", value: 3000 },
      siloMetricGte("rebellion", 0.8),
    ),
  ),
  script: {
    apply: function(ctx) {
      return { effects: [], emit: ["special:fracture_confirmed"] };
    },
  },
});

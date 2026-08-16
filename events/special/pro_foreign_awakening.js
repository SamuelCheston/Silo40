defineEvent({
  id: "pro_foreign_awakening",
  title: "亲外思潮觉醒",
  description: "多个部门的亲外倾向越过临界点，旧有秩序的正当性开始被公开质疑。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  trigger: professionIdeologyGteCount("pro_foreign", 0.5, 4),
  effects: [
    siloMetricDelta("legitimacy", -0.05),
  ],
});

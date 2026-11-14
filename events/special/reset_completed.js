defineEvent({
  id: "reset_completed",
  title: "遗忘完成",
  description: "减口、药物抹忆与清档已经完成，公共接收网络也被拆除。醒来的人认得走廊，却认不出那些追问为何曾经存在；官方记录里只剩一次寻常事故。",
  type: "INTERNAL",
  fire_mode: "ONCE",
  script: {
    canTrigger: function(ctx) {
      return false;
    },
  },
  effects: [
    siloMetricDelta("history_burden", 0.02),
  ],
});

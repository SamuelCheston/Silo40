defineEvent({
  id: "bailiff_evidence_public_b",
  title: "公开披露：越区执法",
  description: "越区执法的目击证词与空白调令理由被整理成一份可公开查阅的记录：谁在场、谁被调走、哪一栏是空的。司法部核对了自己的出勤簿，确认了名单。记录公开的那天，没有部门出面否认。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 与 bailiff_evidence_public_a 独立的第二条发布链 (§3.7)。
  // TODO(content): 子项 6 落地玩家发布动作后，本节点改为玩家主动触发。
  script: {
    canTrigger: function(ctx) {
      var beliefs = (ctx.actor && ctx.actor.beliefs) || [];
      var b = 0;
      for (var i = 0; i < beliefs.length; i++) {
        if (beliefs[i].fact_id === "Judicial_2") b = beliefs[i].b;
      }
      if (b < 0.90) return false;
      return (ctx.public_evidence || []).indexOf("Judicial_2") === -1;
    },
  },
  effects: [
    publicEvidenceRegister("Judicial_2", 1.0, 0.9),
    beliefObserve("Judicial", "Judicial_2", 1.0),
    beliefObserve("residents", "Judicial_2", 0.6),
  ],
});

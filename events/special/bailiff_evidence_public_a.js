defineEvent({
  id: "bailiff_evidence_public_a",
  title: "公开披露：深夜调令",
  description: "调令的誊抄副本被放进公开可查的档案位：签发栏的空白、被调走的名单、无人认领的'维持秩序'。司法部自己的记录员核对了编号，没有提出异议。从这一天起，这份调令不再是抽屉里的秘密。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 发布门槛 (§3.6)：玩家对事实的确信 ≥0.90 才可公开；公开登记 (§3.7)
  // provenance=1.0 (已核验档案)、coverage=0.9。公开同时让司法部与具名
  // 人物吸收观察——Q 只读司法部层信念与登记，玩家私下确信不直接进 Q。
  // TODO(content): 子项 6 落地玩家发布动作后，本节点改为玩家主动触发。
  script: {
    canTrigger: function(ctx) {
      var beliefs = (ctx.actor && ctx.actor.beliefs) || [];
      var b = 0;
      for (var i = 0; i < beliefs.length; i++) {
        if (beliefs[i].fact_id === "Judicial_1") b = beliefs[i].b;
      }
      if (b < 0.90) return false;
      return (ctx.public_evidence || []).indexOf("Judicial_1") === -1;
    },
  },
  effects: [
    publicEvidenceRegister("Judicial_1", 1.0, 0.9),
    beliefObserve("Judicial", "Judicial_1", 1.0),
    beliefObserve("residents", "Judicial_1", 0.6),
  ],
});

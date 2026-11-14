defineEvent({
  id: "publish_evidence",
  title: "公开发布证据",
  description: "把核验过的材料放进公共可查的位置：档案室、公告板、每个能读到字的人手里。发布是单向的——从这一刻起，它不再只属于你。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 2,
  player_action: {
    id: "publish_evidence",
    label: "公开发布证据",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 20,
    duration_months: 0,
  },
  script: {
    // §3.6：B ≥ 0.90 且材料已核验才可发布；发布过的事实不再重复发布。
    canTrigger: function(ctx) {
      return findPublishableFact(ctx) !== "";
    },
    apply: function(ctx) {
      var fact = findPublishableFact(ctx);
      if (fact === "") return {};
      // §3.7：已核验档案级 provenance=1.0；coverage 取公开传播档 0.9
      // (TODO(balance): 之后接宣传/威望决定实际覆盖率)。公开传播让全部部门
      // 与具名人物按各自层级吸收 (§3.3)；丑闻动摇法警忠诚与司法部情绪
      // (§4：一次关键忠诚损失 ≈ -5，TODO(balance))。
      return {
        effects: [
          publicEvidenceRegister(fact, 1.0, 0.9),
          beliefObserve("departments", fact, 0.85),
          beliefObserve("residents", fact, 0.6),
          allianceLoyaltyDelta("bailiff", -0.25),
          allianceLoyaltyDelta("it_internal", -0.15),
          // §4：一次关键忠诚损失 ≈ -5 均衡压力 (经冲击队列，TODO(balance))。
          authorityWShock(-5),
          // 丑闻直接冲击被曝光的司法与 IT 两部门 (§4)，不泛化为全社会恐慌。
          { type: "profession_metric_delta", profession: "Judicial", metric: "panic_value", value: 0.25 },
          { type: "profession_metric_delta", profession: "IT", metric: "panic_value", value: 0.25 },
        ],
      };
    },
  },
});

// 确定性选择：已核验、未发布、确信 ≥0.90 的核心事实，按 ID 升序取第一个。
function findPublishableFact(ctx) {
  var beliefs = (ctx.actor && ctx.actor.beliefs) || [];
  var verified = ctx.verified_evidence || [];
  var published = ctx.public_evidence || [];
  var candidates = [];
  for (var i = 0; i < beliefs.length; i++) {
    var belief = beliefs[i];
    if (belief.fact_id !== "Judicial_1" && belief.fact_id !== "Judicial_2") continue;
    if ((belief.b || 0) < 0.90) continue;
    if (verified.indexOf(belief.fact_id) === -1) continue;
    if (published.indexOf(belief.fact_id) !== -1) continue;
    candidates.push(belief.fact_id);
  }
  candidates.sort();
  return candidates.length > 0 ? candidates[0] : "";
}

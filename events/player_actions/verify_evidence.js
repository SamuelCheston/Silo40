defineEvent({
  id: "verify_evidence",
  title: "核验档案",
  description: "把材料送进可复核的渠道：比对编号、核对签发栏、让经手人确认。核验过的档案才算得上已核验来源，这是公开发布的前置。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 1,
  player_action: {
    id: "verify_evidence",
    label: "核验档案",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 15,
    duration_months: 1,
  },
  script: {
    // §3.6：B ≥ 0.85 才能支撑正式举报/申请归档级的核验。
    canTrigger: function(ctx) {
      return findVerifiableFact(ctx) !== "";
    },
    apply: function(ctx) {
      var fact = findVerifiableFact(ctx);
      if (fact === "") return {};
      return { effects: [evidenceVerify(fact)] };
    },
  },
});

// 确定性选择：已核验名单之外、确信 ≥0.85 的核心事实，按 ID 升序取第一个。
function findVerifiableFact(ctx) {
  var beliefs = (ctx.actor && ctx.actor.beliefs) || [];
  var verified = ctx.verified_evidence || [];
  var candidates = [];
  for (var i = 0; i < beliefs.length; i++) {
    var belief = beliefs[i];
    if (belief.fact_id !== "Judicial_1" && belief.fact_id !== "Judicial_2") continue;
    if ((belief.b || 0) < 0.85) continue;
    if (verified.indexOf(belief.fact_id) !== -1) continue;
    candidates.push(belief.fact_id);
  }
  candidates.sort();
  return candidates.length > 0 ? candidates[0] : "";
}

defineEvent({
  id: "investigate_dispatch",
  title: "定向调查：法警调度",
  description: "顺着调令、出勤簿与目击者的缝隙往下查：谁被调走、以谁的名义、去了哪里。定向调查需要已有相当确信的材料打底。这一轮，先沿你掌握得较少的那条调度线继续核对。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 1,
  player_action: {
    id: "investigate_dispatch",
    label: "调查法警调度",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 15,
    duration_months: 1,
  },
  script: {
    // §3.6：B ≥ 0.60 才能支撑定向调查；两条核心证据链至少一条已有相当确信。
    canTrigger: function(ctx) {
      var beliefs = (ctx.actor && ctx.actor.beliefs) || [];
      var best = 0;
      for (var i = 0; i < beliefs.length; i++) {
        if (beliefs[i].fact_id === "Judicial_1" || beliefs[i].fact_id === "Judicial_2") {
          if (beliefs[i].b > best) best = beliefs[i].b;
        }
      }
      return best >= 0.60;
    },
    // 确定性选择较弱的一条链深挖 (同分取 Judicial_1)。
    apply: function(ctx) {
      var beliefs = (ctx.actor && ctx.actor.beliefs) || [];
      var b1 = 0;
      var b2 = 0;
      for (var i = 0; i < beliefs.length; i++) {
        if (beliefs[i].fact_id === "Judicial_1") b1 = beliefs[i].b;
        if (beliefs[i].fact_id === "Judicial_2") b2 = beliefs[i].b;
      }
      var target = b1 <= b2 ? "Judicial_1" : "Judicial_2";
      return { effects: [beliefObserve("player", target, 0.9)] };
    },
  },
});

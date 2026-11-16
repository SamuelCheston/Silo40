defineEvent({
  id: "expose_execution_failure",
  title: "公开执行失败",
  description: "把一次有目共睹的执行失败摆到台面上：命令是谁下的、现场发生了什么、为什么没有人出来认领。这份公开记录削弱了幕后权威，也让司法部与 IT 部门紧张起来。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 3,
  player_action: {
    id: "expose_execution_failure",
    label: "公开执行失败",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 20,
    duration_months: 1,
  },
  script: {
    // 前置：本局必须真的发生过危机 (执行失败的现场)，且玩家对相关事实
    // 已有 ≥0.60 的确信 (§3.6 定向调查档)。
    canTrigger: function(ctx) {
      var states = ctx.states || {};
      var crisisFired = false;
      for (var key in states) {
        if (key.indexOf("crisis:") === 0 && states[key].triggered) {
          crisisFired = true;
          break;
        }
      }
      if (!crisisFired) return false;
      var beliefs = (ctx.actor && ctx.actor.beliefs) || [];
      var best = 0;
      for (var i = 0; i < beliefs.length; i++) {
        if (beliefs[i].fact_id === "Judicial_1" || beliefs[i].fact_id === "Judicial_2") {
          if (beliefs[i].b > best) best = beliefs[i].b;
        }
      }
      return best >= 0.60;
    },
    // §4/§7.4：一次公开执行失败 ≈ -8 均衡压力 (经冲击队列)；在场者同时得到
    // 调度链的公开观察；法警忠诚受挫 (关键忠诚损失档，TODO(balance))。
    apply: function(ctx) {
      return {
        effects: [
          authorityWShock(-8),
          beliefObserve("residents", "Judicial_2", 0.7),
          beliefObserve("departments", "Judicial_2", 0.6),
          allianceLoyaltyDelta("bailiff", -0.10),
          allianceLoyaltyDelta("it_internal", -0.05),
          { type: "profession_metric_delta", profession: "Judicial", metric: "panic_value", value: 0.15 },
          { type: "profession_metric_delta", profession: "IT", metric: "panic_value", value: 0.10 },
        ],
      };
    },
  },
});

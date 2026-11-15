defineEvent({
  id: "escape_prep_suit_warehouse",
  title: "防护服库开启",
  description: "IT 把清洁服库的钥匙交出来的那天，没有仪式。库里的服体按完好率清点入库，不再需要逐套暗中修复——破坏者停止了破坏。这是'配合'两个字的实物形态。",
  type: "TECHNICAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 1, // TODO(balance)
  // 修服链（16 §2.4 立场口径）：IT 立场为"配合"时库开启，可用服 = 库存 ×
  // 完好率，来源二暴露成本消失。canTrigger 读立场——立场塑形准备可得性。
  player_action: {
    id: "ESCAPE_PREP_SUIT_WAREHOUSE",
    label: "接收防护服库",
    description: "IT 配合下开启防护服库，按完好率入库。需要 IT 立场为配合。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 1, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_PREP_SUIT_WAREHOUSE"),
  script: {
    canTrigger: function(ctx) {
      var stance = ctx.top_stance || {};
      return stance.it === "cooperate";
    },
    apply: function(ctx) {
      return {
        effects: [
          preparationDelta("suits_functional", 4), // 库存 × 完好率，TODO(balance)
        ],
        emit: [],
      };
    },
  },
});

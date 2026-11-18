const transferAmount = 100; // TODO(balance)

defineEvent({
  id: "escape_prep_transfer_materials",
  title: "划拨：封存装备材料",
  description: "材料库的账目被做平了一次：100 材料从公用库存划进封存。维修与补给转化可用的材料也随之减少。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 1, // TODO(balance)
  // 划拨链（16 §2.1）：Materials → 封存账本，划拨即扣减，挤压维修与生产。
  player_action: {
    id: "ESCAPE_PREP_TRANSFER_MATERIALS",
    label: "划拨封存材料",
    description: "从堡内库存划拨 100 材料，用于装备与生产起点载体。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_PREP_TRANSFER_MATERIALS"),
  script: {
    canTrigger: function(ctx) {
      return (ctx.resources || {}).Materials >= transferAmount;
    },
    apply: function(ctx) {
      var effects = [
        preparationTransfer("Materials", transferAmount), // TODO(balance)
      ];
      var stance = ctx.top_stance || {};
      if (stance.mayor !== "cooperate") {
        effects.push(preparationDelta("prep_exposure", 0.04)); // TODO(balance)
      }
      return { effects: effects, emit: [] };
    },
  },
});

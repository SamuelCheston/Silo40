defineEvent({
  id: "escape_prep_transfer_materials",
  title: "划拨：封存装备材料",
  description: "材料库的账目被做平了一次：一批装备材料与生产起点载体从维修与生产的份额里被挤出来，转进封存。堡里的机器会因此慢下来——这也是代价的一部分。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 1, // TODO(balance)
  // 划拨链（16 §2.1）：Materials → 封存账本，划拨即扣减，挤压维修与生产。
  player_action: {
    id: "ESCAPE_PREP_TRANSFER_MATERIALS",
    label: "划拨封存材料",
    description: "从堡内材料中划拨一份封存为装备材料（含生产起点载体）。立即扣减堡内资源。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_PREP_TRANSFER_MATERIALS"),
  script: {
    apply: function(ctx) {
      var effects = [
        preparationTransfer("Materials", 100), // TODO(balance)
      ];
      var stance = ctx.top_stance || {};
      if (stance.mayor !== "cooperate") {
        effects.push(preparationDelta("prep_exposure", 0.04)); // TODO(balance)
      }
      return { effects: effects, emit: [] };
    },
  },
});

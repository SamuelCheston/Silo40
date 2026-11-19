const transferAmount = 200; // TODO(balance)

defineEvent({
  id: "escape_prep_transfer_supplies",
  title: "划拨：封存便携补给",
  description: "补给从配给线上被截下一段，转进封存的账本。堡内库存少了 200，准备的代价记在每一个还没离开的人头上。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 1, // TODO(balance)
  // 划拨链（16 §2.1）：Supplies → 封存账本，划拨即扣减。机会成本立即生效：
  // 喂大 supplies_shortage 一类既有规则。市长立场塑形暴露成本：
  // 阻挠/不阻挠下暗中划拨被审计记账；配合下为正式调拨，不计暴露。
  player_action: {
    id: "ESCAPE_PREP_TRANSFER_SUPPLIES",
    label: "划拨封存补给",
    description: "从堡内库存划拨 200 补给，封存为便携补给（人·月）。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_PREP_TRANSFER_SUPPLIES"),
  script: {
    canTrigger: function(ctx) {
      return (ctx.resources || {}).Supplies >= transferAmount;
    },
    apply: function(ctx) {
      var effects = [
        preparationTransfer("Supplies", transferAmount), // TODO(balance)：划拨量与人·月换算
      ];
      var stance = ctx.top_stance || {};
      if (stance.mayor !== "cooperate") {
        effects.push(preparationDelta("prep_exposure", 0.04)); // 暗中划拨被审计，TODO(balance)
      }
      return { effects: effects, emit: [] };
    },
  },
});

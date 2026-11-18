const transferAmount = 150; // TODO(balance)

defineEvent({
  id: "escape_reception_stocking",
  title: "接应点囤备",
  description: "先遣确认过的接应点开始接收首批补给：150 补给从堡内划拨，按批次埋设、按暗号交接。接应点是实的，物资也是实的。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 接应建设（16 §2.1/§6.3）：与先遣链共用完成键，写 ReceptionStage 档位。
  // confirmed → stocked：向接应点划拨封存补给。
  player_action: {
    id: "ESCAPE_RECEPTION_STOCKING",
    label: "向接应点囤备物资",
    description: "向已确认且尚未囤备的接应点划拨 150 堡内补给。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_RECEPTION_STOCKING"),
  script: {
    canTrigger: function(ctx) {
      var prep = ctx.escape_prep || {};
      return prep.reception_stage === "confirmed" && (ctx.resources || {}).Supplies >= transferAmount;
    },
    apply: function(ctx) {
      return {
        effects: [
          preparationTransfer("Supplies", transferAmount), // TODO(balance)
          preparationStage("stocked"),
        ],
        emit: [],
      };
    },
  },
});

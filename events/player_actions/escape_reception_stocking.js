defineEvent({
  id: "escape_reception_stocking",
  title: "接应点囤备",
  description: "先遣确认过的接应点开始接收第一批封存补给：按批次埋设、按暗号交接。后面的人不再是'应该已经到了'的猜测——接应点是实的，物资是实的。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 接应建设（16 §2.1/§6.3）：与先遣链共用完成键，写 ReceptionStage 档位。
  // confirmed → stocked：向接应点划拨封存补给。
  player_action: {
    id: "ESCAPE_RECEPTION_STOCKING",
    label: "向接应点囤备物资",
    description: "向已确认的接应点划拨补给。需要先遣往返确认。",
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
      return prep.reception_stage === "confirmed" || prep.reception_stage === "stocked";
    },
    apply: function(ctx) {
      var prep = ctx.escape_prep || {};
      if (prep.reception_stage === "stocked") {
        return { effects: [], emit: [] }; // 已囤备：不重复划拨
      }
      return {
        effects: [
          preparationTransfer("Supplies", 150), // TODO(balance)
          preparationStage("stocked"),
        ],
        emit: [],
      };
    },
  },
});

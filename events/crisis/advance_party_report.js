defineEvent({
  id: "advance_party_report",
  title: "先遣回报",
  description: "先遣小组的消息按约定方式传回。路线能不能重复使用、接应点是不是真实存在、后面的人能不能跟上——答案都在这一次回报里。",
  type: "TECHNICAL",
  fire_mode: "ONCE",
  // 纯调度事件（13 分册 §1）：由 advance_party_dispatch 延迟 4 个月点火。
  // apply 守卫区分完好/受损：先遣不在场期间 (派遣之后) 发生底层坍塌事故则
  // 路线受损。读时间戳而不是"是否曾经触发"——坍塌在派遣之前发生过属于既有
  // 地形事实，先遣正是去实测它的，不构成路线受损。
  script: {
    canTrigger: function(ctx) {
      return false;
    },
    apply: function(ctx) {
      var states = ctx.states || {};
      var collapse = states["crisis:lower_deck_collapse"];
      var dispatch = states["player_actions:advance_party_dispatch"];
      var damaged = !!(collapse && collapse.triggered &&
        (!dispatch ||
          (collapse.last_triggered_timestamp || 0) >= (dispatch.last_triggered_timestamp || 0)));
      if (damaged) {
        // 受损分支：接应档位停在 surveyed（未确认），存活率结算走地板值。
        return {
          effects: [
            professionMetricDeltaAll("panic_value", 0.05), // TODO(balance)
            siloMetricDelta("rebellion", 0.01),            // TODO(balance)
          ],
          emit: [],
        };
      }
      // 完好分支：先遣往返确认通行 → 接应档位 confirmed（16 §2.1）。
      return {
        effects: [
          preparationStage("confirmed"),
          siloMetricDelta("rebellion", 0.04), // TODO(balance)
        ],
        emit: [],
      };
    },
  },
});

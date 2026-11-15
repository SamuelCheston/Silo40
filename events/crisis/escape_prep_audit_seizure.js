defineEvent({
  id: "escape_prep_audit_seizure",
  title: "IT 审计：胶带去哪了",
  description: "审计从胶带库存的缺口开始，一路查到维修间的优质胶带。已经修复的防护服被当场换回劣质胶带，或者直接没收。筹备的暴露不是抽象的数字——它是一张清单，清单上的每一项都可能被划掉。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 6, // TODO(balance)
  // 修复可被反制（16 §4.2-3）：筹备暴露达到阈值且 IT 立场仍阻挠时，
  // 审计没收已修服。IT 立场为不阻挠/配合时此风险消失（canTrigger）。
  // 窗口开启前审计不点火：零散修服可被解释为正常维护，IT 尚未把胶带缺口
  // 与撤离计划关联；开窗后撤离成为可观测事实，审计才作为反制启动——此时
  // 保障动作已可用，玩家有手段把 IT 推离阻挠、让风险按设计口径消失。
  trigger: preparationGte("prep_exposure", 0.5), // TODO(balance)
  script: {
    canTrigger: function(ctx) {
      var stance = ctx.top_stance || {};
      if (stance.it !== "block") {
        return false;
      }
      var states = ctx.states || {};
      var windowOpen = states["special:escape_window_open"] &&
        states["special:escape_window_open"].triggered;
      return windowOpen;
    },
  },
  effects: [
    preparationDelta("suits_functional", -2),   // 没收/换回劣质胶带，TODO(balance)
    preparationDelta("prep_exposure", -0.3),    // 暴露随查处被部分"消费"，TODO(balance)
    professionMetricDeltaAll("panic_value", 0.02), // TODO(balance)
  ],
});

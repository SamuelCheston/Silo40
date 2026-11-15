defineEvent({
  id: "silo1_purge_exposed",
  title: "整肃图谋被揭穿",
  description: "请领清单、名册调阅与深夜清点的对应关系被公开讲了出来。当越来越多的人开始互相转告、拒绝单独落单时，那场整肃还没动手就失去了突然性。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // 本地整肃反制的信息侧结算 (20 分册 §4.2/§5)：调度-only，由玩家动作
  // PURGE_PLOT_LEAK 在征兆后、执行前唤起。宣传不是反制，所以本节点不接宣传
  // 门控（既有裁决不变）；绑定的是"把清点的时间讲出去"这个日常小动作。
  //
  // 与名册篡改（silo1_purge_countermeasure，物证侧）分工：本节点不写任何数值，
  // 只留下"图谋已公开"的事实；结局后果由 silo1_purge_execution 按中档结算。
  trigger: eventTriggered("special", "silo1_purge_omen"),
  script: {
    canTrigger: function(ctx) {
      // Scheduled-only settlement hook: fired via schedule_event from
      // silo1_purge_plot_leak.
      return false;
    },
    apply: function(ctx) {
      // No arithmetic here: the exposed grade is read by the execution node.
      return {};
    },
  },
});

defineEvent({
  id: "silo1_purge_softened",
  title: "被动过的名册",
  description: "复核之后，名册上的两页不再对得上：几个本该被划进『转移』段落的名字回到了日常班表，另一批则从纸面上彻底消失。执行整肃的人拿着这份名册做事，会少带走一些人——也会少清掉几行记录。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // 整肃的日常反制结算 (33/34 分册)：玩家在复核夜动过名册之后，整肃执行节点
  // 读本事实决定后果档位。调度-only，由 PURGE_ROSTER_TAMPER 动作唤起，
  // 不参与常规触发扫描。
  script: {
    canTrigger: function(ctx) {
      // Scheduled-only settlement hook: fired via schedule_event from
      // silo1_purge_countermeasure.
      return false;
    },
    apply: function(ctx) {
      // No arithmetic here: the softened grade is read by the execution node.
      return {};
    },
  },
});

defineEvent({
  id: "silo1_purge_execution",
  title: "本地整肃执行",
  description: "整肃在一个夜里开始：名单上的人被分批带走，官方口径是转移与隔离。关于那一夜的记录，随后一并消失了。留下的人学会了两件事——别再问，别再记得。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Scheduled-only settlement hook: fired via schedule_event from
  // silo1_purge_omen.
  //
  // TODO(Countermeasure & consequence modeling): the local purge (投药/清档)
  // deliberately produces NO numerical consequence this round. Its outcome must
  // depend on who mounted which countermeasure and how the crackdown lands —
  // a single constant cannot stand in for that. Backfill the settlement only
  // after the countermeasure-and-consequence modeling batch lands; until then
  // this node is a declared hook that records the fact the purge happened.
  script: {
    canTrigger: function(ctx) {
      return false;
    },
    apply: function(ctx) {
      // No numerical consequence yet (see TODO above). The completion key
      // records that the local purge was carried out.
      return {};
    },
  },
});

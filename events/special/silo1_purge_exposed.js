defineEvent({
  id: "silo1_purge_exposed",
  title: "整肃图谋被揭穿",
  description: "请领清单、名册调阅与深夜清点的对应关系被公开讲了出来。当越来越多的人开始互相转告、拒绝单独落单时，那场整肃还没动手就失去了突然性。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // DECLARED-BUT-UNBOUND countermeasure hook (local-execution channel).
  // Ruling: propaganda is NOT a countermeasure and must not act as one, so the
  // former propaganda gate is removed. The real countermeasure will later be
  // hidden inside everyday small actions (contextualized actions with an object,
  // place, timing, material and a trace). Until that system lands this node is
  // intentionally unbound: canTrigger returns false, so it never fires yet.
  //
  // TODO(Countermeasure binding): wire canTrigger to the future everyday-action
  // countermeasure for the information side (the roster-tampering action already
  // binds the material side; see silo1_purge_countermeasure), and give this node
  // a real consequence once the countermeasure-and-consequence modeling lands.
  trigger: eventTriggered("special", "silo1_purge_omen"),
  script: {
    canTrigger: function(ctx) {
      // Unbound: no information-side countermeasure action is wired this round.
      return false;
    },
    apply: function(ctx) {
      // No consequence yet; the material countermeasure is graded by
      // silo1_purge_execution (see its apply script).
      return {};
    },
  },
});

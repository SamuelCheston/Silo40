defineEvent({
  id: "bombing_terminal",
  title: "远程投送打击",
  description: "没有警报，也没有来袭的机群。只有一次从极高处落下的沉重投送，把上层结构连同它下面的一切一并抹平。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Scheduled-only settlement hook: fired via schedule_event from
  // strike_reconnaissance. Ruling: the settlement writes NO population figures
  // and does NOT zero population — it connects directly to the ending
  // arbitration. Its completion key crisis:bombing_terminal is read there as a
  // DEATH fact (《寂堡》). Physical delivery has no local countermeasure; the only
  // mitigation is a surviving escape group, which P0 arbitration reads, not this
  // node.
  script: {
    canTrigger: function(ctx) {
      return false;
    },
    apply: function(ctx) {
      // Settlement hook: no numerical consequence. The completion key itself is
      // the terminal fact; arbitration settles the ending.
      return {};
    },
  },
});

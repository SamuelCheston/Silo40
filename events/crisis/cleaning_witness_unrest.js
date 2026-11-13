defineEvent({
  id: "cleaning_witness_unrest",
  title: "目击者余波",
  description: "围观过清洁的人在底层一遍遍复述同一个细节：他走出去了，走得比谁预想的都稳。恐惧没有落地，反而生了根。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // Scheduled-only event: fired via schedule_event from public_cleaning_sentence.
  script: {
    canTrigger: function(ctx) {
      return false;
    },
  },
  effects: [
    siloMetricDelta("rebellion", 0.04),
    professionMetricDeltaAll("panic_value", 0.06),
    siloMetricDelta("history_burden", 0.03),
  ],
});

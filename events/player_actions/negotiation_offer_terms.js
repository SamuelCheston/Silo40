defineEvent({
  id: "negotiation_offer_terms",
  title: "谈判：把条款摆上桌面",
  description: "赦免名单、职能保留、不清算求生者——条款经由中间人递上了市长的桌面。市长可以一直不表态，但条款在桌，妥协就有了具体的形状。即使你们已经打到了市长办公室，这一步仍然算数。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 谈判层·递条款（18 设计卷 §5）：叛乱方条款在桌，提高保障可信度并
  // 触发一次高强度回合；市长让步即接受条款（提议方不否决自己的提案）。
  player_action: {
    id: "NEGOTIATION_OFFER_TERMS",
    label: "递出妥协条款",
    description: "把赦免与职能保留条款正式递上桌面，提高妥协的现实性。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("NEGOTIATION_OFFER_TERMS"),
  effects: [
    topStanceGuarantee(0.15), // TODO(balance) 条款本身抬高保障可信度
    negotiationRound("terms"),
  ],
});

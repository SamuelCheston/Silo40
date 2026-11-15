defineEvent({
  id: "escape_negotiation_guarantee",
  title: "磋商：给出可信的保障",
  description: "你通过中间人递出一份清单：赦免名单、职能保留、公开证据不用于清算。'我们不审判求生的人，'清单的最后一行写着，'我们只需要门。'这不是谈判的全部——真正的谈判远比这复杂——但它是顶层愿意重新掂量立场的理由。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 顶层链 · 保障动作（16 §3.2/§6.3）：叛乱方给出可信保障，提高市长/IT
  // "不阻挠/配合"的权重，并立即触发一次立场重估。完整谈判机制（市长三条路
  // 中的"谈判"）已由谈判层落地（21 设计卷、negotiation_* 动作），本动作
  // 保留为最小保障动作，与谈判层并存。
  player_action: {
    id: "ESCAPE_NEGOTIATION_GUARANTEE",
    label: "向顶层递出保障清单",
    description: "以赦免与职能保留承诺换取顶层重新掂量立场。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_NEGOTIATION_GUARANTEE"),
  effects: [
    topStanceGuarantee(0.3), // TODO(balance)
    topStanceReeval("both"),
  ],
});

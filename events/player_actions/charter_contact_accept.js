defineEvent({
  id: "charter_contact_accept",
  title: "接受：把断点交给程序",
  description: "你把断点材料的副本推过桌面。他收下的动作很慢，像是在给自己最后一次反悔的机会。'约章要由本地人起草，'他说，'你只负责供证。'从这一天起，起草会的门为这些材料开着，六个月的动能窗口也开始计时。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // T-2 accept (26-nodes §2, 27-spec §1): accepting the contact opens the
  // drafting track and schedules the +6-month guard timeout (K-2).
  player_action: {
    id: "CHARTER_CONTACT_ACCEPT",
    label: "接受接触：交给程序",
    description: "把规则断点材料交给司法侧程序派联系人，开启约章起草线。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 0,            // initial calibration, TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("CHARTER_CONTACT_ACCEPT"),
  effects: [
    scheduleEvent("charter_window_expiry", 6), // initial calibration, TODO(balance)
  ],
});

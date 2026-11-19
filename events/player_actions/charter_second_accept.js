defineEvent({
  id: "charter_second_accept",
  title: "再接受：把起草会重新点起来",
  description: "你把抄本推回桌面。他收下的动作还是很慢，像给自己最后一次反悔的机会。'约章要由本地人起草，'还是六个月前那句话，'你只负责供证。'起草会的灯重新点起来了，但动能得从第一条条款重新攒——六个月的窗口重新开始计时。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 二次敲门的接受分支（charter_defer_knock 解锁）：新完成键重开起草线。
  // charter_draft_step 同时订阅两个接受键；动能窗口守卫重新挂表。
  player_action: {
    id: "CHARTER_SECOND_ACCEPT",
    label: "接受第二次接触",
    description: "把断点材料重新交给程序派联系人，重开约章起草线（动能从零重建）。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 0,            // initial calibration, TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("CHARTER_SECOND_ACCEPT"),
  effects: [
    scheduleEvent("charter_window_expiry", 6), // initial calibration, TODO(balance)
  ],
});
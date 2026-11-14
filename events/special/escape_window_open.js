defineEvent({
  id: "escape_window_open",
  title: "开窗",
  description: "不同部门第一次公开按照一套未经旧权力批准的班次行动：停工是有组织的，占用通道是有秩序的，拒绝封闸是有署名的。这不是暴乱，是驻地长出了另一套骨骼。撤离窗口打开了——顶层的每一个人都在重新掂量自己的立场。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 分支二节点三 · 开窗（the Witness）：有组织反抗打开撤离窗口。
  // 无路径的叛乱锁死本节点：必须有已确认的路线事实；必须有组织
  // （地下协调网络成立）；叛乱水平达到开窗阈值。
  // 开窗即立场重估的第一次生产（16 §3.2 时序链：先有胜负，再有转向）。
  trigger: all(
    siloMetricGte("rebellion", 0.72), // TODO(balance)
    eventTriggered("special", "route_confirmed"),
    eventTriggered("special", "underground_coordination"),
  ),
  effects: [
    topStanceReeval("both"),
    siloMetricDelta("rebellion", 0.05),           // TODO(balance)
    professionMetricDeltaAll("panic_value", 0.03), // TODO(balance)
  ],
});

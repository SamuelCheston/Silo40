defineEvent({
  id: "escape_departure_launch",
  title: "出发",
  description: "批次名单最后一次核对。防护服按人头分发，补给按人月上秤，走不动的人被编进有照护的班次。闸门开启的那一刻，没有人欢呼——所有人都在数人数。这是《越岭》的结算点：能带走多少、路上剩多少、落地后活不活，都在此落定。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 分支二节点四 · 离开（the Witness）：成规模群体实际离开并存续。
  // 结算只看逃出去了多少人（裁决 F）：
  //   1. 终局立场重估——磋商是持续过程，最后一刻妥协也算（16 §3.2）；
  //   2. escape_departure_settle：服务层读立场＋账本＋路线事实，
  //      经容量 D → 存活率 S → 维存 M → 追踪 T 产出逃出群体事实；
  //      仲裁只读 EscapedGroup，不读立场与账本（ADR-0008、Layer 1 纪律）。
  player_action: {
    id: "ESCAPE_DEPARTURE_LAUNCH",
    label: "下令出发",
    description: "按批次启动撤离。结算将读取顶层立场与全部准备账本。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 3, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_DEPARTURE_LAUNCH"),
  effects: [
    topStanceReeval("both"),
    escapeDepartureSettle(),
  ],
});

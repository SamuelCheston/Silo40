defineEvent({
  id: "escape_spillover_physical",
  title: "信道：实体交接",
  description: "最古老的信道：人或物，物理地穿过那段黑暗。逃出者可以携带包裹——但带走的材料首先归幸存事实；要计入外溢，必须有显式的再传递任务。这条规矩写在交接清单的第一行。",
  type: "SOCIAL",
  fire_mode: "REPEATABLE",
  cooldown_months: 4, // TODO(balance)
  // 外溢信道建立（16 §2.3）：实体交接类。与逃跑的耦合点：逃出群体可携带
  // 包裹，但按三线交叉裁决，材料首先归《越岭》幸存事实；再传递任务属 Layer 3。
  player_action: {
    id: "ESCAPE_SPILLOVER_PHYSICAL",
    label: "建立实体交接信道",
    description: "以人或物的物理携带建立跨堡交接信道。",
    scope: "common",
    action_type: "PLAYER_EVENT",
    target_type: "NONE",
    ap_cost: 2, // TODO(balance)
    duration_months: 0,
    unavailable_behavior: "hide",
  },
  trigger: playerActionIs("ESCAPE_SPILLOVER_PHYSICAL"),
  effects: [
    spilloverChannelAdd("physical_handoff", 3, 0.35), // TODO(balance)
    // 通信规模驱动组织怀疑度：实体交接较信号更不显眼（20 修订/ADR-0014）。
    orgSuspicionDelta(0.04), // TODO(balance)
  ],
});

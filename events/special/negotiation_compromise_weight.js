defineEvent({
  id: "negotiation_compromise_weight",
  title: "妥协的分量",
  description: "三个月了。赦免名单上的墨迹干了，承诺的分量却还在落定：接过办公室的人正在学坐那把椅子，转向离开的人在清点自己的人，而迟迟没有表态的——沉默本身也被别人记着账。妥协不是终点，是另一段算账的开始。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 延迟回访（妥协成立键 +3 月，TODO(balance)）：妥协不是快照——三个月后
  // 按妥协之后的真实状态分流算账（读只读视图 negotiation.takeover）：
  //   接管 → 旧班底对委员会入主的不安记在部门张力上（dept_tension +0.02）；
  //   带组织离开 → 组织完整、方向一致，凝聚小量上行（cohesion +0.02）；
  //   悬置（未表态）→ 承诺无人认领，保障可信度磨损（guarantee −0.05）。
  // 悬置不是死胡同：世界明确引用这次沉默，保障磨损会进入后续立场权重。
  trigger: all(
    eventTriggered("negotiation", "compromise_established"),
    monthsSinceEventGte("negotiation", "compromise_established", 3)
  ),
  script: {
    apply: function(ctx) {
      var takeover = ctx.negotiation ? ctx.negotiation.takeover : "";
      if (takeover === "takeover") {
        // 接管：委员会入主，旧班底的不安记在部门裂缝里 (initial calibration, TODO(balance))。
        return { effects: [siloMetricDelta("dept_tension", 0.02)] };
      }
      if (takeover === "hands_off") {
        // 带组织离开：组织完整转向，方向一致凝成凝聚 (initial calibration, TODO(balance))。
        return { effects: [siloMetricDelta("cohesion", 0.02)] };
      }
      // 悬置：承诺摆在那里无人认领，保障的分量被沉默磨损 (initial calibration, TODO(balance))。
      return { effects: [topStanceGuarantee(-0.05)] };
    },
  },
});

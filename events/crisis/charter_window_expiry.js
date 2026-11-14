defineEvent({
  id: "charter_window_expiry",
  title: "约章动能流失",
  description: "六个月的窗口走完了，约章还没有锁进不可逆的程序。起草会的人还在，条款还在，但争吵的劲头散了半截：有人开始掂量重新站队的代价。动能流失不等于线断了——锁定日仍在等一个真正打开的窗口。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // K-2 (26-nodes §4, 27-spec §1): guard timeout scheduled by the accept
  // choice (+6 months). Pure schedule event: dispatched unconditionally, the
  // guard lives in apply (foundation §3). The timeout itself is never
  // subscribed as a failure condition.
  script: {
    canTrigger: function(ctx) {
      return false;
    },
    apply: function(ctx) {
      var lockDay = ctx.states["special:charter_lock_day"];
      if (lockDay && lockDay.triggered) {
        // Already locked: the guard spins down harmlessly.
        return { effects: [], emit: [] };
      }
      // Unlocked branch: momentum bleeds (initial calibration, TODO(balance)).
      return {
        effects: [
          siloMetricDelta("rebellion", 0.04),
          professionMetricDeltaAll("panic_value", 0.04),
        ],
      };
    },
  },
});

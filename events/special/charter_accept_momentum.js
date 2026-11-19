defineEvent({
  id: "charter_accept_momentum",
  title: "起草会的账",
  description: "五个月，足够让一场起草会显出原形。有人把出勤名册和供证记录誊成一张单子，推过桌面——吵过的条款在长根，没人喂的条款在落灰。这一次，轮到你读账。",
  type: "SOCIAL",
  fire_mode: "ONCE",
  // 延迟回访（接受键 +5 月，TODO(balance)）：锚定完成键原形。按起草心跳
  // charter_draft_step 的 trigger_count 分流——供过证，司法侧记得这份供给；
  // 一直沉默，动能原地流失。回访只认选择本身，不剧透锁定日之后的事。
  trigger: all(
    eventTriggered("player_action", "CHARTER_CONTACT_ACCEPT"),
    monthsSinceEventGte("player_action", "CHARTER_CONTACT_ACCEPT", 5)
  ),
  script: {
    apply: function(ctx) {
      var step = ctx.states["special:charter_draft_step"];
      if (step && step.trigger_count >= 1) {
        // 供证分支：条款有本地主人，司法侧的信念层再吸收一次供给
        // (initial calibration, TODO(balance))。
        return {
          effects: [
            beliefObserve("Judicial", "Judicial_1", 0.3),
            siloMetricDelta("history_burden", 0.02),
          ],
        };
      }
      // 沉默分支：窗口还在走，劲头已经散了半截
      // (initial calibration, TODO(balance))。
      return {
        effects: [
          siloMetricDelta("rebellion", 0.02),
          professionMetricDeltaAll("panic_value", 0.02),
        ],
      };
    },
  },
});
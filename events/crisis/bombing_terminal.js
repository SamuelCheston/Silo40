defineEvent({
  id: "bombing_terminal",
  title: "远程投送打击",
  description: "没有警报，也没有来袭的机群。只有一次从极高处落下的沉重投送，把上层结构连同它下面的一切一并抹平。幸存者数不清，因为已经没有人能数了。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Scheduled-only terminal node: fired via schedule_event from
  // strike_reconnaissance. Physical delivery has no local countermeasure by
  // design (《寂堡》链 21)；唯一的“守卫”是人已经离开——逃出群体例外在终局仲裁
  // (P0) 读取，不在本事件内判定。
  script: {
    canTrigger: function(ctx) {
      return false;
    },
  },
  effects: [
    // One-time large negative delta clamps the in-silo population to zero; the
    // next victory_check settles DEATH unless a surviving escape group exists.
    siloPopulationDelta(-1000000),
  ],
});

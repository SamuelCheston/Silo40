defineEvent({
  id: "silo1_destroyed_signal",
  title: "一号地堡失去联系",
  description: "所有与一号地堡的通信协议均已超时，服务器不再响应。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  trigger: timestampAtOrAfter(130, 1),
  effects: [
    siloFlagSet("silo1_destroyed", true),
  ],
});

defineEvent({
  id: "termination_channel_test",
  title: "终止链路自检",
  description: "IT 值夜记录里多了一段无法归档的链路自检：一条久未启用的远程投放通道被点亮，短暂握手后又归于沉寂。居民只当是又一次设备误报；只有极少数人意识到，那条通道的另一端连着活人。",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  // Remote-termination channel omen (《寂堡》链 21)：整肃前兆之后压力仍不回落，
  // 上层控制方点亮终止信道做投放前自检。确定性触发，无随机。阶梯串联在整肃
  // 前兆之后 (＋6 月，TODO(balance))：终止投放 = 通牒+4+6+4 = U+14，物理投送 =
  // U+4+6+6+3 = U+19——无论何时越线，终止总是先落地，且《断层》的快速维生
  // 崩塌 (~U+10) 仍有机会先于任何远程结算，不留"越晚越必然被炸"的隐含规则。
  // 远程信道只对"仍在运转"的驻地开启 (Energy ≥ 1)：维生已经崩塌的堡留给
  // 《断层》自己的结算，两条失败路径按脚本状态读数分流，不做智能判断。
  trigger: all(
    eventTriggered("special", "silo1_purge_omen"),
    monthsSinceEventGte("special", "silo1_purge_omen", 6),
    siloMetricGte("order_pressure", 0.45),
    actorMetricGte("propaganda_level", 2),
    { type: "silo_resource_gte", metric: "Energy", value: 1 }
  ),
  effects: [
    professionMetricDeltaAll("panic_value", 0.05),
    siloMetricDelta("rebellion", 0.03),
    scheduleEvent("termination_release", 4),
  ],
});

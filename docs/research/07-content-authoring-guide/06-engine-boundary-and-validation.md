# Silo40 内容作者手册 · 必须动引擎的清单与本地验证手册

> 拆分自 [Silo40 内容作者手册](../07-content-authoring-guide.md)；目录、章节映射与出处见该索引。

## 10. 内容改不动、必须动引擎的清单

出现以下需求时**先立项改代码**，不要试图用内容绕过：

1. 新效果/触发器/变异类型，或新 flag、新结局、新管线阶段（`content_events.go` 校验 +
   `mechanics.go` 变异 + JS prelude helper 三处同改）。
2. 内容事件动 `safeguard_risk`、直达 cohort 思潮（§8）、按 Function 改楼层。
3. 新事件分组的加载（白名单在 `internal/service/game_service.go`；加完跑防呆闸）。
4. 事件权重/随机导演、on-off 周期、硬保底（当前机制层不存在）。
5. 管线顺序与结算时机（`runTickPipeline` 步骤 1–9 硬编码）。
6. 结局判定优先级（TIME 先于 REBELLION 等；`victory_check` JS + Go fallback 双份同改）。

## 11. 本地验证手册

```bash
# 1. 快速门槛：构建 + 全量测试（含防呆闸与内容加载校验）
go build ./... && go test ./internal/...

# 2. 只跑内容守卫
go test ./internal/service -run 'TestContentGuard|TestRulesGroupIsLoaded' -v

# 3. 重构 GUI 二进制（事件随内嵌 FS 打包；改了 Go 代码必须重构）
wails build -skipbindings
#    注意：wails build 可能把 frontend/wailsjs/runtime/* 的文件模式改掉，提交前 git checkout -- 还原

# 4. 起服务 + 无头模拟（需要 GUI 环境，DISPLAY 可用）
rm -rf build/bin/events            # 清旧释放副本，确保加载仓库最新事件
SILO40_DEBUG_HTTP_ADDR=127.0.0.1:18080 ./build/bin/silo40 &
go run ./cmd/sim -addr http://127.0.0.1:18080 -games 20 -seed 7 -months 200
```

- **种子复现**：`-seed S` 时第 i 局种子 = `S+i`，行动策略也由 S 派生；同参数双跑必须逐局
  一致（不一致即引入了未受控随机源）。存档种子随 `Silo.RNGSeed` 持久化。
- **基线对照**：改动前后以同种子、同门控配置跑 sim 或 parity，比较结局分布与关键指标。
  当前确定性基线为 `references/parity-v7-three-silo.json`；旧版本结局分布不能直接套到
  当前三堡与门控实现，也不能把 ungated parity 当成生产平衡验收。
- **逐事件时序验证**：临时探针模板——`bootstrapRealContent(t)`（见
  `internal/service/rules_loading_test.go`）+ `CreateGame{Seed: 固定}` + 逐月 `PassTime`，
  断言 `svc.contentStates[key].TriggerCount/LastTriggeredTimestamp`。跑完删除，不入库。
- **结局分布看什么**：分布偏移、结局月份提前/推迟、终局人口均值大幅移动、叛乱接近 1.0、
  出现"必爆/从不触发"——任何一条都要在报告里给出前后对照。


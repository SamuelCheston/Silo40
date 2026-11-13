# Silo40 Linux 前端渲染性能：审计结论与修复记录

> 结论先行：Linux 卡顿根因是 Wails v2 缺省把 WebKitGTK 硬件加速策略置为 `NEVER`（本仓 `main.go` 未设 `options.Linux`），叠加前端大面积阴影/`transition: all`/canvas `shadowBlur` 在软件光栅下逐帧重光栅。已修：配置档打开合成（`OnDemand`）+ 前端档削减逐帧绘制负载。Windows/macOS 不受影响（见 §4）。
> 审计原始数据与完整报告：`~/.trae-cn/assistant/2026-11-12/silo40-frontend-perf/outputs/report.md`（含 work/runs/ 实测 trace）。
> 本文只记录结论、改动与复核数字；不含待落地设计。

## 1. 根因（证据链）

1. **渲染路径配置**：[main.go](../../main.go) 原先未设 `options.Linux` → Wails v2.13.0 `internal/frontend/desktop/linux/window.go:93-99` 缺省 `WebviewGpuPolicyNever` → `window.c:600-601` 设 `WEBKIT_HARDWARE_ACCELERATION_POLICY_NEVER`：WebKitGTK 关闭 accelerated compositor，滚动/动画走主线程逐帧 CPU 重光栅。Windows 走 WebView2（硬件加速默认开），故两端表现不同。
2. **前端逐帧负载**：`BunkerMap.css` 容器双层大阴影、`.bunker-class { transition: all }`；`BunkerMap.tsx` 卡片 hover `boxShadow: md` + `transition: all`；`HeaderStats.tsx` sticky 顶栏阴影；转盘库每帧 4 处 `shadowBlur`（底盘 24 / 每段文字 4 / 外环 10 / 按钮 16）。软件光栅下这些是每帧 blur 重算。
3. **实测因果**（真实 WebKitGTK 2.54.2，Broadway 虚拟显示，同口径交互+3s 滚动）：
   - 修复前：policy NEVER 时 WebKitWebProcess CPU **74 jiffies/5s**；ON_DEMAND 或关掉 CSS 效果均降至 **6**（12.3×）。
   - Chromium headless（SwiftShader）全程 59.9fps、forced reflow=0 → 前端 JS/布局本身不是瓶颈；掉帧特异性指向 WebKitGTK 非合成路径 + 每帧 blur。

## 2. 已实施改动

| 档 | 改动 | 位置 |
| --- | --- | --- |
| 配置 | `Linux: &linuxoptions.Options{WebviewGpuPolicy: WebviewGpuPolicyOnDemand}` | [main.go](../../main.go) |
| 前端 | 容器双层阴影 → 单层小 blur；删除 `.bunker-class` 的 `transition: all` 与死 CSS `.department-block` | [BunkerMap.css](../../frontend/src/components/BunkerMap.css) |
| 前端 | 卡片 hover 去 `boxShadow: md`、`transition: all` → `transform, border-color` 定向 | [BunkerMap.tsx](../../frontend/src/components/BunkerMap.tsx#L86-L87) |
| 前端 | sticky 顶栏去 `boxShadow="sm"`（保留 borderBottom 分隔） | [HeaderStats.tsx](../../frontend/src/components/HeaderStats.tsx#L23-L26) |
| 前端 | 转盘本地化 `PrizeWheelLocal`：阴影/光晕/按钮一次性预渲染进两张 offscreen canvas，每帧仅 2 次 drawImage（旋转 blit + 静态叠层）；视觉层级保留 | [PrizeWheelLocal.tsx](../../frontend/src/components/PrizeWheelLocal.tsx)、TimeWheel/SiloWheel 改导入 |
| 依赖 | 移除 `react-prize-wheel-advanced`（含其 d.ts），`package.json.md5` 同步重算 | frontend/package.json、package-lock.json、package.json.md5 |

未做（审计证明不需要）：列表虚拟化、状态管理重构、绑定批量化等架构级改动。

## 3. 修复后同口径复核（前后对比）

| 指标（同交互脚本） | 修复前 | 修复后 |
| --- | --- | --- |
| WebKitGTK NEVER 滚动+交互 web 进程 CPU | 74 jiffies/5s | **6**（−92%） |
| WebKitGTK NEVER 下"关效果"对照差值 | 74 vs 6（差 12.3×） | **6 vs 6（差值消失）** |
| WebKitGTK 转盘 1s spin CPU | 46 jiffies | **18**（−61%），无 >33ms 帧 |
| Chromium headless 交互帧节奏 | p50 16.7 / 59.9fps | p50 16.7 / 59.9fps（不变，原本即满帧） |
| Chromium 加载长任务 | 93.4 / 73.7 / 69.4 ms | 71.8 / 62.7 ms |
| Chromium paint / layout 总耗时 | 10.5 / 103 ms | 8.5 / 63.5 ms |

门禁：`npm run build`、`go vet ./...`、`go test -race ./...`、`wails build`、E2E sim（3 局全结局）全绿。

## 4. 跨平台影响面（最小验证）

- `GOOS=windows GOARCH=amd64 go build ./...` 通过：新增 import 与字段在 Windows 全量编译无影响。
- `GOOS=darwin GOARCH=arm64 go build github.com/wailsapp/wails/v2/pkg/options/linux` 通过：该包纯 Go、无 build 约束；`Linux` 字段唯一消费方在 `//go:build linux` 内（`internal/frontend/desktop/linux`），macOS 上为惰性数据。darwin 全量交叉编译本就不支持（Wails objc/cgo 后端），与本次改动无关。

## 5. 需要作者在自己桌面上 A/B 确认（唯一未验证项）

虚拟显示无法复现真实桌面的呈现链路（vsync/compositor feedback）。请在**实际卡顿的 Linux 桌面**上：

1. 基线复现：`git stash` 或 checkout 修复前 commit 构建运行，确认卡顿可复现（主观 + 观察交互时 CPU）。
2. 新构建直接运行：预期交互/滚动不再掉帧。
3. 若仍卡：启动器加 `WEBKIT_DISABLE_DMABUF_RENDERER=1` 再跑一次（NVIDIA 专有驱动 + Wayland 的已知呈现问题）；X11/XWayland 会话再试 `GDK_BACKEND=wayland` 覆盖 Wails 的 x11 强制。
4. 判定口径：交互（滚动地图/悬停卡片）时帧是否跟手；`top` 看 WebKitWebProcess 单核占用是否从"持续高"变为"脉冲式"。任一 env 变体生效即说明剩余问题在呈现链路而非本仓代码，把生效变量反馈即可。

package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	linuxoptions "github.com/wailsapp/wails/v2/pkg/options/linux"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed all:events
var events embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp(events)

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "Silo40",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		// Linux 缺省（options.Linux == nil）会被 Wails 置为 WebviewGpuPolicyNever，
		// WebKitGTK 关闭 accelerated compositor 后滚动/动画逐帧 CPU 重光栅（审计实测 12x 绘制成本）。
		// 显式 OnDemand 恢复合成路径；该字段仅 Linux 桌面后端读取，Windows/macOS 构建不受影响。
		Linux: &linuxoptions.Options{
			WebviewGpuPolicy: linuxoptions.WebviewGpuPolicyOnDemand,
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

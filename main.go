package main

import (
	"embed"
	"log"

	"github.com/narcilee7/aland/backend"
	"github.com/narcilee7/aland/backend/eye"
	"github.com/narcilee7/aland/backend/tribes"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app, err := backend.NewApp()
	if err != nil {
		log.Fatalf("new app: %v", err)
	}

	// 灵动岛菜单栏图标（macOS 优先，其他平台退化为 no-op）。
	// 必须在 goroutine 里启动——systray.Run 内部会调 [NSApp run]，
	// 与 Wails 主线程的 NSApp 复用同一个实例。
	tray := eye.New()
	tray.SetRunningGetter(func() []string {
		// 共享一份轻读——eyestate 的 Running 已经是 stable 的快照
		return app.GetEyeState().Running
	})
	app.SetTray(tray)
	go func() {
		tray.Run(
			func() {
				// Open Aland: 把主窗口拉到前台
				if app.GetWailsContext() != nil {
					runtime.Show(app.GetWailsContext())
					runtime.WindowUnminimise(app.GetWailsContext())
				}
			},
			func() {
				// Quit: 退出 Wails（systray.Quit 也会关闭事件循环）
				if app.GetWailsContext() != nil {
					runtime.Quit(app.GetWailsContext())
				}
			},
		)
	}()

	err = wails.Run(&options.App{
		Title:  "Aland",
		Width:  1280,
		Height: 820,
		// 极深的靛蓝到墨绿渐变（深夜大陆基调）
		BackgroundColour: &options.RGBA{R: 10, G: 14, B: 26, A: 1},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: app.Startup,
		Bind: []interface{}{
			app,
		},
		// macOS 专属：透明标题栏，融入地图的烧焦羊皮纸边框
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  true,
				FullSizeContent:            true,
			},
			Appearance:           mac.NSAppearanceNameDarkAqua,
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
		},
	})

	if err != nil {
		log.Fatalf("wails run: %v", err)
	}
	_ = tribes.StatusIdle // 保留 import 用于未来扩展
}
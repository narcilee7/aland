package backend

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/narcilee7/aland/backend/core"
	"github.com/narcilee7/aland/backend/infra"
	"github.com/narcilee7/aland/backend/tribes"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App 是 Wails 绑定到前端的根对象。
// 它把 Land / ProcManager / FSWatch 串起来，再把状态变化推给前端。
type App struct {
	ctx    context.Context
	Land   *core.Land
	Forge  *core.Forge
	Eye    *core.EyeState
	proc   *infra.ProcManager
	fs     *infra.FSWatch
	mu     sync.Mutex
	prev   map[string]int // 上一次扫描到的 PID，用于检测 born / death
}

// NewApp 构造一个未启动的 App。
func NewApp() *App {
	land := core.NewLand()
	tribes.RegisterAll(land)
	return &App{
		Land:  land,
		Forge: core.NewForge(),
		Eye:   core.NewEyeState(),
		prev:  make(map[string]int),
	}
}

// Startup Wails 启动回调（必须大写，wails.App.OnStartup 需要导出方法）。
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// 启动进程扫描
	a.proc = infra.NewProcManager(a.Land)
	a.proc.Start(ctx)

	// 启动文件监听（汇总所有部落的配置路径）
	var paths []string
	for _, t := range a.Land.Tribes {
		if ad := tribes.AsAdapter(t); ad != nil {
			paths = append(paths, ad.ConfigPaths()...)
		}
	}
	a.fs = infra.NewFSWatch(ctx, paths)
	_ = a.fs.Start()

	// 周期性把生命体征推给前端 + 检测 born/death
	go a.vitalLoop(ctx)
}

// vitalLoop 每秒把 snapshot 推到前端，并检测 born/death 事件。
func (a *App) vitalLoop(ctx context.Context) {
	t := time.NewTicker(1 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			snap := a.Land.Snapshot()
			runtime.EventsEmit(ctx, "tribe:vital", snap)
			a.detectBornDeath(ctx, snap)
		}
	}
}

func (a *App) detectBornDeath(ctx context.Context, snap map[string]core.Tribe) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for id, t := range snap {
		prev, ok := a.prev[id]
		cur := t.Vital.PID
		if !ok && cur > 0 {
			runtime.EventsEmit(ctx, "tribe:born", map[string]interface{}{
				"id":   id,
				"pid":  cur,
				"name": t.Name,
			})
		}
		if ok && prev > 0 && cur == 0 {
			runtime.EventsEmit(ctx, "tribe:death", map[string]interface{}{
				"id":   id,
				"pid":  prev,
				"name": t.Name,
			})
		}
		a.prev[id] = cur
	}
}

// ===== Wails 绑定方法（前端通过 window.go.main.App 调用） =====

// GetLand 返回大陆全貌。
func (a *App) GetLand() map[string]core.Tribe {
	return a.Land.Snapshot()
}

// GetTribe 返回单个部落。
func (a *App) GetTribe(id string) (core.Tribe, error) {
	t, ok := a.Land.Get(id)
	if !ok {
		return core.Tribe{}, fmt.Errorf("tribe not found: %s", id)
	}
	return core.Tribe{
		ID:     t.ID,
		Name:   t.Name,
		Eco:    t.Eco,
		Status: t.Status,
		Vital:  t.GetVital(),
	}, nil
}

// LaunchTribe 启动一个部落进程。
func (a *App) LaunchTribe(id, cwd string, args []string) error {
	t, ok := a.Land.Get(id)
	if !ok {
		return fmt.Errorf("tribe not found: %s", id)
	}
	ad := tribes.AsAdapter(t)
	if ad == nil {
		return fmt.Errorf("tribe %s has no adapter", id)
	}
	return ad.Launch(cwd, args)
}

// KillTribe 终止一个部落进程。
func (a *App) KillTribe(id string) error {
	t, ok := a.Land.Get(id)
	if !ok {
		return fmt.Errorf("tribe not found: %s", id)
	}
	ad := tribes.AsAdapter(t)
	if ad == nil {
		return fmt.Errorf("tribe %s has no adapter", id)
	}
	return ad.Kill(t.GetVital().PID)
}

// ReadTribeConfig 读取部落配置。
func (a *App) ReadTribeConfig(id string) (map[string]interface{}, error) {
	t, ok := a.Land.Get(id)
	if !ok {
		return nil, fmt.Errorf("tribe not found: %s", id)
	}
	ad := tribes.AsAdapter(t)
	if ad == nil {
		return nil, fmt.Errorf("tribe %s has no adapter", id)
	}
	return ad.ParseConfig()
}

// GetForge 读取 Token 熔炉状态（M0 占位）。
func (a *App) GetForge() core.Forge {
	return a.Forge.Snapshot()
}

// GetTribeMeta 返回部落元信息（主题色、生态等），前端用来渲染地形。
func (a *App) GetTribeMeta(id string) (map[string]string, error) {
	t, ok := a.Land.Get(id)
	if !ok {
		return nil, fmt.Errorf("tribe not found: %s", id)
	}
	ad := tribes.AsAdapter(t)
	if ad == nil {
		return nil, fmt.Errorf("tribe %s has no adapter", id)
	}
	return map[string]string{
		"id":          t.ID,
		"name":        t.Name,
		"eco":         t.Eco,
		"themeColor":  ad.ThemeColor(),
		"accentColor": ad.AccentColor(),
	}, nil
}

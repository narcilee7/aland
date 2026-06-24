// Package backend 是 Wails 绑定到前端的根层。
// 它把 tribes.Land / infra.ProcManager / infra.FSWatch 串起来，
// 并把状态变化通过 Wails Event 推给前端。
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

// App 是绑定到前端的根对象。
// 前端通过 window.go.main.App 调用其导出的方法。
type App struct {
	ctx context.Context

	land  *tribes.Land
	forge *core.Forge
	eye   *core.EyeState

	proc *infra.ProcManager
	fs   *infra.FSWatch

	mu   sync.Mutex
	prev map[string]int // 上次扫描到的 PID，用于检测 born / death
}

// NewApp 构造一个未启动的 App。
func NewApp() (*App, error) {
	land := tribes.NewLand()
	if err := tribes.RegisterAll(land); err != nil {
		return nil, fmt.Errorf("register tribes: %w", err)
	}
	return &App{
		land:  land,
		forge: core.NewForge(),
		eye:   core.NewEyeState(),
		prev:  make(map[string]int),
	}, nil
}

// Startup Wails 启动回调。
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// 启动进程扫描
	a.proc = infra.NewProcManager(a.land)
	a.proc.Start(ctx)

	// 启动文件监听（汇总所有部落 reader 的配置路径）
	paths := a.collectConfigPaths()
	a.fs = infra.NewFSWatch(ctx, paths)
	_ = a.fs.Start()

	// 周期把生命体征推给前端 + 检测 born/death
	go a.vitalLoop(ctx)
}

func (a *App) collectConfigPaths() []string {
	var paths []string
	for _, id := range a.land.IDs() {
		r, ok := a.land.Reader(id)
		if !ok {
			continue
		}
		paths = append(paths, r.ConfigPaths()...)
	}
	return paths
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
			snap := a.land.Snapshot()
			runtime.EventsEmit(ctx, "tribe:vital", snap)
			a.detectBornDeath(ctx, snap)
		}
	}
}

func (a *App) detectBornDeath(ctx context.Context, snap map[string]tribes.Tribe) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for id, t := range snap {
		prev, ok := a.prev[id]
		cur := t.Vital.PID
		if !ok && cur > 0 {
			runtime.EventsEmit(ctx, "tribe:born", map[string]any{
				"id":   id,
				"pid":  cur,
				"name": t.Meta.Name,
			})
		}
		if ok && prev > 0 && cur == 0 {
			runtime.EventsEmit(ctx, "tribe:death", map[string]any{
				"id":   id,
				"pid":  prev,
				"name": t.Meta.Name,
			})
		}
		a.prev[id] = cur
	}
}

// ===== Wails 绑定方法 =====

// GetLand 返回大陆全貌。
func (a *App) GetLand() map[string]tribes.Tribe {
	return a.land.Snapshot()
}

// GetTribe 返回单个部落。
func (a *App) GetTribe(id string) (tribes.Tribe, error) {
	t, ok := a.land.Get(id)
	if !ok {
		return tribes.Tribe{}, fmt.Errorf("tribe not found: %s", id)
	}
	return tribes.Tribe{
		Meta:   t.Meta,
		Status: t.Status,
		Vital:  t.GetVital(),
	}, nil
}

// LaunchTribe 启动一个部落进程。
func (a *App) LaunchTribe(id, cwd string, args []string) error {
	la, ok := a.land.Launcher(id)
	if !ok {
		return fmt.Errorf("tribe %s has no launcher", id)
	}
	return la.Launch(cwd, args)
}

// KillTribe 终止一个部落进程。
func (a *App) KillTribe(id string) error {
	t, ok := a.land.Get(id)
	if !ok {
		return fmt.Errorf("tribe not found: %s", id)
	}
	la, ok := a.land.Launcher(id)
	if !ok {
		return fmt.Errorf("tribe %s has no launcher", id)
	}
	return la.Kill(t.GetVital().PID)
}

// ReadTribeConfig 读取部落配置。
func (a *App) ReadTribeConfig(id string) (map[string]any, error) {
	r, ok := a.land.Reader(id)
	if !ok {
		return nil, fmt.Errorf("tribe %s has no reader", id)
	}
	return r.ParseConfig()
}

// GetForge 读取 Token 熔炉状态（M0 占位）。
func (a *App) GetForge() core.Forge {
	return a.forge.Snapshot()
}

// GetTribeMeta 返回部落元信息（前端用来渲染地形）。
func (a *App) GetTribeMeta(id string) (tribes.Meta, error) {
	t, ok := a.land.Get(id)
	if !ok {
		return tribes.Meta{}, fmt.Errorf("tribe not found: %s", id)
	}
	return t.Meta, nil
}

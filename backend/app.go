// Package backend 是 Wails 绑定到前端的根层。
// 它把 tribes.Land / infra.ProcManager / infra.FSWatch 串起来，
// 并把状态变化通过 events.Emitter 推给前端。
package backend

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/narcilee7/aland/backend/core"
	"github.com/narcilee7/aland/backend/events"
	"github.com/narcilee7/aland/backend/hotkey"
	"github.com/narcilee7/aland/backend/infra"
	"github.com/narcilee7/aland/backend/tribes"
)

// App 是绑定到前端的根对象。
// 前端通过 window.go.main.App 调用其导出的方法。
type App struct {
	ctx context.Context
	em  *events.Emitter

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
	a.em = events.New(ctx)

	core.Log.Info("aland starting", "tribes", a.land.IDs())

	// 启动进程扫描
	a.proc = infra.NewProcManager(a.land)
	a.proc.Start(ctx)

	// 启动文件监听（汇总所有部落 reader 的配置路径）
	paths := a.collectConfigPaths()
	a.fs = infra.NewFSWatch(ctx, paths)
	_ = a.fs.Start()

	// 注册全局快捷键 Cmd+Shift+A → 唤起 Spotlight
	if err := hotkey.CmdShiftA(ctx, a.em); err != nil {
		core.Log.Warn("hotkey register failed", "err", err)
	}

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
			a.em.EmitTribeVital(snap)
			a.detectBornDeath(snap)
			a.refreshEye(snap)
		}
	}
}

// refreshEye 把部落 vital 投影成 TribeVitalInput 喂给 Eye 状态机。
// 模式或 Running 变化时 EmitEyeUpdate，前端订阅即拿到最新。
func (a *App) refreshEye(snap map[string]tribes.Tribe) {
	inputs := make([]core.TribeVitalInput, 0, len(snap))
	for _, t := range snap {
		inputs = append(inputs, core.TribeVitalInput{
			ID:     t.Meta.ID,
			Status: string(t.Status),
			PID:    t.Vital.PID,
			CPU:    t.Vital.CPU,
		})
	}
	if !a.eye.Recompute(inputs) {
		return
	}
	snap2 := a.eye.Snapshot()
	a.em.EmitEyeUpdate(events.EyeUpdateEvent{
		Mode:      snap2.Mode,
		Running:   snap2.Running,
		UpdatedAt: snap2.UpdatedAt,
	})
}

func (a *App) detectBornDeath(snap map[string]tribes.Tribe) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for id, t := range snap {
		prev, ok := a.prev[id]
		cur := t.Vital.PID
		if !ok && cur > 0 {
			a.em.EmitTribeBorn(id, cur, t.Meta.Name)
			flash := a.eye.PushFlash(core.FlashBorn, id, fmt.Sprintf("%s 启动 · pid %d", t.Meta.Name, cur))
			a.em.EmitEyeFlash(flash)
		}
		if ok && prev > 0 && cur == 0 {
			a.em.EmitTribeDeath(id, prev, t.Meta.Name)
			flash := a.eye.PushFlash(core.FlashDeath, id, fmt.Sprintf("%s 已停止", t.Meta.Name))
			a.em.EmitEyeFlash(flash)
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
	core.Log.Info("launch tribe", "tribe", id, "cwd", cwd)
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
	pid := t.GetVital().PID
	core.Log.Info("kill tribe", "tribe", id, "pid", pid)
	return la.Kill(pid)
}

// ReadTribeConfig 读取部落配置。
func (a *App) ReadTribeConfig(id string) (map[string]any, error) {
	r, ok := a.land.Reader(id)
	if !ok {
		return nil, fmt.Errorf("tribe %s has no reader", id)
	}
	return r.ParseConfig()
}

// ReadTribeConfigDNA 读取三层结构化配置。
func (a *App) ReadTribeConfigDNA(id string) (tribes.ConfigDNA, error) {
	cp, ok := a.land.ConfigParse(id)
	if !ok {
		return tribes.ConfigDNA{}, fmt.Errorf("tribe %s has no config parser", id)
	}
	return cp.ParseConfigDNA()
}

// GetForge 读取 Token 熔炉状态。
// 实现：聚合所有部落有 TokenStatReader 能力的输出。
func (a *App) GetForge() core.Forge {
	byTribe := map[string]int64{}
	byModel := map[string]int64{}
	var total int64
	for _, id := range a.land.IDs() {
		ts, ok := a.land.TokenStat(id)
		if !ok {
			continue
		}
		usage, err := ts.ParseTokenUsage()
		if err != nil || usage == nil {
			continue
		}
		sum := usage.InputTokens + usage.OutputTokens
		byTribe[id] = sum
		total += sum
		if usage.Model != "" {
			byModel[usage.Model] += sum
		}
	}
	return core.Forge{
		TodaySpent:  total,
		ByTribe:     byTribe,
		ByModel:     byModel,
		DailyBudget: a.forge.DailyBudget,
	}
}

// ListSessions 列出某部落的全部会话（记忆碎片）。
func (a *App) ListSessions(id string) ([]tribes.SessionShard, error) {
	sl, ok := a.land.Sessions(id)
	if !ok {
		return nil, fmt.Errorf("tribe %s has no session lister", id)
	}
	return sl.ParseSessions()
}

// ReadSession 读取某 session 的完整事件流。
func (a *App) ReadSession(id, sessionID string) ([]tribes.SessionEvent, error) {
	sr, ok := a.land.SessionRead(id)
	if !ok {
		return nil, fmt.Errorf("tribe %s has no session reader", id)
	}
	return sr.ReadSession(sessionID)
}

// ListMCPServers 列出某部落的 MCP servers。
func (a *App) ListMCPServers(id string) ([]tribes.MCPServer, error) {
	m, ok := a.land.MCPServers(id)
	if !ok {
		return nil, fmt.Errorf("tribe %s has no mcp lister", id)
	}
	return m.ListMCPServers()
}

// ListSkills 列出 skills。
func (a *App) ListSkills(id string) ([]tribes.Skill, error) {
	s, ok := a.land.Skills(id)
	if !ok {
		return nil, fmt.Errorf("tribe %s has no skills lister", id)
	}
	return s.ListSkills()
}

// ListPlans 列出 plan files。
func (a *App) ListPlans(id string) ([]tribes.PlanFile, error) {
	p, ok := a.land.Plans(id)
	if !ok {
		return nil, fmt.Errorf("tribe %s has no plans lister", id)
	}
	return p.ListPlans()
}

// ListFileHistory 列出 file history。
func (a *App) ListFileHistory(id string) ([]tribes.FileEdit, error) {
	fh, ok := a.land.FileHistory(id)
	if !ok {
		return nil, fmt.Errorf("tribe %s has no file history", id)
	}
	return fh.ListFileHistory()
}

// RestoreFile 恢复某 file edit 的 backup 版本。
func (a *App) RestoreFile(id string, edit tribes.FileEdit) error {
	fh, ok := a.land.FileHistory(id)
	if !ok {
		return fmt.Errorf("tribe %s has no file history", id)
	}
	return fh.RestoreFile(edit)
}

// ListDailyActivity 列出每日活跃。
func (a *App) ListDailyActivity(id string) ([]tribes.DailyActivity, error) {
	s, ok := a.land.Stats(id)
	if !ok {
		return nil, fmt.Errorf("tribe %s has no stats", id)
	}
	return s.DailyActivity()
}

// ListModelTokenUsage 按 model 列出 token 消耗。
func (a *App) ListModelTokenUsage(id string) ([]tribes.ModelTokenUsage, error) {
	s, ok := a.land.Stats(id)
	if !ok {
		return nil, fmt.Errorf("tribe %s has no stats", id)
	}
	return s.ModelTokenUsage()
}

// RecentSlashCommands 最近 n 条 slash command 历史。
func (a *App) RecentSlashCommands(id string, n int) ([]tribes.SlashCommand, error) {
	s, ok := a.land.Stats(id)
	if !ok {
		return nil, fmt.Errorf("tribe %s has no stats", id)
	}
	return s.RecentSlashCommands(n)
}

// ListPlugins 列出启用的 plugins。
func (a *App) ListPlugins(id string) ([]tribes.Plugin, error) {
	s, ok := a.land.Stats(id)
	if !ok {
		return nil, fmt.Errorf("tribe %s has no stats", id)
	}
	return s.ListPlugins()
}

// StreamLatestSession 启动 session 实时 tail。
// 事件通过 events.SESSION_EVENT 推到前端。
func (a *App) StreamLatestSession(id string) error {
	ss, ok := a.land.SessionStream(id)
	if !ok {
		return fmt.Errorf("tribe %s has no session streamer", id)
	}
	cb := func(ev tribes.SessionEvent) {
		a.em.Emit(events.SessionEvent, ev)
	}
	return ss.StreamLatest(a.ctx, cb)
}

// StopLatestSession 停止 tail。
func (a *App) StopLatestSession(id string) {
	ss, ok := a.land.SessionStream(id)
	if !ok {
		return
	}
	ss.StopStream()
}

// GetTribeMeta 返回部落元信息（前端用来渲染地形）。
func (a *App) GetTribeMeta(id string) (tribes.Meta, error) {
	t, ok := a.land.Get(id)
	if !ok {
		return tribes.Meta{}, fmt.Errorf("tribe not found: %s", id)
	}
	return t.Meta, nil
}

// GetTribeCapabilities 返回部落声明的能力清单。
// 前端用这个决定渲染哪些面板（能力感知 UI）。
func (a *App) GetTribeCapabilities(id string) (tribes.Capabilities, error) {
	c, ok := a.land.Capabilities(id)
	if !ok {
		return tribes.Capabilities{}, fmt.Errorf("tribe not found: %s", id)
	}
	return c, nil
}

// GetAllCapabilities 一次性拿所有部落的能力（产品视角矩阵）。
func (a *App) GetAllCapabilities() map[string]tribes.Capabilities {
	return a.land.AllCapabilities()
}

// WriteTribeConfig 写回配置。会自动备份到 ~/.aland/backups/。
func (a *App) WriteTribeConfig(id string, dna tribes.ConfigDNA) error {
	w, ok := a.land.Writer(id)
	if !ok {
		return fmt.Errorf("tribe %s has no config writer", id)
	}
	core.Log.Info("write tribe config", "tribe", id, "surface", len(dna.Surface), "middle", len(dna.Middle), "deep", len(dna.Deep))
	return w.WriteConfig(dna)
}

// ===== Eye 灵动岛 =====

// GetEyeState 返回当前 Eye 完整快照。
// 前端在挂载时调用一次拿到初始 Mode/Running/Flashing，
// 之后通过订阅 eye:update / eye:flash 增量更新。
func (a *App) GetEyeState() core.EyeState {
	return a.eye.Snapshot()
}

// ConsumeEyeFlash 把指定 flash 标记为已读（从队列移除）。
// 前端在用户点击/关闭一条通知时调用。
func (a *App) ConsumeEyeFlash(id string) bool {
	return a.eye.ConsumeFlash(id)
}

// ClearEyeFlashes 清空全部 flash（前端"全部已读"按钮）。
func (a *App) ClearEyeFlashes() {
	a.eye.ClearFlashes()
}

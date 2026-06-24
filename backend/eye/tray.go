// Package eye 是 macOS 菜单栏灵动岛的实现。
//
// Sprint 1 用 fyne.io/systray 做"系统托盘"——但 macOS 上 systray 实际是 NSStatusItem。
// 4 状态（dormant/active/storm/alert）对应 4 个图标；flash 触发时临时切换。
//
// 关于 Wails + systray 的进程模型：
//   - Wails 启动时把 NSApp 跑在主线程
//   - systray.Run() 也会调 NSApp.run()——所以必须在 goroutine 里启动，
//     让 Wails 先拿到主线程
//   - 实际上 macOS 上 NSApp 是单例，第二次 [NSApp run] 通常立即返回，
//     不会冲突；菜单栏图标会被加到当前 NSApp 上
//
// Sprint 1 只做 macOS。Windows/Linux 推迟到 Sprint 2。
package eye

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"sync"
	"time"

	"fyne.io/systray"

	"github.com/narcilee7/aland/backend/core"
)

// onClick 菜单项点击回调类型。
type onClick func()

// Tray 灵动岛菜单栏图标。
type Tray struct {
	mu         sync.Mutex
	current    core.EyeStateMode
	flashUntil time.Time

	// runningMenu 周期性刷新的"Running tribes"菜单项。
	runningMenu *systray.MenuItem
	// runningGetter 由调用方注入，返回当前活跃 tribe 列表。
	runningGetter func() []string

	// 启动后置 true，systray.OnReady 已经触发。
	readyMu sync.Mutex
	ready   bool
}

// New 创建一个未启动的 Tray。
func New() *Tray {
	return &Tray{current: core.DormantMode}
}

// SetRunningGetter 注入一个回调，定期把"当前活跃 tribe 列表"刷到菜单项。
// 必须在 Run() 之前调用。
func (t *Tray) SetRunningGetter(fn func() []string) {
	t.runningGetter = fn
}

// Run 启动菜单栏图标的事件循环。**会阻塞**当前 goroutine。
//
// onOpen —— 菜单"Open Aland"点击时调用（通常用来 Show 主窗口）。
// onQuit —— 菜单"Quit"点击时调用（通常用来 runtime.Quit）。
func (t *Tray) Run(onOpen, onQuit onClick) {
	systray.Run(
		func() { // onReady
			t.setup(onOpen, onQuit)
		},
		func() { // onExit
			t.readyMu.Lock()
			t.ready = false
			t.readyMu.Unlock()
		},
	)
}

// setup 在 systray.OnReady 回调里初始化图标和菜单。
func (t *Tray) setup(onOpen, onQuit onClick) {
	systray.SetIcon(iconForMode(core.DormantMode))
	systray.SetTitle("") // macOS 上如果设了 Title 会跟 Icon 叠加，先空
	systray.SetTooltip("Aland · Agent Land")

	mOpen := systray.AddMenuItem("Open Aland", "Show main window")
	mRunning := systray.AddMenuItem("Running: —", "Currently running tribes")
	mRunning.Disable()
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Exit Aland")

	t.readyMu.Lock()
	t.ready = true
	t.runningMenu = mRunning
	t.readyMu.Unlock()

	// 监听点击
	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				if onOpen != nil {
					onOpen()
				}
			case <-mQuit.ClickedCh:
				if onQuit != nil {
					onQuit()
				}
				systray.Quit()
				return
			}
		}
	}()

	// 周期性刷新 Running 列表 + tooltip
	go t.refreshLoop()
}

// refreshLoop 每秒更新一次菜单项。
func (t *Tray) refreshLoop() {
	tk := time.NewTicker(1 * time.Second)
	defer tk.Stop()
	for range tk.C {
		t.readyMu.Lock()
		ready := t.ready
		menu := t.runningMenu
		t.readyMu.Unlock()
		if !ready || menu == nil {
			continue
		}
		if t.runningGetter != nil {
			running := t.runningGetter()
			if len(running) == 0 {
				menu.SetTitle("Running: —")
			} else {
				menu.SetTitle("Running: " + joinComma(running))
			}
		}
	}
}

// SetMode 切换图标。线程安全，可任意调用。
func (t *Tray) SetMode(mode core.EyeStateMode) {
	t.mu.Lock()
	t.current = mode
	// 清除 flash 状态
	t.flashUntil = time.Time{}
	t.mu.Unlock()

	t.readyMu.Lock()
	ready := t.ready
	t.readyMu.Unlock()
	if !ready {
		return
	}
	systray.SetIcon(iconForMode(mode))
	systray.SetTooltip(tooltipForMode(mode))
}

// ShowFlash 临时切换图标为 flash 类型 X 毫秒，然后回到当前 mode。
// 用于"事件发生→短动画→恢复"效果。
func (t *Tray) ShowFlash(typ core.FlashType, ms int) {
	if ms <= 0 {
		ms = 800
	}
	t.readyMu.Lock()
	ready := t.ready
	t.readyMu.Unlock()
	if !ready {
		return
	}
	t.mu.Lock()
	t.flashUntil = time.Now().Add(time.Duration(ms) * time.Millisecond)
	t.mu.Unlock()

	systray.SetIcon(iconForFlash(typ))
	time.AfterFunc(time.Duration(ms)*time.Millisecond, func() {
		t.mu.Lock()
		expired := time.Now().After(t.flashUntil)
		mode := t.current
		t.mu.Unlock()
		if expired {
			systray.SetIcon(iconForMode(mode))
		}
	})
}

func joinComma(xs []string) string {
	out := ""
	for i, x := range xs {
		if i > 0 {
			out += ", "
		}
		out += x
	}
	return out
}

// —— 图标生成 ——

// iconForMode 根据 mode 生成 22x22 模板图标（黑/透明，macOS menu bar 标准）。
func iconForMode(mode core.EyeStateMode) []byte {
	const sz = 22
	img := image.NewRGBA(image.Rect(0, 0, sz, sz))
	// 透明背景
	for y := 0; y < sz; y++ {
		for x := 0; x < sz; x++ {
			img.Set(x, y, color.RGBA{0, 0, 0, 0})
		}
	}
	black := color.RGBA{0, 0, 0, 255}

	switch mode {
	case core.DormantMode:
		// 空心圆：岛屿沉睡
		drawCircleOutline(img, sz/2, sz/2, 8, black, 2)
	case core.ActiveMode:
		// 实心圆：部落活跃
		drawCircleFilled(img, sz/2, sz/2, 8, black)
	case core.StormMode:
		// 实心三角：风暴
		drawTriangleFilled(img, sz, black)
	case core.AlertMode:
		// 感叹号：告警
		drawExclamation(img, sz/2, sz/2, 7, black)
	}
	return encodePNG(img)
}

// iconForFlash 根据 flash 类型生成图标——用同套几何但稍大表示"事件"。
func iconForFlash(typ core.FlashType) []byte {
	const sz = 22
	img := image.NewRGBA(image.Rect(0, 0, sz, sz))
	for y := 0; y < sz; y++ {
		for x := 0; x < sz; x++ {
			img.Set(x, y, color.RGBA{0, 0, 0, 0})
		}
	}
	black := color.RGBA{0, 0, 0, 255}

	switch typ {
	case core.FlashComplete:
		// 实心圆 + 内圈亮色 = "完成"
		drawCircleFilled(img, sz/2, sz/2, 8, black)
		drawCircleFilled(img, sz/2, sz/2, 4, color.RGBA{0, 0, 0, 0})
	case core.FlashBorn:
		// 上箭头：进程启动
		drawUpArrow(img, sz, black)
	case core.FlashDeath:
		// 下箭头：进程停止
		drawDownArrow(img, sz, black)
	case core.FlashError:
		// X：报错
		drawX(img, sz, black, 6)
	case core.FlashCostAlert:
		// $ 字符太复杂，用三角形代替"警告"
		drawTriangleFilled(img, sz, black)
	case core.FlashConflict:
		// 双竖线：冲突
		drawConflict(img, sz, black)
	}
	return encodePNG(img)
}

func tooltipForMode(mode core.EyeStateMode) string {
	switch mode {
	case core.DormantMode:
		return "Aland · 沉睡守望"
	case core.ActiveMode:
		return "Aland · 部落活跃"
	case core.StormMode:
		return "Aland · 风暴（高负载）"
	case core.AlertMode:
		return "Aland · 告警"
	}
	return "Aland"
}

// encodePNG 把 RGBA 图编码成 PNG 字节流。
func encodePNG(img *image.RGBA) []byte {
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// —— 基础绘图 ——

// drawCircleFilled 画实心圆。
func drawCircleFilled(img *image.RGBA, cx, cy, r int, c color.Color) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r*r {
				img.Set(cx+x, cy+y, c)
			}
		}
	}
}

// drawCircleOutline 画空心圆（指定线宽）。
func drawCircleOutline(img *image.RGBA, cx, cy, r int, c color.Color, thick int) {
	outer := r * r
	inner := (r - thick) * (r - thick)
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			d := x*x + y*y
			if d <= outer && d >= inner {
				img.Set(cx+x, cy+y, c)
			}
		}
	}
}

// drawTriangleFilled 画实心等腰三角形（顶点朝上）。
func drawTriangleFilled(img *image.RGBA, sz int, c color.Color) {
	for y := 0; y < sz-3; y++ {
		halfWidth := y * (sz - 4) / (2 * (sz - 4))
		if halfWidth == 0 && y > 0 {
			halfWidth = 1
		}
		for x := sz/2 - halfWidth; x <= sz/2+halfWidth; x++ {
			if x >= 0 && x < sz {
				img.Set(x, y+2, c)
			}
		}
	}
}

// drawExclamation 画感叹号。
func drawExclamation(img *image.RGBA, cx, cy, r int, c color.Color) {
	// 上竖线
	for y := -r; y <= r/3; y++ {
		for x := -1; x <= 1; x++ {
			img.Set(cx+x, cy+y, c)
		}
	}
	// 下点
	img.Set(cx, cy+r-1, c)
	img.Set(cx-1, cy+r-1, c)
	img.Set(cx+1, cy+r-1, c)
}

// drawX 画 X 形（中心对称）。
func drawX(img *image.RGBA, sz int, c color.Color, halfLen int) {
	for i := -halfLen; i <= halfLen; i++ {
		img.Set(sz/2+i, sz/2+i, c)
		img.Set(sz/2+i, sz/2-i, c)
	}
}

// drawUpArrow 画向上的箭头（三角形 + 竖线）。
func drawUpArrow(img *image.RGBA, sz int, c color.Color) {
	// 三角顶点
	for y := 0; y < sz/2; y++ {
		halfW := y
		if halfW == 0 {
			halfW = 1
		}
		for x := sz/2 - halfW; x <= sz/2+halfW; x++ {
			img.Set(x, y+2, c)
		}
	}
	// 竖线
	for y := sz / 2; y < sz-2; y++ {
		img.Set(sz/2, y, c)
	}
}

// drawDownArrow 画向下的箭头。
func drawDownArrow(img *image.RGBA, sz int, c color.Color) {
	for y := 0; y < sz/2; y++ {
		img.Set(sz/2, y+2, c)
	}
	for y := 0; y < sz/2; y++ {
		halfW := sz/2 - y - 1
		for x := sz/2 - halfW; x <= sz/2+halfW; x++ {
			img.Set(x, sz-y-3, c)
		}
	}
}

// drawConflict 画双竖线（"冲突"）。
func drawConflict(img *image.RGBA, sz int, c color.Color) {
	for y := 3; y < sz-3; y++ {
		img.Set(sz/2-3, y, c)
		img.Set(sz/2+3, y, c)
	}
}
// Package core: 灵动岛（Eye）的状态机。
//
// Eye 是 Aland 的"系统级守望者"——不打开主窗口也能感知 AI CLI 的状态。
// 两个数据维度：
//   - Mode   持续状态（dormant/active/storm/alert），由 Recompute 计算
//   - Flash  瞬时通知（complete/cost_alert/error/conflict），FIFO 队列
//
// 消费方：
//   - app.go vitalLoop 每秒调 Recompute，模式变化时 EmitEyeUpdate
//   - session 完成 / 预算超阈值等触发点调 PushFlash + EmitEyeFlash
//   - 前端通过 events.Emitter 订阅 eye:update / eye:flash
package core

import (
	"sync"
	"time"
)

// EyeStateMode 灵动岛当前模式。
//
// 模式定义：
//   - dormant: 没有任何 CLI 跑——岛屿沉睡
//   - active:  有 CLI 在跑——岛屿"醒着"，显示呼吸光
//   - storm:   多个 CLI 高负载——岛屿有"风暴漩涡"
//   - alert:   出错了（预算超、文件冲突、session 失败）——岛屿闪红
type EyeStateMode string

const (
	DormantMode EyeStateMode = "dormant"
	ActiveMode  EyeStateMode = "active"
	StormMode   EyeStateMode = "storm"
	AlertMode   EyeStateMode = "alert"
)

// FlashType flash 通知类型。
//
// 用于一次性的"事件广播"——session 完成、预算告警、文件冲突等。
// 跟 Mode（持续状态）不同，Flash 是瞬时消费品。
type FlashType string

const (
	FlashComplete  FlashType = "complete"   // session / 任务完成
	FlashCostAlert FlashType = "cost_alert" // 预算超阈值
	FlashError     FlashType = "error"      // CLI 报错
	FlashConflict  FlashType = "conflict"   // 多 CLI 改同一文件
)

// Flash 单条灵动岛通知。
//
// 通知从 PushFlash 入队，前端通过 eye:flash 实时收到；
// ConsumeFlash 把它从队列移除（用户已读）。
type Flash struct {
	ID        string    `json:"id"`        // 唯一 ID，前端用来去重
	Type      FlashType `json:"type"`
	Tribe     string    `json:"tribe"`     // 关联部落 ID（可空）
	Content   string    `json:"content"`   // 显示文本
	CreatedAt int64     `json:"createdAt"` // unix ms
}

// TribeVitalInput 是 Recompute 的最小输入 DTO。
//
// 定义在 core 而不是直接吃 tribes.Tribe，是为了避免 import cycle：
// core 被 tribes 反向依赖（tribes 用 core.Log），所以 core 不能 import tribes。
// 调用方（app.go vitalLoop）从 tribes.Tribe 投影出这个 DTO 即可。
type TribeVitalInput struct {
	ID     string  // 部落 ID
	Status string  // "idle" | "running" | "busy" | "error"
	PID    int     // 0 表示没在跑
	CPU    float64 // 0-100（多核可能 > 100）
}

// maxFlashing 队列上限。超过时丢弃最旧的，避免无限增长。
const maxFlashing = 16

// EyeState 灵动岛（系统级悬浮胶囊）的状态。
//
// 所有修改走 mutex，前端通过 Snapshot() 拿一致快照。
// Sprint 1 把状态机从占位升级为真实结构：Mode 由 Recompute 计算，Flashing 是 flash 队列。
type EyeState struct {
	mu        sync.Mutex
	Mode      EyeStateMode `json:"mode"`
	Running   []string     `json:"running"`  // 当前运行的部落 ID
	Flashing  []Flash      `json:"flashing"` // 队列里的通知
	UpdatedAt int64        `json:"updatedAt"`
}

// NewEyeState 创建一个沉睡的灵动岛。
func NewEyeState() *EyeState {
	return &EyeState{
		Mode:      DormantMode,
		Running:   []string{},
		Flashing:  []Flash{},
		UpdatedAt: time.Now().UnixMilli(),
	}
}

// Snapshot 返回一份深拷贝快照，前端读取安全。
func (e *EyeState) Snapshot() EyeState {
	e.mu.Lock()
	defer e.mu.Unlock()
	return EyeState{
		Mode:      e.Mode,
		Running:   append([]string(nil), e.Running...),
		Flashing:  append([]Flash(nil), e.Flashing...),
		UpdatedAt: e.UpdatedAt,
	}
}

// CurrentMode 仅返回模式（廉价读，用于高频调用路径）。
func (e *EyeState) CurrentMode() EyeStateMode {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.Mode
}

// Recompute 根据部落 vital 计算新 Mode 和 Running。
//
// 规则（从最严重往下走，第一个匹配即返回）：
//   1. 有任何部落 error 状态 → alert
//   2. 总 CPU > 150（多核）或活跃数 > 2 且总 CPU > 80 → storm
//   3. 有部落 Running/Busy 状态 → active
//   4. 都没有 → dormant
//
// 返回值表示 Mode 或 Running 是否变化，调用方据此决定是否 EmitEyeUpdate。
func (e *EyeState) Recompute(items []TribeVitalInput) bool {
	var running []string
	var totalCPU float64
	var hasError bool
	for _, t := range items {
		if t.Status == "error" {
			hasError = true
		}
		if t.PID > 0 || t.Status == "running" || t.Status == "busy" {
			running = append(running, t.ID)
		}
		totalCPU += t.CPU
	}

	var newMode EyeStateMode
	switch {
	case hasError:
		newMode = AlertMode
	case totalCPU > 150 || (len(running) > 2 && totalCPU > 80):
		newMode = StormMode
	case len(running) > 0:
		newMode = ActiveMode
	default:
		newMode = DormantMode
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	changed := newMode != e.Mode || !sliceEq(e.Running, running)
	e.Mode = newMode
	e.Running = running
	e.UpdatedAt = time.Now().UnixMilli()
	return changed
}

// PushFlash 入队一条 flash；如果超过上限则丢弃最旧的。
// 返回被推入的 flash（带 ID），调用方可以 EmitEyeFlash。
func (e *EyeState) PushFlash(t FlashType, tribe, content string) Flash {
	e.mu.Lock()
	defer e.mu.Unlock()
	f := Flash{
		ID:        newFlashID(),
		Type:      t,
		Tribe:     tribe,
		Content:   content,
		CreatedAt: time.Now().UnixMilli(),
	}
	e.Flashing = append(e.Flashing, f)
	if len(e.Flashing) > maxFlashing {
		// 保留最新的 maxFlashing 条
		e.Flashing = e.Flashing[len(e.Flashing)-maxFlashing:]
	}
	e.UpdatedAt = time.Now().UnixMilli()
	return f
}

// ConsumeFlash 把指定 ID 的 flash 从队列移除（标记已读）。
// 找不到返回 false。
func (e *EyeState) ConsumeFlash(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i, f := range e.Flashing {
		if f.ID == id {
			e.Flashing = append(e.Flashing[:i], e.Flashing[i+1:]...)
			e.UpdatedAt = time.Now().UnixMilli()
			return true
		}
	}
	return false
}

// ClearFlashes 清空队列（重置按钮用）。
func (e *EyeState) ClearFlashes() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Flashing = []Flash{}
	e.UpdatedAt = time.Now().UnixMilli()
}

// flashCounter 进程内单调递增，配合时间戳生成唯一 ID。
var flashCounter uint64

func newFlashID() string {
	n := flashCounter
	flashCounter++
	return time.Now().Format("20060102150405.000") + "-" + itoa(n)
}

func itoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func sliceEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
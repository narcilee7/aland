package core

import (
	"sync"
	"time"
)

// TribeStatus 大陆对部落状态的抽象。
type TribeStatus string

const (
	StatusIdle   TribeStatus = "idle"   // 闲置
	StatusRun    TribeStatus = "running" // 正在运行
	StatusBusy   TribeStatus = "busy"   // 高负载
	StatusError  TribeStatus = "error"  // 异常
)

// VitalSign 一个部落的"生命体征"。
// 前端用它驱动呼吸、光束、篝火、帆——所有"生命仪式"的源头。
type VitalSign struct {
	PID       int     `json:"pid"`
	CPU       float64 `json:"cpu"`    // 0-100
	Memory    float64 `json:"memory"` // MB
	CWD       string  `json:"cwd"`
	Uptime    int64   `json:"uptime"`  // 秒
	Model     string  `json:"model"`
	UpdatedAt int64   `json:"updatedAt"` // unix ms
}

// Tribe 大陆上的一个部落。
// Adapter 字段是 any 以避免 core 与 tribes 之间的循环依赖。
// 调用方需要时类型断言为 tribes.TribeAdapter。
type Tribe struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Eco     string      `json:"eco"` // 生态类型：classical / modern / oriental ...
	Status  TribeStatus `json:"status"`
	Vital   VitalSign   `json:"vital"`
	Adapter any         `json:"-"`
	mu      sync.RWMutex
}

// SetVital 线程安全地更新生命体征。
func (t *Tribe) SetVital(v VitalSign) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Vital = v
	t.Vital.UpdatedAt = time.Now().UnixMilli()
	if v.PID > 0 {
		t.Status = StatusRun
		if v.CPU > 70 {
			t.Status = StatusBusy
		}
	} else {
		t.Status = StatusIdle
	}
}

// GetVital 读取当前生命体征。
func (t *Tribe) GetVital() VitalSign {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Vital
}

// Land 大陆——所有部落的容器。
// 它是单例，整个 App 共享一片大陆。
type Land struct {
	mu     sync.RWMutex
	Tribes map[string]*Tribe
}

// NewLand 创建一片空的大陆。
func NewLand() *Land {
	return &Land{
		Tribes: make(map[string]*Tribe),
	}
}

// Register 把一个部落（适配器）登记到大陆。
// 之后由 Registry 在 App 启动时统一调用。
func (l *Land) Register(t *Tribe) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.Tribes[t.ID] = t
}

// Get 读取一个部落。
func (l *Land) Get(id string) (*Tribe, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	t, ok := l.Tribes[id]
	return t, ok
}

// Snapshot 大陆全貌——前端俯瞰时拿这个。
func (l *Land) Snapshot() map[string]Tribe {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make(map[string]Tribe, len(l.Tribes))
	for id, t := range l.Tribes {
		t.mu.RLock()
		out[id] = Tribe{
			ID:     t.ID,
			Name:   t.Name,
			Eco:    t.Eco,
			Status: t.Status,
			Vital:  t.Vital,
		}
		t.mu.RUnlock()
	}
	return out
}

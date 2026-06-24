package tribes

import (
	"sync"
	"time"
)

// Status 部落运行状态。
type Status string

const (
	StatusIdle  Status = "idle"   // 闲置
	StatusRun   Status = "running" // 正常运行
	StatusBusy  Status = "busy"   // 高负载
	StatusError Status = "error"  // 异常
)

// VitalSign 生命体征。前端用它驱动呼吸、光束、篝火、帆——所有"生命仪式"的源头。
type VitalSign struct {
	PID       int     `json:"pid"`
	CPU       float64 `json:"cpu"`    // 0-100
	Memory    float64 `json:"memory"` // MB
	CWD       string  `json:"cwd"`
	Uptime    int64   `json:"uptime"` // 秒
	Model     string  `json:"model"`
	UpdatedAt int64   `json:"updatedAt"` // unix ms
}

// Tribe 大陆上的一个部落。纯数据结构，不引用任何适配器。
type Tribe struct {
	Meta   Meta      `json:"meta"`
	Status Status    `json:"status"`
	Vital  VitalSign `json:"vital"`
	mu     sync.RWMutex
}

// SetVital 线程安全地更新生命体征，并根据 CPU 推算状态。
func (t *Tribe) SetVital(v VitalSign) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Vital = v
	t.Vital.UpdatedAt = time.Now().UnixMilli()
	switch {
	case v.PID == 0:
		t.Status = StatusIdle
	case v.CPU > 70:
		t.Status = StatusBusy
	default:
		t.Status = StatusRun
	}
}

// GetVital 读取当前生命体征。
func (t *Tribe) GetVital() VitalSign {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Vital
}

// Land 大陆——所有部落的容器。
// 按能力分别保存适配器引用，让消费方通过类型安全的接口取用。
type Land struct {
	mu        sync.RWMutex
	tribes    map[string]*Tribe
	detectors map[string]Detector
	launchers map[string]Launcher
	readers   map[string]Reader
}

// NewLand 构造一片空大陆。
func NewLand() *Land {
	return &Land{
		tribes:    make(map[string]*Tribe),
		detectors: make(map[string]Detector),
		launchers: make(map[string]Launcher),
		readers:   make(map[string]Reader),
	}
}

// Register 把一个部落登记到大陆。
// adapter 至少要实现 Identity；其它能力按需可选（nil 表示该部落不支持该能力）。
// 只在 Register 时做一次类型断言，运行期不再需要。
func (l *Land) Register(adapter any) error {
	id, ok := adapter.(Identity)
	if !ok {
		return ErrNotIdentity
	}
	m := Meta{
		ID:          id.ID(),
		Name:        id.Name(),
		Eco:         id.EcoType(),
		ThemeColor:  id.ThemeColor(),
		AccentColor: id.AccentColor(),
	}
	tribe := &Tribe{Meta: m, Status: StatusIdle}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.tribes[m.ID] = tribe
	if d, ok := adapter.(Detector); ok {
		l.detectors[m.ID] = d
	}
	if la, ok := adapter.(Launcher); ok {
		l.launchers[m.ID] = la
	}
	if r, ok := adapter.(Reader); ok {
		l.readers[m.ID] = r
	}
	return nil
}

// Detector 取出一个部落的进程检测器。
func (l *Land) Detector(id string) (Detector, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	d, ok := l.detectors[id]
	return d, ok
}

// Launcher 取出一个部落的启动器。
func (l *Land) Launcher(id string) (Launcher, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	la, ok := l.launchers[id]
	return la, ok
}

// Reader 取出一个部落的配置读取器。
func (l *Land) Reader(id string) (Reader, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	r, ok := l.readers[id]
	return r, ok
}

// Get 取出部落数据。
func (l *Land) Get(id string) (*Tribe, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	t, ok := l.tribes[id]
	return t, ok
}

// Snapshot 全貌——前端俯瞰时拿这个。
func (l *Land) Snapshot() map[string]Tribe {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make(map[string]Tribe, len(l.tribes))
	for id, t := range l.tribes {
		t.mu.RLock()
		out[id] = Tribe{
			Meta:   t.Meta,
			Status: t.Status,
			Vital:  t.Vital,
		}
		t.mu.RUnlock()
	}
	return out
}

// IDs 列出所有部落 ID。
func (l *Land) IDs() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]string, 0, len(l.tribes))
	for id := range l.tribes {
		out = append(out, id)
	}
	return out
}

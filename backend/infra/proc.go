package infra

import (
	"context"
	"sync"
	"time"

	"github.com/narcilee7/aland/backend/core"
	"github.com/narcilee7/aland/backend/tribes"
)

// ProcManager 进程扫描器。
// 每 2 秒扫一次各部落，更新 Land.Vital。
// M0 是单进程内同步实现；v1 可以考虑跨平台抽象（macOS lsof / Windows WMI / Linux /proc）。
type ProcManager struct {
	Land  *core.Land
	Tick  time.Duration
	mu    sync.Mutex
	stop  chan struct{}
}

func NewProcManager(land *core.Land) *ProcManager {
	return &ProcManager{
		Land: land,
		Tick: 2 * time.Second,
		stop: make(chan struct{}),
	}
}

// Start 启动扫描循环。Cancel ctx 即停止。
func (p *ProcManager) Start(ctx context.Context) {
	go func() {
		t := time.NewTicker(p.Tick)
		defer t.Stop()
		// 先扫一次，避免启动后 2s 没数据
		p.scanOnce()
		for {
			select {
			case <-ctx.Done():
				return
			case <-p.stop:
				return
			case <-t.C:
				p.scanOnce()
			}
		}
	}()
}

// Stop 主动停止。
func (p *ProcManager) Stop() {
	close(p.stop)
}

func (p *ProcManager) scanOnce() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, tribe := range p.Land.Tribes {
		adapter := tribes.AsAdapter(tribe)
		if adapter == nil {
			continue
		}
		proc, err := adapter.DetectProcess()
		if err != nil || proc == nil {
			tribe.SetVital(core.VitalSign{PID: 0})
			continue
		}
		v := core.VitalSign{
			PID:    proc.PID,
			CWD:    proc.CWD,
			CPU:    proc.CPU,
			Memory: proc.Memory,
		}
		tribe.SetVital(v)
	}
}

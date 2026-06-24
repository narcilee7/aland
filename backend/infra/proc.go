package infra

import (
	"context"
	"sync"
	"time"

	"github.com/narcilee7/aland/backend/core"
	"github.com/narcilee7/aland/backend/tribes"
)

// ProcManager 进程扫描器。
// 每 2 秒扫一次各部落，更新其 Vital。
// 它只依赖 tribes.Detector 与 tribes.Land；不关心具体适配器类型。
type ProcManager struct {
	land *tribes.Land
	tick time.Duration
	mu   sync.Mutex
	stop chan struct{}
}

// NewProcManager 构造一个进程扫描器。
func NewProcManager(land *tribes.Land) *ProcManager {
	return &ProcManager{
		land: land,
		tick: 2 * time.Second,
		stop: make(chan struct{}),
	}
}

// Start 启动扫描循环。Cancel ctx 即停止。
func (p *ProcManager) Start(ctx context.Context) {
	go func() {
		t := time.NewTicker(p.tick)
		defer t.Stop()
		// 启动即扫一次，避免头两秒没数据
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
	for _, id := range p.land.IDs() {
		det, ok := p.land.Detector(id)
		if !ok {
			continue
		}
		tribe, ok := p.land.Get(id)
		if !ok {
			continue
		}
		proc, err := det.DetectProcess()
		if err != nil {
			core.Log.Warn("detect process failed", "tribe", id, "err", err)
			tribe.SetVital(tribes.VitalSign{PID: 0})
			continue
		}
		if proc == nil {
			tribe.SetVital(tribes.VitalSign{PID: 0})
			continue
		}
		tribe.SetVital(tribes.VitalSign{
			PID:    proc.PID,
			CWD:    proc.CWD,
			CPU:    proc.CPU,
			Memory: proc.Memory,
		})
	}
}

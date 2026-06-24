package events

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Emitter 把事件名 + payload 投递给 Wails runtime。
// 抽象这一层有两个目的：
//   1. 调用方不需要重复 import wails runtime
//   2. 后续可以换实现（测试时塞 channel、生产时塞 Wails）
type Emitter struct {
	ctx context.Context
}

// New 构造一个 emitter，绑定到给定 ctx（必须来自 Wails Startup）。
func New(ctx context.Context) *Emitter {
	return &Emitter{ctx: ctx}
}

// Emit 推一个事件。payload 会被 Wails 序列化到前端。
func (e *Emitter) Emit(name string, payload any) {
	if e == nil || e.ctx == nil {
		return
	}
	runtime.EventsEmit(e.ctx, name, payload)
}

// EmitTribeVital tribe:vital 的快捷方法。
func (e *Emitter) EmitTribeVital(snap TribeSnapshotMap) {
	e.Emit(TribeVital, snap)
}

// EmitTribeBorn tribe:born 的快捷方法。
func (e *Emitter) EmitTribeBorn(id string, pid int, name string) {
	e.Emit(TribeBorn, TribeLifecycleEvent{ID: id, PID: pid, Name: name})
}

// EmitTribeDeath tribe:death 的快捷方法。
func (e *Emitter) EmitTribeDeath(id string, pid int, name string) {
	e.Emit(TribeDeath, TribeLifecycleEvent{ID: id, PID: pid, Name: name})
}

// EmitFSChange fs:change 的快捷方法。
func (e *Emitter) EmitFSChange(path, op string) {
	e.Emit(FSChange, FSChangeEvent{Path: path, Op: op})
}

// EmitSpotlightToggle spotlight:toggle 的快捷方法。
func (e *Emitter) EmitSpotlightToggle(action string) {
	e.Emit(SpotlightToggle, SpotlightToggleEvent{Action: action})
}

// EmitSessionEvent session:event 的快捷方法。
func (e *Emitter) EmitSessionEvent(ev any) {
	e.Emit(SessionEvent, ev)
}

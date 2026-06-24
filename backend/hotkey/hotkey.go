// Package hotkey 注册全局系统快捷键，按下时通过 events.Emitter 推送。
//
// 设计：每个快捷键一个 goroutine 监听 channel，ctx cancel 时优雅退出。
package hotkey

import (
	"context"

	"github.com/narcilee7/aland/backend/core"
	"github.com/narcilee7/aland/backend/events"
	"golang.design/x/hotkey"
)

// CmdShiftA 注册 macOS 上的 ⌘⇧A（其他平台走 fallback 修饰键组合）。
func CmdShiftA(ctx context.Context, em *events.Emitter) error {
	hk := hotkey.New(
		[]hotkey.Modifier{hotkey.ModCmd, hotkey.ModShift},
		hotkey.KeyA,
	)
	if err := hk.Register(); err != nil {
		return err
	}
	core.Log.Info("hotkey registered", "combo", "Cmd+Shift+A")

	go func() {
		for {
			select {
			case <-ctx.Done():
				_ = hk.Unregister()
				return
			case <-hk.Keydown():
				core.Log.Debug("hotkey down", "combo", "Cmd+Shift+A")
				em.EmitSpotlightToggle("toggle")
			}
		}
	}()
	return nil
}

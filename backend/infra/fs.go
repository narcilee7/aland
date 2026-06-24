package infra

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/narcilee7/aland/backend/core"
	"github.com/narcilee7/aland/backend/events"
)

// FSWatch 监听配置文件变更，通过 events.Emitter 推送到前端。
// 防抖 500ms，避免编辑器保存时的多事件。
type FSWatch struct {
	watcher  *fsnotify.Watcher
	emitter  *events.Emitter
	ctx      context.Context
	paths    []string
	debounce time.Duration
}

func NewFSWatch(ctx context.Context, paths []string) *FSWatch {
	return &FSWatch{
		emitter:  events.New(ctx),
		ctx:      ctx,
		paths:    paths,
		debounce: 500 * time.Millisecond,
	}
}

// Start 启动监听。建议在 app.startup 里调用。
func (f *FSWatch) Start() error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	f.watcher = w

	// 对每个路径，监听其父目录（fsnotify 只能监听目录）
	watched := map[string]bool{}
	for _, p := range f.paths {
		dir := filepath.Dir(p)
		if watched[dir] {
			continue
		}
		if err := w.Add(dir); err != nil {
			core.Log.Warn("fs watch add failed", "dir", dir, "err", err)
			continue
		}
		watched[dir] = true
	}
	core.Log.Info("fs watch started", "paths", len(f.paths))

	go f.loop()
	return nil
}

func (f *FSWatch) loop() {
	var lastEmit time.Time
	for {
		select {
		case <-f.ctx.Done():
			return
		case ev, ok := <-f.watcher.Events:
			if !ok {
				return
			}
			if !f.match(ev.Name) {
				continue
			}
			now := time.Now()
			if now.Sub(lastEmit) < f.debounce {
				continue
			}
			lastEmit = now
			f.emitter.EmitFSChange(ev.Name, ev.Op.String())
		case err, ok := <-f.watcher.Errors:
			if !ok {
				return
			}
			core.Log.Warn("fs watch error", "err", err)
		}
	}
}

func (f *FSWatch) match(name string) bool {
	base := filepath.Base(name)
	for _, p := range f.paths {
		if strings.HasSuffix(name, p) || strings.HasSuffix(name, base) {
			return true
		}
	}
	return false
}

// Close 释放资源。
func (f *FSWatch) Close() error {
	if f.watcher != nil {
		return f.watcher.Close()
	}
	return nil
}

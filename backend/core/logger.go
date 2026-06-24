package core

import (
	"log/slog"
	"os"
)

// Log 是 Aland 的全局结构化 logger。
// 基于 Go 1.22 stdlib slog，文本 handler（人友好）；后续可换 JSON handler 上报到 Forge。
//
// 所有后端模块都通过 core.Log 调用，禁止直接打 fmt.Println。
// 字段约定：
//   - "tribe" / "id" 部落 ID
//   - "pid"           进程 PID
//   - "op"            文件操作（fs watcher 用）
//   - "err"           错误（用 slog 的 err attribute）
var Log *slog.Logger

func init() {
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false, // 调试时改 true 能拿到调用方文件/行号
	})
	Log = slog.New(h)
}

// SetLevel 动态调整日志等级。
// main.go 在启动早期可调用 core.SetLevel(slog.LevelDebug)。
func SetLevel(level slog.Level) {
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: false,
	})
	Log = slog.New(h)
}

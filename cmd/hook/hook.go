// Package hook 实现 `aland hook <event>` 子命令。
//
// 这是 Aland 二进制的轻量模式——Claude Code 在每次 hook 事件时 spawn 它，
// 不加载 Wails/webview，只读 stdin + POST 到本机 HTTP server。
//
// 协议：
//   - 从 stdin 读 JSON（Claude Code 原生 hook payload）
//   - POST http://127.0.0.1:<port>/hook （端口从 ~/.aland/hook.port）
//   - 把 server 响应回写到 stdout（Claude Code 解析 PreToolUse 决策）
//
// 性能目标：< 50ms。Aland 没运行或 server 不可达时静默 exit 0，
// 这样 Claude Code 不会因为 hook 子命令失败而中断 agent。
package hook

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/narcilee7/aland/backend/hooks"
)

// Run 执行 hook 子命令。
// args[0] 应该是事件名（如 "PreToolUse"）。
// 从 stdin 读 JSON，POST 到 Aland server，写回 stdout。
// 任何错误都返回 ExitCode，由 main 决定 exit。
func Run(args []string) int {
	return RunWithStdio(args, os.Stdin, os.Stdout)
}

// RunWithStdio 是可测试版本——把 stdin/stdout 注入。
func RunWithStdio(args []string, stdin io.Reader, stdout io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintf(stdout, "usage: aland hook <event>\n")
		return 2
	}

	port, err := hooks.ReadPort()
	if err != nil {
		// Aland 没运行——静默 0 退出，不让 Claude Code 受影响
		return 0
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/hook", port)
	req, err := http.NewRequest("POST", url, stdin)
	if err != nil {
		return 0
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	_, _ = io.Copy(stdout, resp.Body)
	return 0
}
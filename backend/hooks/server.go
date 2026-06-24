// hooks HTTP server + 端口探测
package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/narcilee7/aland/backend/core"
)

// PortBase 是 Aland hook server 尝试绑定的起始端口。
// 如果被占用，依次尝试 PortBase+1, +2, ...
const PortBase = 38765
const PortTry = 10

// PortFileName 写入用户主目录的文件名，让 aland hook CLI 知道 server 端口。
const PortFileName = "hook.port"

// Server 是接收 Claude hook 事件并转发给上层消费的 HTTP 服务器。
type Server struct {
	mu       sync.Mutex
	listener net.Listener
	port     int
	srv      *http.Server

	// onEvent 每次收到事件时调用。payload 已解析。
	onEvent func(HookPayload)

	// PortFile 保存端口的文件路径。
	PortFile string
}

// New 构造一个未启动的 Server。
func New(onEvent func(HookPayload)) *Server {
	home, _ := os.UserHomeDir()
	return &Server{
		onEvent:  onEvent,
		PortFile: filepath.Join(home, ".aland", PortFileName),
	}
}

// Start 绑定端口并启动 server。失败返回错误（端口冲突等）。
func (s *Server) Start(ctx context.Context) error {
	port, listener, err := bindPort()
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.port = port
	s.listener = listener
	s.mu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/hook", s.handleHook)

	s.srv = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// 写端口到文件，方便 CLI 子命令发现
	if err := s.writePortFile(port); err != nil {
		core.Log.Warn("hook port file write failed", "err", err)
	}

	go func() {
		if err := s.srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			core.Log.Error("hook server stopped", "err", err)
		}
	}()
	core.Log.Info("hook server started", "port", port, "pid", os.Getpid())
	return nil
}

// Stop 关停 server 并清理 port 文件。
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.srv == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := s.srv.Shutdown(ctx)
	_ = os.Remove(s.PortFile)
	return err
}

// Port 返回 server 绑定的端口。
func (s *Server) Port() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.port
}

// bindPort 尝试绑定 PortBase .. PortBase+PortTry-1 区间。
func bindPort() (int, net.Listener, error) {
	for i := 0; i < PortTry; i++ {
		port := PortBase + i
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		l, err := net.Listen("tcp", addr)
		if err == nil {
			return port, l, nil
		}
	}
	return 0, nil, fmt.Errorf("no free port in %d..%d", PortBase, PortBase+PortTry-1)
}

func (s *Server) writePortFile(port int) error {
	dir := filepath.Dir(s.PortFile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.PortFile, []byte(fmt.Sprintf("%d\n", port)), 0o644)
}

// —— handlers ——

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

// handleHook 接收 Claude hook POST。
// Body 是 Claude 通过 stdin 传给 hook CLI 的同一份 JSON。
func (s *Server) handleHook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB 上限
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}

	var p HookPayload
	if err := json.Unmarshal(body, &p); err != nil {
		http.Error(w, "parse json: "+err.Error(), http.StatusBadRequest)
		return
	}

	if s.onEvent != nil {
		s.onEvent(p)
	}

	// 默认返回"通过"。PreToolUse 的精细决策留给后续 commit。
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"continue":true}`))
}

// ReadPort 从 ~/.aland/hook.port 读出 server 端口。
// CLI 子命令用这个发现 server。
func ReadPort() (int, error) {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".aland", PortFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var port int
	if _, err := fmt.Sscanf(string(data), "%d", &port); err != nil {
		return 0, err
	}
	if port < PortBase || port >= PortBase+PortTry {
		return 0, fmt.Errorf("port %d out of expected range", port)
	}
	return port, nil
}
package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestServer_StartStop(t *testing.T) {
	_, restore := withTempHome(t)
	defer restore()

	srv := New(func(p HookPayload) {
		// noop; just verify start/stop lifecycle
		_ = p
	})

	if err := srv.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Stop()

	if srv.Port() < PortBase {
		t.Errorf("port=%d, want >=%d", srv.Port(), PortBase)
	}

	// 验证 port file 存在
	data, err := os.ReadFile(srv.PortFile)
	if err != nil {
		t.Fatalf("read port file: %v", err)
	}
	if !strings.Contains(string(data), "3876") {
		t.Errorf("port file content unexpected: %s", data)
	}
}

func TestServer_HandleHook(t *testing.T) {
	_, restore := withTempHome(t)
	defer restore()

	var received HookPayload
	srv := New(func(p HookPayload) {
		received = p
	})
	if err := srv.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	// 构造一个 PreToolUse 事件 JSON
	payload := HookPayload{
		SessionID:     "test-session",
		HookEventName: "PreToolUse",
		ToolName:      "Bash",
		CWD:           "/tmp",
		ToolInput: map[string]any{
			"command": "ls -la",
		},
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(
		"http://127.0.0.1:"+strconv.Itoa(srv.Port())+"/hook",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status=%d, want 200", resp.StatusCode)
	}

	// 等待回调触发（goroutine 异步）
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if received.HookEventName == "PreToolUse" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if received.HookEventName != "PreToolUse" {
		t.Errorf("event not received: %+v", received)
	}
	if received.ToolName != "Bash" {
		t.Errorf("tool name: got %s, want Bash", received.ToolName)
	}
}

func TestServer_Health(t *testing.T) {
	_, restore := withTempHome(t)
	defer restore()

	srv := New(nil)
	if err := srv.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	resp, err := http.Get("http://127.0.0.1:" + strconv.Itoa(srv.Port()) + "/health")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"ok":true`) {
		t.Errorf("health body: %s", body)
	}
}

func TestServer_PortConflict_Fallback(t *testing.T) {
	_, restore := withTempHome(t)
	defer restore()

	// 占用 PortBase
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", PortBase))
	if err != nil {
		t.Fatalf("can't bind PortBase: %v", err)
	}
	defer l.Close()

	srv := New(nil)
	if err := srv.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Stop()
	if srv.Port() == PortBase {
		t.Errorf("expected fallback port, got %d", srv.Port())
	}
}

func TestReadPort(t *testing.T) {
	dir, restore := withTempHome(t)
	defer restore()

	// 文件不存在
	if _, err := ReadPort(); err == nil {
		t.Error("expected error when port file missing")
	}

	// 写入合法
	portFile := filepath.Join(dir, ".aland", PortFileName)
	if err := os.MkdirAll(filepath.Dir(portFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(portFile, []byte("38770\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	port, err := ReadPort()
	if err != nil {
		t.Fatalf("ReadPort: %v", err)
	}
	if port != 38770 {
		t.Errorf("port=%d, want 38770", port)
	}
}

// helpers

func listenOn_unused() {} // 占位——移除旧 stub
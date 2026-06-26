package hook

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/narcilee7/aland/backend/hooks"
)

func TestRun_NoArgs_PrintsUsage(t *testing.T) {
	var out bytes.Buffer
	code := RunWithStdio(nil, strings.NewReader(""), &out)
	if code != 2 {
		t.Errorf("exit code=%d, want 2", code)
	}
	if !strings.Contains(out.String(), "usage") {
		t.Errorf("output missing usage: %s", out.String())
	}
}

func TestRun_NoPortFile_Exits0(t *testing.T) {
	// 用临时 HOME 隔离
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", oldHome)

	var out bytes.Buffer
	code := RunWithStdio([]string{"PreToolUse"}, strings.NewReader(`{"foo":"bar"}`), &out)
	if code != 0 {
		t.Errorf("exit code=%d, want 0 (silent no-op)", code)
	}
}

func TestRun_ForwardToServer(t *testing.T) {
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", oldHome)

	// 启 server
	srv := hooks.New(func(p hooks.HookPayload) {
		// 验证：这里不需要读 p，直接通过 http 反向验证 server 收没收到即可
		_ = p
	})
	if err := srv.Start(context.Background()); err != nil {
		t.Fatalf("server start: %v", err)
	}
	defer srv.Stop()

	// POST 一个事件
	payload := hooks.HookPayload{
		SessionID:     "test",
		HookEventName: "PreToolUse",
		ToolName:      "Bash",
	}
	body, _ := json.Marshal(payload)

	var out bytes.Buffer
	code := RunWithStdio([]string{"PreToolUse"}, bytes.NewReader(body), &out)
	if code != 0 {
		t.Errorf("exit code=%d, want 0", code)
	}

	// server 响应默认 {"continue":true}
	if !strings.Contains(out.String(), `"continue":true`) {
		t.Errorf("response: %s", out.String())
	}

	// server 应该已经收到事件（异步，给点时间）
	// 直接 curl 验证 server 仍然工作
	resp, err := http.Get("http://127.0.0.1:" + itoa(srv.Port()) + "/health")
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	resp.Body.Close()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
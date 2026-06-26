package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withTempHome 临时把 HOME 切到 dir，返回恢复函数。
// Install/Uninstall 等函数读 ~/.claude/settings.json，会被这个影响。
func withTempHome(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	return dir, func() { os.Setenv("HOME", oldHome) }
}

func writeSettings(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readSettings(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestIsAlandCommand(t *testing.T) {
	cases := []struct {
		cmd  string
		want bool
	}{
		{"aland hook PreToolUse", true},
		{"/usr/local/bin/aland hook PreToolUse", true},
		{"/Users/x/go/bin/aland hook Stop", true},
		{"my-tool hook something", false},
		{"some-random-command", false},
		{"", false},
		{"  aland hook PreToolUse  ", true}, // leading whitespace ok
	}
	for _, c := range cases {
		if got := IsAlandCommand(c.cmd); got != c.want {
			t.Errorf("IsAlandCommand(%q)=%v, want %v", c.cmd, got, c.want)
		}
	}
}

func TestInstall_EmptyFile(t *testing.T) {
	_, restore := withTempHome(t)
	defer restore()

	res, err := Install()
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(res.Added) != len(AllEvents) {
		t.Errorf("Added=%v, want all %d events", res.Added, len(AllEvents))
	}
	if len(res.Skipped) != 0 {
		t.Errorf("Skipped=%v, want empty", res.Skipped)
	}

	// 验证文件结构
	raw := readSettings(t, SettingsPath())
	hooks, ok := raw["hooks"].(map[string]any)
	if !ok {
		t.Fatal("hooks field missing")
	}
	for _, ev := range AllEvents {
		groups, ok := hooks[ev].([]any)
		if !ok || len(groups) == 0 {
			t.Errorf("event %s: no matchers registered", ev)
		}
	}
}

func TestInstall_Idempotent(t *testing.T) {
	_, restore := withTempHome(t)
	defer restore()

	// 第一次安装
	res1, err := Install()
	if err != nil {
		t.Fatalf("first Install: %v", err)
	}
	if len(res1.Added) != len(AllEvents) {
		t.Fatalf("first install should add all events, got %v", res1.Added)
	}

	// 第二次安装：应该全部 skipped
	res2, err := Install()
	if err != nil {
		t.Fatalf("second Install: %v", err)
	}
	if len(res2.Added) != 0 {
		t.Errorf("second install should add nothing, got %v", res2.Added)
	}
	if len(res2.Skipped) != len(AllEvents) {
		t.Errorf("second install should skip all events, got %v", res2.Skipped)
	}

	// 检查每个事件只有一组 matcher（没有重复注册）
	raw := readSettings(t, SettingsPath())
	hooks, _ := raw["hooks"].(map[string]any)
	for _, ev := range AllEvents {
		groups := hooks[ev].([]any)
		alandCount := 0
		for _, g := range groups {
			gg := g.(map[string]any)
			cmds := gg["hooks"].([]any)
			for _, c := range cmds {
				cc := c.(map[string]any)
				if IsAlandCommand(cc["command"].(string)) {
					alandCount++
				}
			}
		}
		if alandCount != 1 {
			t.Errorf("event %s: aland hook count = %d, want 1", ev, alandCount)
		}
	}
}

func TestInstall_PreservesUserHooks(t *testing.T) {
	_, restore := withTempHome(t)
	defer restore()

	// 写一个已经有 user-defined hook 的 settings.json
	userCmd := "echo user-defined-hook && custom-script"
	writeSettings(t, SettingsPath(), `{
  "model": "claude-opus-4",
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {"type": "command", "command": "`+userCmd+`"}
        ]
      }
    ]
  }
}`)

	if _, err := Install(); err != nil {
		t.Fatalf("Install: %v", err)
	}

	raw := readSettings(t, SettingsPath())

	// 1. 非 hooks 字段被保留
	if raw["model"] != "claude-opus-4" {
		t.Errorf("model field lost: got %v", raw["model"])
	}

	// 2. 用户的 PreToolUse matcher 仍在
	hooks := raw["hooks"].(map[string]any)
	groups := hooks["PreToolUse"].([]any)
	if len(groups) != 2 {
		t.Fatalf("PreToolUse should have 2 matchers (user + aland), got %d", len(groups))
	}
	foundUser := false
	for _, g := range groups {
		gg := g.(map[string]any)
		cmds := gg["hooks"].([]any)
		for _, c := range cmds {
			cc := c.(map[string]any)
			if cc["command"] == userCmd {
				foundUser = true
			}
		}
	}
	if !foundUser {
		t.Error("user-defined hook was removed")
	}
}

func TestUninstall(t *testing.T) {
	_, restore := withTempHome(t)
	defer restore()

	if _, err := Install(); err != nil {
		t.Fatalf("Install: %v", err)
	}

	res, err := Uninstall()
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if len(res.Added) != len(AllEvents) {
		t.Errorf("Uninstall should report removed events, got %v", res.Added)
	}

	// 检查文件里 Aland 痕迹
	data, _ := os.ReadFile(SettingsPath())
	if IsAlandCommand(string(data)) {
		t.Errorf("Uninstall left Aland entries: %s", data)
	}
}

func TestUninstall_KeepsUserHooks(t *testing.T) {
	_, restore := withTempHome(t)
	defer restore()

	userCmd := "echo user-defined"
	writeSettings(t, SettingsPath(), `{
  "hooks": {
    "PreToolUse": [
      {"matcher": "Bash", "hooks": [{"type": "command", "command": "`+userCmd+`"}]}
    ]
  }
}`)
	if _, err := Install(); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if _, err := Uninstall(); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	raw := readSettings(t, SettingsPath())
	hooks, _ := raw["hooks"].(map[string]any)
	groups, _ := hooks["PreToolUse"].([]any)
	if len(groups) != 1 {
		t.Fatalf("user hook should remain, got %d groups", len(groups))
	}
	cmds := groups[0].(map[string]any)["hooks"].([]any)
	if cmds[0].(map[string]any)["command"] != userCmd {
		t.Errorf("user hook changed")
	}
}

func TestIsInstalled(t *testing.T) {
	_, restore := withTempHome(t)
	defer restore()

	// 文件不存在
	ok, err := IsInstalled()
	if err != nil {
		t.Fatalf("IsInstalled (no file): %v", err)
	}
	if ok {
		t.Error("should not be installed when file missing")
	}

	// 安装后
	if _, err := Install(); err != nil {
		t.Fatal(err)
	}
	ok, err = IsInstalled()
	if err != nil {
		t.Fatalf("IsInstalled (after): %v", err)
	}
	if !ok {
		t.Error("should be installed after Install()")
	}
}

func TestInstall_BackupCreated(t *testing.T) {
	_, restore := withTempHome(t)
	defer restore()

	// 先写一个原始文件
	original := `{"model": "claude-opus-4", "hooks": {}}`
	writeSettings(t, SettingsPath(), original)

	if _, err := Install(); err != nil {
		t.Fatal(err)
	}

	// 检查备份目录
	entries, err := os.ReadDir(BackupDir())
	if err != nil {
		t.Fatalf("read backup dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no backup created")
	}
	// 第一个备份应该是原文件
	first := entries[0]
	data, _ := os.ReadFile(filepath.Join(BackupDir(), first.Name()))
	if !strings.Contains(string(data), `"claude-opus-4"`) {
		t.Errorf("backup doesn't contain original content: %s", data)
	}
}
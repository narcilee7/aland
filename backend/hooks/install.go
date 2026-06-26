// hooks install/uninstall —— 把 Aland 注册到 Claude Code 的 settings.json
//
// 目标文件：~/.claude/settings.json
// 备份目标：~/.aland/backups/settings-{unix}.json
//
// 注册格式（在 hooks.PreToolUse 等字段下追加 matcher group）：
//
//	{ "matcher": ".*", "hooks": [{ "type": "command", "command": "<exe> hook PreToolUse" }] }
//
// 幂等性：通过命令字符串包含 " hook " 字段识别 Aland 自己的条目。
package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/narcilee7/aland/backend/core"
)

// alandCommand 返回完整可执行路径——`aland hook <event>` 子命令的格式。
// 这里固定用 "aland"，假设 PATH 里能找到；调用方通过环境变量覆盖。
func alandCommand(event string) string {
	return fmt.Sprintf("aland hook %s", event)
}

// SettingsPath 返回 Claude Code settings.json 路径。
func SettingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "settings.json")
}

// BackupDir 返回 Aland 备份目录。
func BackupDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aland", "backups")
}

// InstallResult Install 的返回——哪些事件被新注册、哪些已存在。
type InstallResult struct {
	Added   []string // 新增 Aland 条目的事件
	Skipped []string // 已经存在 Aland 条目的事件
}

// IsAlandCommand 判定一条 hook command 是不是 Aland 自己注册的。
// 启发式：命令以 "aland hook " 开头（可以是 "aland" 也可以是 "/path/to/aland"）。
// 避免误删用户自定义的同名脚本。
func IsAlandCommand(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	return strings.HasPrefix(cmd, "aland hook ") || strings.HasPrefix(cmd, "/") && strings.Contains(cmd, "/aland hook ")
}

// Install 把 Aland 注册到 settings.json 的所有事件。
//
// 幂等：已经注册过的事件不会重复添加。
// 文件不存在时新建；存在时先备份再写回。
// settings.json 的所有其他字段保持不变。
func Install() (*InstallResult, error) {
	path := SettingsPath()

	// 1. 读取或初始化
	raw := map[string]any{}
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read settings: %w", err)
	}
	if err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse settings: %w", err)
		}
	}

	// 2. 备份原文件（如果存在）
	if _, err := os.Stat(path); err == nil {
		if err := backupSettings(path, data); err != nil {
			core.Log.Warn("settings backup failed", "err", err)
		}
	}

	// 3. 修改 hooks 字段
	hooksRaw, _ := raw["hooks"].(map[string]any)
	if hooksRaw == nil {
		hooksRaw = map[string]any{}
	}

	res := &InstallResult{}
	for _, event := range AllEvents {
		added, err := installEvent(hooksRaw, event)
		if err != nil {
			return res, err
		}
		if added {
			res.Added = append(res.Added, event)
		} else {
			res.Skipped = append(res.Skipped, event)
		}
	}

	raw["hooks"] = hooksRaw

	// 4. 写回（带 2 空格缩进，匹配 Claude Code 习惯）
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return res, fmt.Errorf("marshal settings: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return res, fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return res, fmt.Errorf("write settings: %w", err)
	}
	core.Log.Info("hooks installed", "added", res.Added, "skipped", res.Skipped)
	return res, nil
}

// installEvent 把 Aland 注册到 hooks.<event>。返回是否实际新增。
func installEvent(hooks map[string]any, event string) (bool, error) {
	// 已有 matcher group 列表
	existing, _ := hooks[event].([]any)

	// 检查是否已注册
	for _, m := range existing {
		mm, ok := m.(map[string]any)
		if !ok {
			continue
		}
		cmds, _ := mm["hooks"].([]any)
		for _, c := range cmds {
			cc, ok := c.(map[string]any)
			if !ok {
				continue
			}
			cmd, _ := cc["command"].(string)
			if IsAlandCommand(cmd) {
				return false, nil // 已有，跳过
			}
		}
	}

	// 新增一个 matcher group：Aland 接所有事件
	newMatcher := map[string]any{
		"matcher": ".*",
		"hooks": []any{
			map[string]any{
				"type":    "command",
				"command": alandCommand(event),
			},
		},
	}
	hooks[event] = append(existing, newMatcher)
	return true, nil
}

// Uninstall 移除所有 Aland 注册的 hook 条目。
// 保留用户自定义的其他 hooks。
func Uninstall() (*InstallResult, error) {
	path := SettingsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &InstallResult{}, nil
		}
		return nil, err
	}

	// 备份
	if err := backupSettings(path, data); err != nil {
		core.Log.Warn("settings backup failed", "err", err)
	}

	raw := map[string]any{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	hooksRaw, _ := raw["hooks"].(map[string]any)

	res := &InstallResult{}
	for _, event := range AllEvents {
		removed, ok := hooksRaw[event].([]any)
		if !ok {
			res.Skipped = append(res.Skipped, event)
			continue
		}
		kept := removed[:0]
		var removedOne bool
		for _, m := range removed {
			mm, ok := m.(map[string]any)
			if !ok {
				kept = append(kept, m)
				continue
			}
			cmds, _ := mm["hooks"].([]any)
			newCmds := cmds[:0]
			var alandFound bool
			for _, c := range cmds {
				cc, ok := c.(map[string]any)
				if !ok {
					newCmds = append(newCmds, c)
					continue
				}
				cmd, _ := cc["command"].(string)
				if IsAlandCommand(cmd) {
					alandFound = true
					continue
				}
				newCmds = append(newCmds, c)
			}
			if alandFound {
				removedOne = true
				if len(newCmds) == 0 {
					// 整组只有 Aland → 删整组
					continue
				}
				mm["hooks"] = newCmds
			}
			kept = append(kept, mm)
		}
		if removedOne {
			res.Added = append(res.Added, event)
			if len(kept) == 0 {
				delete(hooksRaw, event)
			} else {
				hooksRaw[event] = kept
			}
		} else {
			res.Skipped = append(res.Skipped, event)
		}
	}
	raw["hooks"] = hooksRaw

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return res, err
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return res, err
	}
	return res, nil
}

func backupSettings(src string, content []byte) error {
	if err := os.MkdirAll(BackupDir(), 0o755); err != nil {
		return err
	}
	stamp := time.Now().Unix()
	dst := filepath.Join(BackupDir(), fmt.Sprintf("settings-%d.json", stamp))
	return os.WriteFile(dst, content, 0o644)
}

// IsInstalled 检查 settings.json 是否已经有 Aland 注册（不修改文件）。
func IsInstalled() (bool, error) {
	data, err := os.ReadFile(SettingsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	raw := map[string]any{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return false, err
	}
	hooksRaw, ok := raw["hooks"].(map[string]any)
	if !ok {
		return false, nil
	}
	for _, event := range AllEvents {
		groups, _ := hooksRaw[event].([]any)
		for _, m := range groups {
			mm, ok := m.(map[string]any)
			if !ok {
				continue
			}
			cmds, _ := mm["hooks"].([]any)
			for _, c := range cmds {
				cc, ok := c.(map[string]any)
				if !ok {
					continue
				}
				cmd, _ := cc["command"].(string)
				if IsAlandCommand(cmd) {
					return true, nil
				}
			}
		}
	}
	return false, nil
}
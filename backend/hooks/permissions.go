// Permission 规则解析与编辑——Claude Code settings.json 的 permissions 字段。
//
// Claude Code 权限模型：
//   - allow: 自动允许的规则列表
//   - deny:  自动拒绝的规则列表
//   - ask:   需要用户确认的规则列表
//
// 规则格式示例：
//   "Bash(npm:*)"           允许所有 npm 命令
//   "Read(./src/**)"        允许读取 ./src 下所有文件
//   "Bash(rm:*)"            拒绝所有 rm 命令
//
// Aland 让你可视化 + 开关这些规则——不重建权限 UI，只是开关现有规则。
package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/narcilee7/aland/backend/core"
)

// Permissions 三类规则集合。
type Permissions struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
	Ask   []string `json:"ask"`
}

// Empty 检查规则集是否完全为空。
func (p Permissions) Empty() bool {
	return len(p.Allow) == 0 && len(p.Deny) == 0 && len(p.Ask) == 0
}

// ReadPermissions 从 settings.json 读取 permissions 字段。
// 字段不存在或为空时返回零值（不报错）。
func ReadPermissions() (Permissions, error) {
	data, err := os.ReadFile(SettingsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return Permissions{}, nil
		}
		return Permissions{}, err
	}
	raw := map[string]any{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return Permissions{}, err
	}
	permRaw, ok := raw["permissions"].(map[string]any)
	if !ok {
		return Permissions{}, nil
	}
	out := Permissions{}
	for _, dst := range []struct {
		key string
		fn  func([]string)
	}{
		{"allow", func(xs []string) { out.Allow = xs }},
		{"deny", func(xs []string) { out.Deny = xs }},
		{"ask", func(xs []string) { out.Ask = xs }},
	} {
		xs, _ := permRaw[dst.key].([]any)
		strings := make([]string, 0, len(xs))
		for _, x := range xs {
			if s, ok := x.(string); ok {
				strings = append(strings, s)
			}
		}
		dst.fn(strings)
	}
	return out, nil
}

// WritePermissions 把规则集写回 settings.json 的 permissions 字段。
// 写之前自动备份。保留其他字段不变。
func WritePermissions(p Permissions) error {
	path := SettingsPath()

	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read settings: %w", err)
	}

	// 备份（仅在文件存在时）
	if _, err := os.Stat(path); err == nil {
		_ = backupSettings(path, data)
	}

	raw := map[string]any{}
	if err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("parse settings: %w", err)
		}
	}

	permMap := map[string]any{
		"allow": anySlice(p.Allow),
		"deny":  anySlice(p.Deny),
		"ask":   anySlice(p.Ask),
	}
	raw["permissions"] = permMap

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	core.Log.Info("permissions written",
		"allow", len(p.Allow), "deny", len(p.Deny), "ask", len(p.Ask),
		"ts", time.Now().Unix())
	return os.WriteFile(path, out, 0o644)
}

// TogglePermission 切换一条规则在某类下的存在状态。
//   存在 → 移除；不存在 → 追加。
// 不会跨类移动（如果该规则在另一类里，会保留）。
func TogglePermission(category, rule string) (Permissions, error) {
	if category != "allow" && category != "deny" && category != "ask" {
		return Permissions{}, fmt.Errorf("invalid category: %s", category)
	}
	p, err := ReadPermissions()
	if err != nil {
		return p, err
	}
	list := map[string]*[]string{
		"allow": &p.Allow,
		"deny":  &p.Deny,
		"ask":   &p.Ask,
	}[category]
	if idx := indexOf(*list, rule); idx >= 0 {
		*list = append((*list)[:idx], (*list)[idx+1:]...)
	} else {
		*list = append(*list, rule)
	}
	if err := WritePermissions(p); err != nil {
		return p, err
	}
	return p, nil
}

func indexOf(xs []string, x string) int {
	for i, v := range xs {
		if v == x {
			return i
		}
	}
	return -1
}

func anySlice(xs []string) []any {
	out := make([]any, len(xs))
	for i, x := range xs {
		out[i] = x
	}
	return out
}
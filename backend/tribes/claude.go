package tribes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// ClaudeAdapter 适配 Claude Code CLI。
// 它是一个具体的实现，结构性满足 Identity/Detector/Launcher/Reader——
// 不需要显式声明接口实现。
type ClaudeAdapter struct {
	home string
}

// NewClaudeAdapter 构造一个 Claude 适配器。
// home 是用户主目录；如果为空，自动取 $HOME。
func NewClaudeAdapter(home string) *ClaudeAdapter {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return &ClaudeAdapter{home: home}
}

// —— Identity ——

func (c *ClaudeAdapter) ID() string      { return "claude" }
func (c *ClaudeAdapter) Name() string    { return "Claude Code" }
func (c *ClaudeAdapter) EcoType() string { return "classical" }

// ThemeColor 琥珀金。
func (c *ClaudeAdapter) ThemeColor() string  { return "#d4a853" }
func (c *ClaudeAdapter) AccentColor() string { return "#ffdf80" }

// —— Reader ——

// ConfigPaths 返回 Claude Code 配置目录下的关键文件。
func (c *ClaudeAdapter) ConfigPaths() []string {
	return []string{
		filepath.Join(c.home, ".claude", "settings.json"),
		filepath.Join(c.home, ".claude"),
	}
}

// ParseConfig 解析 settings.json。
// 失败时返回空 map（CLI 未安装也属于正常情况，不算错误）。
func (c *ClaudeAdapter) ParseConfig() (map[string]interface{}, error) {
	paths := c.ConfigPaths()
	if len(paths) == 0 {
		return map[string]interface{}{}, nil
	}
	data, err := os.ReadFile(paths[0])
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{}, nil
		}
		return nil, err
	}
	out := map[string]interface{}{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// —— Detector ——

// DetectProcess 扫描系统中正在运行的 claude 进程。
// macOS / Linux 用 ps；Windows 走 WMI（v1 再加）。
// 找不到返回 (nil, nil)，不算错误。
func (c *ClaudeAdapter) DetectProcess() (*ProcessInfo, error) {
	cmd := exec.Command("ps", "-axo", "pid=,comm=,args=")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	for _, line := range bytes.Split(out, []byte("\n")) {
		s := strings.TrimSpace(string(line))
		if !strings.Contains(strings.ToLower(s), "claude") {
			continue
		}
		// 跳过 ps 自身
		if strings.Contains(s, "grep") {
			continue
		}
		fields := strings.Fields(s)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil || pid == 0 {
			continue
		}
		return &ProcessInfo{
			PID:     pid,
			Name:    strings.Join(fields[1:], " "),
			CmdLine: s,
		}, nil
	}
	return nil, nil
}

// —— Launcher ——

// Launch 启动 Claude Code。
// 优先用 `claude` 命令，找不到再 fallback 到常见路径。
func (c *ClaudeAdapter) Launch(cwd string, args []string) error {
	bin, err := exec.LookPath("claude")
	if err != nil {
		bin = c.findBinary()
		if bin == "" {
			return fmt.Errorf("claude CLI not found in PATH or common locations")
		}
	}
	cmd := exec.Command(bin, args...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd.Start()
}

// Kill 终止 Claude Code 进程。
func (c *ClaudeAdapter) Kill(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(syscall.SIGTERM)
}

// findBinary 在常见路径中查找 claude 可执行文件。
func (c *ClaudeAdapter) findBinary() string {
	candidates := []string{
		filepath.Join(c.home, ".local", "bin", "claude"),
		"/usr/local/bin/claude",
		"/opt/homebrew/bin/claude",
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

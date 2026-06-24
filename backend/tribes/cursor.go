package tribes

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// CursorAdapter 适配 Cursor IDE 的 CLI（`cursor` command）。
// 进程检测匹配 cursor 主进程。
type CursorAdapter struct {
	home string
}

func NewCursorAdapter(home string) *CursorAdapter {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return &CursorAdapter{home: home}
}

// —— Identity ——

func (c *CursorAdapter) ID() string      { return "cursor" }
func (c *CursorAdapter) Name() string    { return "Cursor" }
// EcoType: "ide" 表示这是 IDE 应用，不是 CLI agent。
// 走另一套 UI（workspace focused），不假装有 Session/Token。
func (c *CursorAdapter) EcoType() string { return "ide" }

// ThemeColor 银蓝玻璃穹顶。
func (c *CursorAdapter) ThemeColor() string  { return "#60a5fa" }
func (c *CursorAdapter) AccentColor() string { return "#a5d8ff" }

// Capabilities Cursor 是 IDE，没有"会话"概念——老实说没有。
// 后续可以加 Extensions / Workspaces 等 IDE 特性。
func (c *CursorAdapter) Capabilities() Capabilities {
	return Capabilities{
		Process:     true,
		Launch:      true,
		Config:      true,
		ConfigEdit:  false, // M2+
		Sessions:    false, // IDE 没有会话
		SessionTail: false,
		Tokens:      false, // 不暴露 token（IDE 内部成本不可见）
		TokensLive:  false,

		Features: []Feature{
			{ID: FeatureExtensions, Label: "Extensions", Description: "Installed Cursor extensions", HasData: false},
			{ID: FeatureWorkspaces, Label: "Recent Workspaces", Description: "Recently opened folders", HasData: false},
		},
	}
}

// —— Reader ——

func (c *CursorAdapter) ConfigPaths() []string {
	return []string{
		filepath.Join(c.home, ".cursor", "settings.json"),
		filepath.Join(c.home, ".config", "Cursor", "User", "settings.json"),
		filepath.Join(c.home, "Library", "Application Support", "Cursor", "User", "settings.json"),
	}
}

func (c *CursorAdapter) ParseConfig() (map[string]any, error) {
	for _, p := range c.ConfigPaths() {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		out := map[string]any{}
		if err := json.Unmarshal(data, &out); err != nil {
			return nil, err
		}
		return out, nil
	}
	return map[string]any{}, nil
}

// ParseConfigDNA Cursor 的 settings.json 是用户级 IDE 配置，
// 没有"敏感 vs 运行时"这种分层语义——所有项都进 deep 层。
func (c *CursorAdapter) ParseConfigDNA() (ConfigDNA, error) {
	raw, err := c.ParseConfig()
	if err != nil {
		return ConfigDNA{}, err
	}
	dna := ConfigDNA{Source: "cursor"}
	for k, v := range raw {
		typ := inferType(v)
		dna.Deep = append(dna.Deep, ConfigItem{Key: k, Value: v, Type: typ, Layer: "deep"})
		dna.Schema.Fields = append(dna.Schema.Fields, ConfigField{
			Key: k, Type: typ, Editable: false, // IDE 暂不支持编辑
		})
	}
	return dna, nil
}

// —— Detector ——

func (c *CursorAdapter) DetectProcess() (*ProcessInfo, error) {
	cmd := exec.Command("ps", "-axo", "pid=,comm=,args=")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	for _, line := range bytes.Split(out, []byte("\n")) {
		s := strings.TrimSpace(string(line))
		lower := strings.ToLower(s)
		if !strings.Contains(lower, "cursor") {
			continue
		}
		// 跳过 ps 自身和 electron 框架
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
		// Cursor 是 Electron 应用，进程名是 "Cursor" 或 "Electron"
		if !strings.Contains(lower, "cursor") || strings.Contains(lower, "cursor helper") {
			// 接受所有 cursor 进程
		}
		info := &ProcessInfo{PID: pid, Name: s, CmdLine: s}
		if stats, err := readProcStats(pid); err == nil {
			info.CPU = stats.cpu
			info.Memory = stats.memMB
			info.StartTime = stats.startUnix
		}
		if cwd, err := readProcessCWD(pid); err == nil {
			info.CWD = cwd
		}
		return info, nil
	}
	return nil, nil
}

// —— Launcher ——

func (c *CursorAdapter) Launch(cwd string, args []string) error {
	bin, err := exec.LookPath("cursor")
	if err != nil {
		// Cursor 标准安装路径
		candidates := []string{
			"/usr/local/bin/cursor",
			"/Applications/Cursor.app/Contents/Resources/app/bin/cursor",
		}
		for _, p := range candidates {
			if _, statErr := os.Stat(p); statErr == nil {
				bin = p
				break
			}
		}
		if bin == "" {
			// 退而求其次：open -a Cursor
			args := []string{"-a", "Cursor"}
			if cwd != "" {
				args = append(args, cwd)
			}
			return exec.Command("open", args...).Start()
		}
	}
	cmd := exec.Command(bin, args...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd.Start()
}

func (c *CursorAdapter) Kill(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(syscall.SIGTERM)
}

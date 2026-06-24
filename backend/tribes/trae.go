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

// TraeAdapter 适配 Trae IDE。
type TraeAdapter struct {
	home string
}

func NewTraeAdapter(home string) *TraeAdapter {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return &TraeAdapter{home: home}
}

// —— Identity ——

func (t *TraeAdapter) ID() string      { return "trae" }
func (t *TraeAdapter) Name() string    { return "Trae" }
// EcoType: "ide" 同 Cursor 一样走 IDE 面板。
func (t *TraeAdapter) EcoType() string { return "ide" }

// ThemeColor 翠竹。
func (t *TraeAdapter) ThemeColor() string  { return "#4ade80" }
func (t *TraeAdapter) AccentColor() string { return "#86efac" }

// Capabilities Trae 同 Cursor：IDE，不假装有 Session/Token。
func (t *TraeAdapter) Capabilities() Capabilities {
	return Capabilities{
		Process:     true,
		Launch:      true,
		Config:      true,
		ConfigEdit:  false,
		Sessions:    false,
		SessionTail: false,
		Tokens:      false,
		TokensLive:  false,

		Features: []Feature{
			{ID: FeatureExtensions, Label: "Extensions", Description: "Installed Trae extensions", HasData: false},
		},
	}
}

// —— Reader ——

func (t *TraeAdapter) ConfigPaths() []string {
	return []string{
		filepath.Join(t.home, ".trae", "config.json"),
		filepath.Join(t.home, "Library", "Application Support", "Trae", "User", "settings.json"),
	}
}

func (t *TraeAdapter) ParseConfig() (map[string]any, error) {
	for _, p := range t.ConfigPaths() {
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

// ParseConfigDNA 同 Cursor：所有项进 deep。
func (t *TraeAdapter) ParseConfigDNA() (ConfigDNA, error) {
	raw, err := t.ParseConfig()
	if err != nil {
		return ConfigDNA{}, err
	}
	dna := ConfigDNA{Source: "trae"}
	for k, v := range raw {
		typ := inferType(v)
		dna.Deep = append(dna.Deep, ConfigItem{Key: k, Value: v, Type: typ, Layer: "deep"})
		dna.Schema.Fields = append(dna.Schema.Fields, ConfigField{
			Key: k, Type: typ, Editable: false,
		})
	}
	return dna, nil
}

// —— Detector ——

func (t *TraeAdapter) DetectProcess() (*ProcessInfo, error) {
	cmd := exec.Command("ps", "-axo", "pid=,comm=,args=")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	for _, line := range bytes.Split(out, []byte("\n")) {
		s := strings.TrimSpace(string(line))
		lower := strings.ToLower(s)
		if !strings.Contains(lower, "trae") {
			continue
		}
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

func (t *TraeAdapter) Launch(cwd string, args []string) error {
	bin, err := exec.LookPath("trae")
	if err != nil {
		candidates := []string{
			"/usr/local/bin/trae",
			"/Applications/Trae.app/Contents/Resources/app/bin/trae",
		}
		for _, p := range candidates {
			if _, statErr := os.Stat(p); statErr == nil {
				bin = p
				break
			}
		}
		if bin == "" {
			args := []string{"-a", "Trae"}
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

func (t *TraeAdapter) Kill(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(syscall.SIGTERM)
}

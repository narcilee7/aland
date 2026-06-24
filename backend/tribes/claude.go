package tribes

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/narcilee7/aland/backend/core"
)

// ClaudeAdapter 适配 Claude Code CLI。
// 它是一个具体的实现，结构性满足 Identity/Detector/Launcher/Reader/
// SessionLister/TokenStatReader/MCPServerLister/.../SessionStreamer ——
// 不需要显式声明接口实现。
type ClaudeAdapter struct {
	home string

	// streamMu / streamCancel 保护实时 tail 状态。
	streamMu     sync.Mutex
	streamCancel context.CancelFunc
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

// Capabilities 报家门：Claude Code 是 A 级 CLI agent，19 项全做透。
func (c *ClaudeAdapter) Capabilities() Capabilities {
	return Capabilities{
		Process:     true,
		Launch:      true,
		Config:      true,
		ConfigEdit:  true,
		Sessions:    true,
		SessionTail: true, // M2 杀手锏
		Tokens:      true,
		TokensLive:  true, // 实时增量：来自 session stream 的 SessionEvent.Tokens

		Features: []Feature{
			{ID: FeatureMCPServers, Label: "MCP Servers", Description: "Model Context Protocol servers configured for this CLI", HasData: true},
			{ID: FeatureSkills, Label: "Skills", Description: "Custom slash commands and skills loaded", HasData: true},
			{ID: FeaturePlugins, Label: "Plugins", Description: "Enabled plugins (LSP, hooks, etc.)", HasData: true},
			{ID: FeaturePlans, Label: "Plan Files", Description: "Plan mode artifacts", HasData: true},
			{ID: FeatureFileHistory, Label: "File History", Description: "Tracked file backups and edits", HasData: true},
		},
	}
}

// —— 内部 helper ——

// claudeHome 返回 ~/.claude。
func (c *ClaudeAdapter) claudeHome() string {
	return filepath.Join(c.home, ".claude")
}

// projectsDir 返回 ~/.claude/projects。
func (c *ClaudeAdapter) projectsDir() string {
	return filepath.Join(c.claudeHome(), "projects")
}

// —— Reader ——

// ConfigPaths 返回 Claude Code 配置目录下的关键文件。
func (c *ClaudeAdapter) ConfigPaths() []string {
	return []string{
		filepath.Join(c.claudeHome(), "settings.json"),
		filepath.Join(c.claudeHome(), "stats-cache.json"),
		c.claudeHome(),
	}
}

// ParseConfig 解析 settings.json。
// 失败时返回空 map（CLI 未安装也属于正常情况，不算错误）。
func (c *ClaudeAdapter) ParseConfig() (map[string]any, error) {
	data, err := os.ReadFile(filepath.Join(c.claudeHome(), "settings.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	out := map[string]any{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// —— Detector ——

// DetectProcess 扫描系统中正在运行的 claude 进程。
// 拿真实 PID/CPU/内存/CWD/启动时间。
//
// 匹配规则：必须 comm 字段等于 "claude"（实际二进制名），不只匹配 cmdline 子串。
// 否则 zsh 这种跑 ~/.claude/... 路径的 shell 会被误中。
// 找不到返回 (nil, nil)，不算错误。
func (c *ClaudeAdapter) DetectProcess() (*ProcessInfo, error) {
	// 1. 找 PID + comm（真实进程名）+ 命令行
	cmd := exec.Command("ps", "-axo", "pid=,comm=,args=")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var pid int
	var comm string
	var cmdline string
	for _, line := range bytes.Split(out, []byte("\n")) {
		s := strings.TrimSpace(string(line))
		if s == "" {
			continue
		}
		// ps 输出格式：PID COMM ARGS
		// COMM 不能包含空格；ARGS 可包含
		// 第一个 whitespace-separated token 是 PID
		// 找到第一个空格后第一个 token 是 COMM（截至下一个 ARG 起始）
		// 因为 COMM 在 ps 里默认是截断的（不超过 16 字节），简化处理：
		// 取前 2 个 whitespace-separated 字段做 pid + comm
		fields := strings.SplitN(s, " ", 3)
		if len(fields) < 2 {
			continue
		}
		// COMM 必须正好是 "claude"（精确匹配）
		if fields[1] != "claude" {
			continue
		}
		p, err := strconv.Atoi(strings.TrimSpace(fields[0]))
		if err != nil || p == 0 {
			continue
		}
		pid = p
		comm = fields[1]
		if len(fields) > 2 {
			cmdline = fields[2]
		} else {
			cmdline = comm
		}
		break
	}
	if pid == 0 {
		return nil, nil
	}

	// 2. 拿真实 CPU/内存/启动时间
	info := &ProcessInfo{PID: pid, Name: comm, CmdLine: cmdline}
	if stats, err := readProcStats(pid); err == nil {
		info.CPU = stats.cpu
		info.Memory = stats.memMB
		info.StartTime = stats.startUnix
	}

	// 3. 拿 CWD（macOS 用 lsof，Linux 用 /proc）
	if cwd, err := readProcessCWD(pid); err == nil {
		info.CWD = cwd
	}

	return info, nil
}

// procStats 是 ps 输出的单行解析结果。
type procStats struct {
	cpu       float64 // 0-100（单核），多核会 >100
	memMB     float64
	startUnix int64
}

// readProcStats 用 `ps -p <pid> -o %cpu,%mem,rss,etime,start` 读出实时指标。
// 失败返回错误（不致命，ProcessInfo 仍能返回）。
func readProcStats(pid int) (*procStats, error) {
	out, err := exec.Command(
		"ps", "-p", strconv.Itoa(pid),
		"-o", "pcpu=,pmem=,rss=,etime=,start=",
	).Output()
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) < 5 {
		return nil, fmt.Errorf("ps output too short: %q", string(out))
	}
	cpu, _ := strconv.ParseFloat(fields[0], 64)
	_, _ = strconv.ParseFloat(fields[1], 64) // pmem 也存下，未来用
	rssKB, _ := strconv.ParseFloat(fields[2], 64) // KB
	etime := fields[3]                              // e.g. "01:23:45" or "5-12:34:56"
	startStr := fields[4]                           // e.g. "10:30AM" or "Jun24"

	return &procStats{
		cpu:       cpu,
		memMB:     rssKB / 1024,
		startUnix: parseStartTime(etime, startStr),
	}, nil
}

// parseStartTime 把 ps 的 etime / start 转 unix 秒。
// etime: 进程已运行时长，倒推出 start。
// 简化：只精确到分钟。
func parseStartTime(etime, _ string) int64 {
	d, err := parseDuration(etime)
	if err != nil {
		return 0
	}
	return time.Now().Add(-d).Unix()
}

// parseDuration 解析 "5-12:34:56" / "12:34:56" / "34:56" / "56" 形式。
func parseDuration(s string) (time.Duration, error) {
	// 5-12:34:56 形式
	if idx := strings.Index(s, "-"); idx >= 0 {
		days, _ := strconv.Atoi(s[:idx])
		rest, err := parseDuration(s[idx+1:])
		if err != nil {
			return 0, err
		}
		return time.Duration(days)*24*time.Hour + rest, nil
	}
	parts := strings.Split(s, ":")
	switch len(parts) {
	case 3:
		h, _ := strconv.Atoi(parts[0])
		m, _ := strconv.Atoi(parts[1])
		sec, _ := strconv.Atoi(parts[2])
		return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(sec)*time.Second, nil
	case 2:
		m, _ := strconv.Atoi(parts[0])
		sec, _ := strconv.Atoi(parts[1])
		return time.Duration(m)*time.Minute + time.Duration(sec)*time.Second, nil
	case 1:
		sec, _ := strconv.Atoi(parts[0])
		return time.Duration(sec) * time.Second, nil
	}
	return 0, fmt.Errorf("invalid duration: %s", s)
}

// readProcessCWD 读进程的 CWD。macOS 走 lsof，Linux 走 /proc。
func readProcessCWD(pid int) (string, error) {
	out, err := exec.Command("lsof", "-a", "-p", strconv.Itoa(pid), "-d", "cwd", "-Fn").Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "n") {
				return strings.TrimPrefix(line, "n"), nil
			}
		}
	}
	// fallback: /proc/<pid>/cwd
	cwd, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid))
	if err != nil {
		return "", err
	}
	return cwd, nil
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

// —— SessionLister（设计 doc 提到的 Memory Shards）——

// ParseSessions 列出 ~/.claude/projects/*/*.jsonl 会话，提取摘要。
// 摘要规则：找 "ai-title" 事件（AI 自动给会话起的标题），找不到就用首条 user 消息。
func (c *ClaudeAdapter) ParseSessions() ([]SessionShard, error) {
	dir := c.projectsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var shards []SessionShard
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		projectPath, err := decodeProjectPath(e.Name())
		if err != nil {
			continue
		}
		projectDir := filepath.Join(dir, e.Name())
		files, err := os.ReadDir(projectDir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".jsonl") {
				continue
			}
			shard, err := parseSessionFile(filepath.Join(projectDir, f.Name()), projectPath)
			if err != nil {
				continue
			}
			shards = append(shards, shard)
		}
	}
	return shards, nil
}

// parseSessionFile 读一个 .jsonl，提取前若干事件构造 SessionShard。
// 性能优化：只读前 1MB；超过就截断。
func parseSessionFile(path, projectPath string) (SessionShard, error) {
	f, err := os.Open(path)
	if err != nil {
		return SessionShard{}, err
	}
	defer f.Close()

	stat, _ := f.Stat()
	s := SessionShard{
		ID:        strings.TrimSuffix(filepath.Base(path), ".jsonl"),
		Tribe:     "claude",
		Project:   projectPath,
		FilePath:  path,
		SizeBytes: stat.Size(),
	}

	br := bufio.NewReader(io.LimitReader(f, 1<<20)) // 1MB
	var firstUserMsg string
	var tokenIn, tokenOut int64
	enc := json.NewDecoder(br)
	for enc.More() {
		var ev map[string]any
		if err := enc.Decode(&ev); err != nil {
			break
		}
		t, _ := ev["type"].(string)

		switch t {
		case "ai-title":
			if title, _ := ev["aiTitle"].(string); title != "" {
				s.Summary = title
			}
		case "user":
			if s.Summary == "" {
				// 第一次 user 消息作为 fallback 摘要
				if msg, ok := ev["message"].(map[string]any); ok {
					if content, ok := msg["content"].(string); ok {
						firstUserMsg = truncate(content, 80)
					}
				}
			}
			if ts, ok := ev["timestamp"].(string); ok {
				if t, err := time.Parse(time.RFC3339Nano, ts); err == nil && s.Timestamp == 0 {
					s.Timestamp = t.UnixMilli()
				}
			}
			if cwd, _ := ev["cwd"].(string); cwd != "" {
				s.CWD = cwd
			}
			s.MessageCount++
		case "assistant":
			s.MessageCount++
			if msg, ok := ev["message"].(map[string]any); ok {
				if usage, ok := msg["usage"].(map[string]any); ok {
					if v, ok := usage["input_tokens"].(float64); ok {
						tokenIn += int64(v)
					}
					if v, ok := usage["output_tokens"].(float64); ok {
						tokenOut += int64(v)
					}
				}
				if model, _ := msg["model"].(string); model != "" {
					s.Model = model
				}
			}
		}
	}
	if s.Summary == "" {
		s.Summary = firstUserMsg
	}
	if s.Summary == "" {
		s.Summary = "(no summary)"
	}
	s.TokenCount = tokenIn + tokenOut
	return s, nil
}

// decodeProjectPath 把 "Users-foo-bar" 还原成 "/Users/foo/bar" 形式。
// 实际 Claude 编码规则：cwd 里的 "/" 替换成 "-"。所以 "-Users-foo-bar" → "/Users/foo/bar"。
func decodeProjectPath(encoded string) (string, error) {
	trimmed := strings.TrimPrefix(encoded, "-")
	return "/" + strings.ReplaceAll(trimmed, "-", "/"), nil
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// —— TokenStatReader ——

// ParseTokenUsage 从 ~/.claude/stats-cache.json + 遍历 session 估算 Token 消耗。
// stats-cache 提供每日 message/session/tool 计数；
// session 里的 assistant.usage 给出真实 token 数；
// 按 model 价格表算 USD（不在表里返回 0，不瞎算）。
func (c *ClaudeAdapter) ParseTokenUsage() (*TokenUsage, error) {
	usages, err := c.ModelTokenUsage()
	if err != nil {
		return nil, err
	}
	if len(usages) == 0 {
		return &TokenUsage{}, nil
	}

	var inTok, outTok, cacheRead, cacheWrite int64
	var cost float64
	modelSet := map[string]bool{}
	hasKnownPrice := false
	for _, u := range usages {
		inTok += u.InputTokens
		outTok += u.OutputTokens
		cacheRead += u.CacheRead
		cacheWrite += u.CacheWrite
		c, ok := CostFromUsage(u.Model, u.InputTokens, u.OutputTokens, u.CacheRead, u.CacheWrite)
		if ok {
			cost += c
			hasKnownPrice = true
		}
		if u.Model != "" {
			modelSet[u.Model] = true
		}
	}
	// 主要 model（消耗最多的）
	var model string
	max := int64(0)
	for _, u := range usages {
		if u.InputTokens+u.OutputTokens > max {
			max = u.InputTokens + u.OutputTokens
			model = u.Model
		}
	}
	_ = modelSet
	result := &TokenUsage{
		InputTokens:  inTok,
		OutputTokens: outTok,
		Model:        model,
	}
	// 只在有已知价格时填 CostUSD；否则留 0 让前端显示 "n/a"
	if hasKnownPrice {
		result.CostUSD = cost
	}
	return result, nil
}

// ListPlugins 从 settings.enabledPlugins 读。
// 未来：扫 ~/.claude/plugins/ 找本地安装的。
func (c *ClaudeAdapter) ListPlugins() ([]Plugin, error) {
	data, err := os.ReadFile(filepath.Join(c.claudeHome(), "settings.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	enabled, ok := raw["enabledPlugins"].(map[string]any)
	if !ok {
		return nil, nil
	}
	var out []Plugin
	for name, v := range enabled {
		enabled := false
		if b, ok := v.(bool); ok {
			enabled = b
		}
		out = append(out, Plugin{
			Name:    name,
			Enabled: enabled,
			Source:  "settings.json",
		})
	}
	return out, nil
}

// —— ConfigParser（设计 doc 提到的 Config DNA / 三层）——

// ParseConfigDNA 解析 settings.json，按 key 前缀分到三层。
// 同时构建 Schema，让前端能 schema-driven 渲染。
//   - surface: model / 运行时
//   - middle:  permissions / API / TOKEN
//   - deep:    其他（plugins、统计等元信息）
func (c *ClaudeAdapter) ParseConfigDNA() (ConfigDNA, error) {
	raw, err := c.ParseConfig()
	if err != nil {
		return ConfigDNA{}, err
	}
	dna := ConfigDNA{Source: filepath.Join(c.claudeHome(), "settings.json")}

	// schema 字段的元信息表
	fieldMeta := map[string]ConfigField{
		"model": {
			Key: "model", Label: "Default Model",
			Description: "Model ID used for new sessions",
			Type: "string", Editable: true,
		},
		"env": {
			Key: "env", Label: "Environment Variables",
			Description: "Custom env vars passed to Claude (includes API keys)",
			Type: "json", Editable: true,
		},
		"enabledPlugins": {
			Key: "enabledPlugins", Label: "Enabled Plugins",
			Description: "Map of plugin name to enabled state",
			Type: "json", Editable: true,
		},
		"permissions": {
			Key: "permissions", Label: "Permissions",
			Description: "Tool and file access permissions",
			Type: "json", Editable: true,
		},
	}

	for k, v := range raw {
		typ := inferType(v)
		meta, hasMeta := fieldMeta[k]
		if !hasMeta {
			meta = ConfigField{Key: k, Type: typ, Editable: true}
		} else {
			meta.Type = typ
		}

		switch {
		case strings.HasPrefix(k, "model"):
			dna.Surface = append(dna.Surface, ConfigItem{Key: k, Value: v, Type: typ, Layer: "surface"})
		case strings.HasPrefix(k, "permission"), k == "env",
			strings.Contains(k, "API"), strings.Contains(k, "TOKEN"), strings.Contains(k, "AUTH"):
			meta.Sensitive = true
			meta.Type = "secret"
			dna.Middle = append(dna.Middle, ConfigItem{Key: k, Value: v, Type: typ, Layer: "middle", Sensitive: true})
		default:
			dna.Deep = append(dna.Deep, ConfigItem{Key: k, Value: v, Type: typ, Layer: "deep"})
		}
		dna.Schema.Fields = append(dna.Schema.Fields, meta)
	}
	return dna, nil
}

// WriteConfig 把 ConfigDNA 写回 settings.json。
// 修改前自动备份到 ~/.aland/backups/settings-<unix>.json。
// 原子写：先写临时文件，再 rename，避免半写状态。
func (c *ClaudeAdapter) WriteConfig(dna ConfigDNA) error {
	src := filepath.Join(c.claudeHome(), "settings.json")

	// 1. 读现有 raw（保留未在 dna 里出现的 key）
	existing := map[string]any{}
	if data, err := os.ReadFile(src); err == nil {
		_ = json.Unmarshal(data, &existing)
	}

	// 2. 合并 dna 的三层回 existing
	for _, item := range dna.Surface {
		existing[item.Key] = item.Value
	}
	for _, item := range dna.Middle {
		existing[item.Key] = item.Value
	}
	for _, item := range dna.Deep {
		existing[item.Key] = item.Value
	}

	// 3. 备份当前
	if err := c.backupConfig(src); err != nil {
		core.Log.Warn("config backup failed", "err", err, "path", src)
		// 备份失败不阻塞写——但要 log
	}

	// 4. 原子写
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err
	}
	tmp := src + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, src)
}

// backupConfig 复制文件到 ~/.aland/backups/<basename>-<unix>.json。
func (c *ClaudeAdapter) backupConfig(src string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".aland", "backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	base := filepath.Base(src)
	stamp := time.Now().Unix()
	dst := filepath.Join(dir, fmt.Sprintf("%s-%d", base, stamp))

	data, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.WriteFile(dst, data, 0o600)
}

func inferType(v any) string {
	switch v.(type) {
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64:
		return "number"
	default:
		return "json"
	}
}

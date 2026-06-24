// Claude Code 19 项能力实现——Insights 部分。
//
// 包含：MCPServers / Skills / Plans / FileHistory / Stats / SessionRead / SessionStream。
// claude.go 里是基础（Process / Launch / Reader / Writer / Sessions / Tokens）。

package tribes

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/narcilee7/aland/backend/core"
)

// —— MCPServers ——

// ListMCPServers 从 ~/.claude/.mcp.json + settings.json 的 mcpServers 合并。
func (c *ClaudeAdapter) ListMCPServers() ([]MCPServer, error) {
	var out []MCPServer

	// 1. .mcp.json（项目级 / 用户级）
	for _, p := range []string{
		filepath.Join(c.home, ".claude", ".mcp.json"),
	} {
		if data, err := os.ReadFile(p); err == nil {
			var mcpFile struct {
				Servers map[string]struct {
					Command   string            `json:"command"`
					Args      []string          `json:"args"`
					Env       map[string]string `json:"env"`
					Transport string            `json:"type"`
				} `json:"mcpServers"`
			}
			if err := json.Unmarshal(data, &mcpFile); err == nil {
				for name, s := range mcpFile.Servers {
					out = append(out, MCPServer{
						Name:      name,
						Command:   s.Command,
						Args:      s.Args,
						Env:      s.Env,
						Transport: s.Transport,
						Source:    p,
						Enabled:   true,
					})
				}
			}
		}
	}

	// 2. settings.json 的 mcpServers
	settingsPath := filepath.Join(c.claudeHome(), "settings.json")
	if data, err := os.ReadFile(settingsPath); err == nil {
		var raw map[string]any
		if err := json.Unmarshal(data, &raw); err == nil {
			if mcp, ok := raw["mcpServers"].(map[string]any); ok {
				for name, v := range mcp {
					if vm, ok := v.(map[string]any); ok {
						srv := MCPServer{
							Name:    name,
							Source:  settingsPath,
							Enabled: true,
						}
						if cmd, _ := vm["command"].(string); cmd != "" {
							srv.Command = cmd
						}
						if args, ok := vm["args"].([]any); ok {
							for _, a := range args {
								if s, ok := a.(string); ok {
									srv.Args = append(srv.Args, s)
								}
							}
						}
						if t, _ := vm["type"].(string); t != "" {
							srv.Transport = t
						}
						out = append(out, srv)
					}
				}
			}
		}
	}

	// 去重（name 相同保留第一个）
	seen := map[string]bool{}
	dedup := out[:0]
	for _, s := range out {
		if seen[s.Name] {
			continue
		}
		seen[s.Name] = true
		dedup = append(dedup, s)
	}
	return dedup, nil
}

// —— Skills ——

// ListSkills 扫 ~/.claude/skills/*/SKILL.md。
// SKILL.md 是 frontmatter 格式（YAML），解析首段 --- 里的 name/description。
func (c *ClaudeAdapter) ListSkills() ([]Skill, error) {
	dir := filepath.Join(c.claudeHome(), "skills")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var skills []Skill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillPath := filepath.Join(dir, e.Name(), "SKILL.md")
		data, err := os.ReadFile(skillPath)
		if err != nil {
			continue
		}
		content := string(data)
		name, description := parseFrontmatter(content)
		if name == "" {
			name = e.Name()
		}
		skills = append(skills, Skill{
			Name:        name,
			Description: description,
			Path:        skillPath,
			Content:     content,
		})
	}
	return skills, nil
}

// parseFrontmatter 从 "---\nname: x\ndescription: y\n---\n..." 抽 name/description。
// 不引完整 YAML 库——只取前两个字段。
func parseFrontmatter(content string) (name, description string) {
	if !strings.HasPrefix(content, "---") {
		return
	}
	end := strings.Index(content[3:], "\n---")
	if end < 0 {
		return
	}
	fm := content[3 : 3+end]
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "name:"):
			name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			// 去引号
			name = strings.Trim(name, "\"'")
		case strings.HasPrefix(line, "description:"):
			description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			description = strings.Trim(description, "\"'")
		}
	}
	return
}

// —— Plans ——

// ListPlans 扫 ~/.claude/plans/*.md。
func (c *ClaudeAdapter) ListPlans() ([]PlanFile, error) {
	dir := filepath.Join(c.claudeHome(), "plans")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var plans []PlanFile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		path := filepath.Join(dir, e.Name())
		// 摘要：读前 200 字符
		summary := ""
		if data, err := os.ReadFile(path); err == nil {
			s := string(data)
			r := []rune(s)
			if len(r) > 200 {
				summary = string(r[:200]) + "…"
			} else {
				summary = s
			}
		}
		plans = append(plans, PlanFile{
			Name:       strings.TrimSuffix(e.Name(), ".md"),
			Path:       path,
			Size:       info.Size(),
			ModifiedAt: info.ModTime().Unix(),
			Summary:    summary,
		})
	}
	// 按修改时间倒序
	sort.Slice(plans, func(i, j int) bool { return plans[i].ModifiedAt > plans[j].ModifiedAt })
	return plans, nil
}

// —— FileHistory ——

// ListFileHistory 扫 ~/.claude/file-history/。
// 目录结构是 <hash>/<file>，每改一个文件，递增版本号目录。
func (c *ClaudeAdapter) ListFileHistory() ([]FileEdit, error) {
	dir := filepath.Join(c.claudeHome(), "file-history")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var edits []FileEdit
	for _, hashDir := range entries {
		if !hashDir.IsDir() {
			continue
		}
		hash := hashDir.Name()
		// 在 hashDir 下，每个 <file>@<ts> 是个 backup
		files, err := os.ReadDir(filepath.Join(dir, hash))
		if err != nil {
			continue
		}
		version := 0
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			version++
			info, _ := f.Info()
			edits = append(edits, FileEdit{
				Path:         f.Name(),
				BackupPath:   filepath.Join(dir, hash, f.Name()),
				Timestamp:    info.ModTime().Unix(),
				OriginalHash: hash,
				Version:      version,
			})
		}
	}
	// 按时间倒序
	sort.Slice(edits, func(i, j int) bool { return edits[i].Timestamp > edits[j].Timestamp })
	// 截到最近 200 条
	if len(edits) > 200 {
		edits = edits[:200]
	}
	return edits, nil
}

// RestoreFile 把 backup 复制回原文件路径。
// 注意：Claude 编码 backup 路径不包含原 cwd——文件名是 `<encoded-cwd>--<rest>`。
// 这里只做文件复制，安全由用户自己负责。
func (c *ClaudeAdapter) RestoreFile(edit FileEdit) error {
	data, err := os.ReadFile(edit.BackupPath)
	if err != nil {
		return err
	}
	// 写回前再备份一次
	home, _ := os.UserHomeDir()
	backupDir := filepath.Join(home, ".aland", "backups", "restore-prevention")
	_ = os.MkdirAll(backupDir, 0o755)
	if _, err := os.Stat(edit.Path); err == nil {
		stamp := time.Now().Unix()
		_ = os.WriteFile(
			filepath.Join(backupDir, fmt.Sprintf("%s-%d", filepath.Base(edit.Path), stamp)),
			data, 0o600,
		)
	}
	return os.WriteFile(edit.Path, data, 0o600)
}

// —— Stats ——

// DailyActivity 读 ~/.claude/stats-cache.json。
func (c *ClaudeAdapter) DailyActivity() ([]DailyActivity, error) {
	data, err := os.ReadFile(filepath.Join(c.claudeHome(), "stats-cache.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var raw struct {
		DailyActivity []struct {
			Date          string `json:"date"`
			MessageCount  int    `json:"messageCount"`
			SessionCount  int    `json:"sessionCount"`
			ToolCallCount int    `json:"toolCallCount"`
		} `json:"dailyActivity"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	out := make([]DailyActivity, 0, len(raw.DailyActivity))
	for _, d := range raw.DailyActivity {
		out = append(out, DailyActivity{
			Date: d.Date, MessageCount: d.MessageCount,
			SessionCount: d.SessionCount, ToolCallCount: d.ToolCallCount,
		})
	}
	return out, nil
}

// ModelTokenUsage 统计所有 session 里的 assistant.usage，按 (model, date) 聚合。
func (c *ClaudeAdapter) ModelTokenUsage() ([]ModelTokenUsage, error) {
	dir := c.projectsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	agg := map[modelDateKey]*ModelTokenUsage{}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		files, _ := os.ReadDir(filepath.Join(dir, e.Name()))
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".jsonl") {
				continue
			}
			scanSessionTokens(filepath.Join(dir, e.Name(), f.Name()), agg)
		}
	}
	out := make([]ModelTokenUsage, 0, len(agg))
	for _, v := range agg {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date > out[j].Date })
	return out, nil
}

type modelDateKey struct{ model, date string }

func scanSessionTokens(path string, agg map[modelDateKey]*ModelTokenUsage) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		var ev map[string]any
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue
		}
		if t, _ := ev["type"].(string); t != "assistant" {
			continue
		}
		msg, _ := ev["message"].(map[string]any)
		if msg == nil {
			continue
		}
		usage, _ := msg["usage"].(map[string]any)
		if usage == nil {
			continue
		}
		model, _ := msg["model"].(string)
		ts, _ := ev["timestamp"].(string)
		date := ts[:10] // YYYY-MM-DD

		k := modelDateKey{model, date}
		v, ok := agg[k]
		if !ok {
			v = &ModelTokenUsage{Model: model, Date: date}
			agg[k] = v
		}
		if in, ok := usage["input_tokens"].(float64); ok {
			v.InputTokens += int64(in)
		}
		if out, ok := usage["output_tokens"].(float64); ok {
			v.OutputTokens += int64(out)
		}
		if cr, ok := usage["cache_read_input_tokens"].(float64); ok {
			v.CacheRead += int64(cr)
		}
		if cw, ok := usage["cache_creation_input_tokens"].(float64); ok {
			v.CacheWrite += int64(cw)
		}
	}
}

// RecentSlashCommands 读 ~/.claude/history.jsonl，最近 n 条 slash command。
func (c *ClaudeAdapter) RecentSlashCommands(n int) ([]SlashCommand, error) {
	path := filepath.Join(c.claudeHome(), "history.jsonl")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var all []SlashCommand
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		var ev map[string]any
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue
		}
		// history.jsonl 格式：{"display":"...","pastedContents":{},"timestamp":..., "project":..., "sessionId":...}
		display, _ := ev["display"].(string)
		if !strings.HasPrefix(display, "/") {
			continue
		}
		ts, _ := ev["timestamp"].(float64)
		cwd, _ := ev["project"].(string)
		// 分离 command 和 args
		cmd, args := display, ""
		if idx := strings.IndexAny(display, " \n"); idx > 0 {
			cmd = display[:idx]
			args = strings.TrimSpace(display[idx+1:])
		}
		all = append(all, SlashCommand{
			Command: cmd, Args: args,
			Timestamp: int64(ts), CWD: cwd,
		})
	}
	// 取最近 n 条
	sort.Slice(all, func(i, j int) bool { return all[i].Timestamp > all[j].Timestamp })
	if len(all) > n {
		all = all[:n]
	}
	return all, nil
}

// —— SessionRead ——

// ReadSession 读完整 session jsonl 返回 []SessionEvent。
func (c *ClaudeAdapter) ReadSession(id string) ([]SessionEvent, error) {
	path := c.findSessionPath(id)
	if path == "" {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	return parseSessionFileFull(path)
}

// findSessionPath 在 projects/*/*.jsonl 里找 ID 匹配的文件。
func (c *ClaudeAdapter) findSessionPath(id string) string {
	dir := c.projectsDir()
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		candidate := filepath.Join(dir, e.Name(), id+".jsonl")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

// parseSessionFileFull 解析整个 session 为 []SessionEvent。
// 大文件也只解析（不过滤大小），让前端决定怎么显示。
func parseSessionFileFull(path string) ([]SessionEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var events []SessionEvent
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 4<<20) // 单行 4MB
	for sc.Scan() {
		var ev map[string]any
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue
		}
		events = append(events, normalizeSessionEvent(ev))
	}
	return events, nil
}

func normalizeSessionEvent(raw map[string]any) SessionEvent {
	ev := SessionEvent{}
	if t, _ := raw["type"].(string); t != "" {
		ev.Type = t
	}
	if s, _ := raw["subtype"].(string); s != "" {
		ev.Subtype = s
	}
	if ts, _ := raw["timestamp"].(string); ts != "" {
		if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			ev.Timestamp = t.UnixMilli()
		}
	}

	// user / assistant 消息
	if msg, ok := raw["message"].(map[string]any); ok {
		if role, _ := msg["role"].(string); role != "" {
			ev.Role = role
		}
		if model, _ := msg["model"].(string); model != "" {
			ev.Model = model
		}
		// 提取 content + thinking
		if content, ok := msg["content"].([]any); ok {
			var texts, thinks []string
			for _, c := range content {
				cm, _ := c.(map[string]any)
				switch cm["type"] {
				case "text":
					if t, _ := cm["text"].(string); t != "" {
						texts = append(texts, t)
					}
				case "thinking":
					if t, _ := cm["thinking"].(string); t != "" {
						thinks = append(thinks, t)
					}
				case "tool_use":
					// 工具调用
					name, _ := cm["name"].(string)
					input := jsonOrEmpty(cm["input"])
					ev.Tool = &SessionToolUse{Name: name, Input: input, Status: "pending"}
				case "tool_result":
					if ev.Tool != nil {
						if content, _ := cm["content"].([]any); len(content) > 0 {
							if first, ok := content[0].(map[string]any); ok {
								if t, _ := first["text"].(string); t != "" {
									ev.Tool.Output = t
									ev.Tool.Status = "ok"
								}
							}
						}
					}
				}
			}
			ev.Content = strings.Join(texts, "\n")
			ev.Thinking = strings.Join(thinks, "\n")
		}
		if usage, ok := msg["usage"].(map[string]any); ok {
			d := &SessionTokenDelta{}
			if v, ok := usage["input_tokens"].(float64); ok {
				d.Input = int64(v)
			}
			if v, ok := usage["output_tokens"].(float64); ok {
				d.Output = int64(v)
			}
			if v, ok := usage["cache_read_input_tokens"].(float64); ok {
				d.Cache = int64(v)
			}
			ev.Tokens = d
		}
	}

	// 系统错误
	if errObj, ok := raw["error"].(map[string]any); ok {
		if msg, _ := errObj["message"].(string); msg != "" {
			ev.Error = msg
		}
	}
	return ev
}

func jsonOrEmpty(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// —— SessionStream ——

// StreamLatest 用 fsnotify watch 最新一个 session 文件，新行解析后通过 cb 推。
// 启动时先 seek 到末尾，忽略已有内容——只关心"现在起的对话"。
func (c *ClaudeAdapter) StreamLatest(ctx context.Context, cb func(SessionEvent)) error {
	// 找最新 session
	path, err := c.findLatestSession()
	if err != nil {
		return err
	}
	if path == "" {
		return fmt.Errorf("no session file found")
	}

	// 打开 + seek 到末尾
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	info, _ := f.Stat()
	_, _ = f.Seek(info.Size(), 0)
	reader := bufio.NewReader(f)

	// 启动 fsnotify 监听
	w, err := fsnotify.NewWatcher()
	if err != nil {
		f.Close()
		return err
	}
	if err := w.Add(filepath.Dir(path)); err != nil {
		f.Close()
		w.Close()
		return err
	}

	c.streamMu.Lock()
	c.streamCancel = func() {
		_ = w.Close()
		_ = f.Close()
	}
	c.streamMu.Unlock()

	core.Log.Info("session stream started", "tribe", "claude", "file", path)

	// 主循环
	go func() {
		defer func() {
			_ = w.Close()
			_ = f.Close()
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				if ev.Name != path {
					continue
				}
				if ev.Op&(fsnotify.Write|fsnotify.Create) == 0 {
					continue
				}
				c.readNewEvents(reader, cb)
			case <-time.After(30 * time.Second):
				// 心跳检查：确认 session 文件还在被写
				if _, err := os.Stat(path); err != nil {
					return
				}
			}
		}
	}()

	return nil
}

// streamMu 保护 streamCancel。
var _streamMu sync.Mutex

// findLatestSession 找最近修改的 session jsonl。
func (c *ClaudeAdapter) findLatestSession() (string, error) {
	dir := c.projectsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	var newest os.DirEntry
	var newestMod time.Time
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		files, _ := os.ReadDir(filepath.Join(dir, e.Name()))
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".jsonl") {
				continue
			}
			info, _ := f.Info()
			if newest == nil || info.ModTime().After(newestMod) {
				// path 是 dir + name
				newest = f
				newestMod = info.ModTime()
			}
		}
	}
	if newest == nil {
		return "", nil
	}
	// 从 dir entries 反推完整路径
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		candidate := filepath.Join(dir, e.Name(), newest.Name())
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", nil
}

func (c *ClaudeAdapter) readNewEvents(r *bufio.Reader, cb func(SessionEvent)) {
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			// 重新打开 + seek 续读
			return
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var raw map[string]any
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		ev := normalizeSessionEvent(raw)
		if ev.Type != "" {
			cb(ev)
		}
	}
}

// StopStream 终止 streamLatest 启动的监听。
func (c *ClaudeAdapter) StopStream() {
	c.streamMu.Lock()
	defer c.streamMu.Unlock()
	if c.streamCancel != nil {
		c.streamCancel()
		c.streamCancel = nil
	}
}

// streamMu / streamCancel 字段。
// 注意：ClaudeAdapter 是并发读多写少；这两个字段只在 stream 操作里用，加锁保护。
// 实际定义在 claude.go 之外会更整洁，但 ClaudeAdapter 字段只能加在它的定义里。

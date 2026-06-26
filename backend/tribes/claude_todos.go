// Claude Code 的 TodoList / Subagent / Compact 解析实现。
//
// 三者都从 session jsonl 中提取特定模式的事件：
//   - TodoWrite 工具调用 → 当前 todo 列表
//   - Task 工具调用 + agent_id → Subagent 父子树
//   - compact_boundary 系统事件 → 压缩标记
//
// 由于数据全在 jsonl 文件里，复用 SessionReader 的 findSessionPath + 流式扫描。
package tribes

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// —— TodoLister 实现 ——

// ListTodos 返回该 session 最新一次 TodoWrite 调用的快照。
// 多次 TodoWrite 会覆盖（取最后一个）。
func (c *ClaudeAdapter) ListTodos(sessionID string) ([]Todo, error) {
	path := c.findSessionPath(sessionID)
	if path == "" {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return parseLatestTodos(path)
}

// parseLatestTodos 扫整个 jsonl（按时间顺序），返回最后一次 TodoWrite 的 todos。
func parseLatestTodos(path string) ([]Todo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var latest []Todo
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		var ev map[string]any
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue
		}
		todos := extractTodosFromEvent(ev)
		if todos != nil {
			latest = todos
		}
	}
	return latest, sc.Err()
}

// extractTodosFromEvent 从单条事件中提取 TodoWrite 的 todos。
// 返回 nil 表示该事件不是 TodoWrite。
func extractTodosFromEvent(ev map[string]any) []Todo {
	if t, _ := ev["type"].(string); t != "assistant" {
		return nil
	}
	msg, _ := ev["message"].(map[string]any)
	if msg == nil {
		return nil
	}
	content, _ := msg["content"].([]any)
	for _, c := range content {
		cc, ok := c.(map[string]any)
		if !ok {
			continue
		}
		ct, _ := cc["type"].(string)
		if ct != "tool_use" {
			continue
		}
		name, _ := cc["name"].(string)
		if name != "TodoWrite" {
			continue
		}
		input, _ := cc["input"].(map[string]any)
		if input == nil {
			continue
		}
		todosRaw, _ := input["todos"].([]any)
		return convertTodos(todosRaw)
	}
	return nil
}

func convertTodos(raw []any) []Todo {
	out := make([]Todo, 0, len(raw))
	for _, r := range raw {
		m, ok := r.(map[string]any)
		if !ok {
			continue
		}
		t := Todo{
			Content:    stringField(m, "content"),
			Status:     TodoStatus(stringField(m, "status")),
			ActiveForm: stringField(m, "activeForm"),
		}
		out = append(out, t)
	}
	return out
}

// —— SubagentTreeLister 实现 ——

// GetSubagentTree 从 session jsonl 构建子 agent 树。
//
// 数据来源：
//   1. assistant 事件里的 Task 工具调用（每个 = 一个父节点）
//   2. 该 Task 的 tool_result 里返回 agent_id（子 session 标识）
//   3. 子 session 的 jsonl 文件给出 agent 类型 + 完成时间 + 消息统计
//
// 注意：父子关系靠 agent_id ↔ 子 session id 匹配。
func (c *ClaudeAdapter) GetSubagentTree(sessionID string) (*AgentNode, error) {
	path := c.findSessionPath(sessionID)
	if path == "" {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	root := &AgentNode{
		ID:          sessionID,
		Type:        "main",
		Description: "main session",
		Children:    []*AgentNode{},
	}
	if err := buildSubagentTree(c.home, path, root); err != nil {
		return root, err
	}
	return root, nil
}

// buildSubagentTree 递归构建子 agent 树。
// parent 是当前节点；扫描 parent 的 jsonl 找 Task 调用，挂到 parent.Children。
func buildSubagentTree(home, parentPath string, parent *AgentNode) error {
	tasks, err := extractTaskCalls(parentPath)
	if err != nil {
		return err
	}
	// 按时间排序，保证稳定的展示顺序
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].startedAt < tasks[j].startedAt
	})

	for _, t := range tasks {
		node := &AgentNode{
			ID:          t.agentID,
			Type:        t.agentType,
			Description: t.description,
			Prompt:      t.prompt,
			Status:      "running",
			StartedAt:   t.startedAt,
			Children:    []*AgentNode{},
		}
		parent.Children = append(parent.Children, node)

		// 找子 session 的 jsonl
		if t.agentID == "" {
			continue
		}
		childPath := findSessionPathByID(home, t.agentID)
		if childPath == "" {
			node.Status = "unknown"
			continue
		}
		// 拿子 session 的统计
		fillAgentStats(childPath, node)

		// 递归：子 agent 可能也派生子 agent
		_ = buildSubagentTree(home, childPath, node)
	}
	return nil
}

// taskCall 扫描到的 Task 工具调用 + 结果。
type taskCall struct {
	agentID     string
	agentType   string
	description string
	prompt      string
	startedAt   int64
}

// extractTaskCalls 从 jsonl 提取所有 Task 工具调用 + 关联的 agent_id。
//
// 流程：
//   1. 收集 assistant 消息里的 Task tool_use（带 input.subagent_type / description / prompt + 时间戳）
//   2. 收集 user 消息里的 tool_result（按 tool_use_id 关联回上一步）
//   3. 把 (tool_use_id → agent_id) 注入 taskCall
func extractTaskCalls(path string) ([]taskCall, error) {
	type useRecord struct {
		toolUseID   string
		agentType   string
		description string
		prompt      string
		startedAt   int64
	}
	uses := map[string]*useRecord{}
	results := map[string]string{} // tool_use_id → agent_id
	order := []string{}           // preserve order of uses

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		var ev map[string]any
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue
		}
		ts := parseTimestamp(ev["timestamp"])

		switch t, _ := ev["type"].(string); t {
		case "assistant":
			msg, _ := ev["message"].(map[string]any)
			if msg == nil {
				continue
			}
			content, _ := msg["content"].([]any)
			for _, c := range content {
				cc, ok := c.(map[string]any)
				if !ok {
					continue
				}
				if ct, _ := cc["type"].(string); ct != "tool_use" {
					continue
				}
				name, _ := cc["name"].(string)
				if name != "Task" {
					continue
				}
				id, _ := cc["id"].(string)
				input, _ := cc["input"].(map[string]any)
				if input == nil || id == "" {
					continue
				}
				uses[id] = &useRecord{
					toolUseID:   id,
					agentType:   stringField(input, "subagent_type"),
					description: stringField(input, "description"),
					prompt:      stringField(input, "prompt"),
					startedAt:   ts,
				}
				order = append(order, id)
			}

		case "user":
			msg, _ := ev["message"].(map[string]any)
			if msg == nil {
				continue
			}
			content, _ := msg["content"].([]any)
			for _, c := range content {
				cc, ok := c.(map[string]any)
				if !ok {
					continue
				}
				if ct, _ := cc["type"].(string); ct != "tool_result" {
					continue
				}
				toolUseID, _ := cc["tool_use_id"].(string)
				if toolUseID == "" {
					continue
				}
				// content 可能是 string（agent_id）或 array of blocks
				if s, ok := cc["content"].(string); ok {
					results[toolUseID] = s
				} else if arr, ok := cc["content"].([]any); ok {
					for _, b := range arr {
						bb, ok := b.(map[string]any)
						if !ok {
							continue
						}
						if bt, _ := bb["type"].(string); bt == "text" {
							if s, ok := bb["text"].(string); ok {
								results[toolUseID] = s
								break
							}
						}
					}
				}
			}
		}
	}

	out := make([]taskCall, 0, len(order))
	for _, id := range order {
		u := uses[id]
		out = append(out, taskCall{
			agentID:     results[id],
			agentType:   u.agentType,
			description: u.description,
			prompt:      u.prompt,
			startedAt:   u.startedAt,
		})
	}
	return out, nil
}

// fillAgentStats 填子 agent 的统计（消息数、tool 调用数、起止时间、状态）。
func fillAgentStats(path string, node *AgentNode) {
	f, err := os.Open(path)
	if err != nil {
		node.Status = "unknown"
		return
	}
	defer f.Close()

	var minTs, maxTs int64
	var lastIsAssistant bool
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		var ev map[string]any
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue
		}
		ts := parseTimestamp(ev["timestamp"])
		if ts > 0 {
			if minTs == 0 || ts < minTs {
				minTs = ts
			}
			if ts > maxTs {
				maxTs = ts
			}
		}
		switch t, _ := ev["type"].(string); t {
		case "assistant":
			node.MessageCount++
			lastIsAssistant = true
			// 数 tool_use
			msg, _ := ev["message"].(map[string]any)
			if msg == nil {
				continue
			}
			if content, ok := msg["content"].([]any); ok {
				for _, c := range content {
					if cc, ok := c.(map[string]any); ok {
						if ct, _ := cc["type"].(string); ct == "tool_use" {
							node.ToolUseCount++
						}
					}
				}
			}
		case "user":
			lastIsAssistant = false
		}
	}

	if minTs > 0 {
		node.StartedAt = minTs
	}
	node.EndedAt = maxTs

	// 启发式：最后一条消息是 assistant → 还在跑；最后是 user → 已结束
	if maxTs > 0 {
		// 简化判断：30 分钟前结束的视为完成
		ageMs := time.Now().UnixMilli() - maxTs
		if ageMs > 30*60*1000 {
			node.Status = "completed"
		} else if lastIsAssistant {
			node.Status = "running"
		} else {
			node.Status = "completed"
		}
	}
}

// findSessionPathByID 在所有 projects 子目录里找 session_id 匹配的文件。
func findSessionPathByID(home, id string) string {
	dir := filepath.Join(home, ".claude", "projects")
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

// —— CompactLister 实现 ——

// ListCompactEvents 列出 session 中的 compact_boundary 事件。
func (c *ClaudeAdapter) ListCompactEvents(sessionID string) ([]CompactEvent, error) {
	path := c.findSessionPath(sessionID)
	if path == "" {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return parseCompactEvents(path, sessionID)
}

func parseCompactEvents(path, sessionID string) ([]CompactEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []CompactEvent
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		var ev map[string]any
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue
		}
		if t, _ := ev["type"].(string); t != "system" {
			continue
		}
		if sub, _ := ev["subtype"].(string); sub != "compact_boundary" {
			continue
		}
		ts := parseTimestamp(ev["timestamp"])
		meta, _ := ev["compact_metadata"].(map[string]any)
		var trigger string
		var preTokens int64
		if meta != nil {
			trigger, _ = meta["trigger"].(string)
			switch v := meta["pre_tokens"].(type) {
			case float64:
				preTokens = int64(v)
			case int64:
				preTokens = v
			}
		}
		out = append(out, CompactEvent{
			SessionID: sessionID,
			Trigger:   trigger,
			PreTokens: preTokens,
			Timestamp: ts,
			At:        ts,
		})
	}
	return out, sc.Err()
}

// —— helpers ——

func stringField(m map[string]any, key string) string {
	if s, ok := m[key].(string); ok {
		return s
	}
	return ""
}

// parseTimestamp 把 jsonl 的 timestamp 字段转 unix ms。
// Claude Code 一般用 RFC3339Nano 字符串；少数用 unix seconds float。
func parseTimestamp(v any) int64 {
	if v == nil {
		return 0
	}
	if s, ok := v.(string); ok && s != "" {
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return t.UnixMilli()
		}
		// 兜底：截取前 10 字符当日期（不精确，但有就行）
		if len(s) >= 10 {
			s = s[:10]
			if t, err := time.Parse("2006-01-02", s); err == nil {
				return t.UnixMilli()
			}
		}
	}
	if f, ok := v.(float64); ok {
		// > 1e12 → 已经是毫秒；否则当秒
		if f > 1e12 {
			return int64(f)
		}
		return int64(f * 1000)
	}
	return 0
}

// 用于 io.LimitReader 占位 import（避免移除 import 后 lint 警告）
var _ = io.LimitReader
var _ = strings.TrimSpace
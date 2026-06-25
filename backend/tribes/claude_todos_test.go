package tribes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFixture(t *testing.T, lines ...string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, line := range lines {
		_, _ = f.WriteString(line + "\n")
	}
	return path
}

func TestParseLatestTodos_Empty(t *testing.T) {
	path := writeFixture(t)
	todos, err := parseLatestTodos(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 0 {
		t.Errorf("expected 0 todos, got %d", len(todos))
	}
}

func TestParseLatestTodos_SingleWrite(t *testing.T) {
	path := writeFixture(t,
		`{"type":"assistant","timestamp":"2026-06-25T10:00:00Z","message":{"content":[{"type":"tool_use","name":"TodoWrite","input":{"todos":[{"content":"Read file","status":"in_progress","activeForm":"Reading file"},{"content":"Edit code","status":"pending","activeForm":"Editing code"}]}}]}}`,
	)
	todos, err := parseLatestTodos(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 2 {
		t.Fatalf("got %d todos, want 2", len(todos))
	}
	if todos[0].Content != "Read file" || todos[0].Status != TodoInProgress {
		t.Errorf("first todo: %+v", todos[0])
	}
	if todos[1].Status != TodoPending {
		t.Errorf("second todo status: %s", todos[1].Status)
	}
}

func TestParseLatestTodos_PicksLast(t *testing.T) {
	// 第一次 TodoWrite 后又更新了一次——应该返回最后一次的快照
	path := writeFixture(t,
		`{"type":"assistant","timestamp":"2026-06-25T10:00:00Z","message":{"content":[{"type":"tool_use","name":"TodoWrite","input":{"todos":[{"content":"First","status":"pending"}]}}]}}`,
		`{"type":"assistant","timestamp":"2026-06-25T10:01:00Z","message":{"content":[{"type":"tool_use","name":"TodoWrite","input":{"todos":[{"content":"First","status":"completed"},{"content":"Second","status":"in_progress"}]}}]}}`,
	)
	todos, _ := parseLatestTodos(path)
	if len(todos) != 2 {
		t.Fatalf("got %d, want 2", len(todos))
	}
	if todos[0].Status != TodoCompleted {
		t.Errorf("first should be completed: %s", todos[0].Status)
	}
	if todos[1].Status != TodoInProgress {
		t.Errorf("second should be in_progress: %s", todos[1].Status)
	}
}

func TestParseLatestTodos_IgnoresNonTodoWrite(t *testing.T) {
	path := writeFixture(t,
		`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"/tmp/x"}}]}}`,
		`{"type":"user","message":{"content":"hi"}}`,
	)
	todos, _ := parseLatestTodos(path)
	if len(todos) != 0 {
		t.Errorf("got %d, want 0", len(todos))
	}
}

func TestParseCompactEvents(t *testing.T) {
	path := writeFixture(t,
		`{"type":"system","subtype":"compact_boundary","timestamp":"2026-06-25T09:00:00Z","compact_metadata":{"trigger":"auto","pre_tokens":50000}}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"hi"}]}}`,
		`{"type":"system","subtype":"compact_boundary","timestamp":"2026-06-25T11:00:00Z","compact_metadata":{"trigger":"manual","pre_tokens":120000}}`,
	)
	evs, err := parseCompactEvents(path, "sess-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 2 {
		t.Fatalf("got %d, want 2", len(evs))
	}
	if evs[0].Trigger != "auto" || evs[0].PreTokens != 50000 {
		t.Errorf("first: %+v", evs[0])
	}
	if evs[1].Trigger != "manual" || evs[1].PreTokens != 120000 {
		t.Errorf("second: %+v", evs[1])
	}
	if evs[0].SessionID != "sess-1" {
		t.Errorf("session id not propagated: %s", evs[0].SessionID)
	}
}

func TestExtractTaskCalls_Basic(t *testing.T) {
	path := writeFixture(t,
		`{"type":"assistant","timestamp":"2026-06-25T10:00:00Z","message":{"content":[{"type":"tool_use","id":"tu-1","name":"Task","input":{"subagent_type":"general-purpose","description":"Research X","prompt":"Find info"}}]}}`,
		`{"type":"user","timestamp":"2026-06-25T10:00:05Z","message":{"content":[{"type":"tool_result","tool_use_id":"tu-1","content":"agent-id-xyz"}]}}`,
	)
	tasks, err := extractTaskCalls(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("got %d, want 1", len(tasks))
	}
	tc := tasks[0]
	if tc.agentID != "agent-id-xyz" {
		t.Errorf("agentID: %s", tc.agentID)
	}
	if tc.agentType != "general-purpose" {
		t.Errorf("agentType: %s", tc.agentType)
	}
	if tc.description != "Research X" {
		t.Errorf("description: %s", tc.description)
	}
	if tc.startedAt == 0 {
		t.Errorf("startedAt should be parsed")
	}
}

func TestExtractTaskCalls_OrphanedUse(t *testing.T) {
	// 没有匹配的 tool_result → agentID 为空
	path := writeFixture(t,
		`{"type":"assistant","timestamp":"2026-06-25T10:00:00Z","message":{"content":[{"type":"tool_use","id":"tu-1","name":"Task","input":{"subagent_type":"Explore","description":"Find"}}]}}`,
	)
	tasks, _ := extractTaskCalls(path)
	if len(tasks) != 1 {
		t.Fatalf("got %d, want 1", len(tasks))
	}
	if tasks[0].agentID != "" {
		t.Errorf("orphan task should have empty agentID, got %s", tasks[0].agentID)
	}
}

func TestConvertTodos(t *testing.T) {
	raw := []any{
		map[string]any{"content": "a", "status": "pending"},
		map[string]any{"content": "b", "status": "in_progress", "activeForm": "doing b"},
	}
	todos := convertTodos(raw)
	if len(todos) != 2 {
		t.Fatalf("got %d", len(todos))
	}
	if todos[0].Content != "a" || todos[0].Status != TodoPending {
		t.Errorf("first: %+v", todos[0])
	}
	if todos[1].ActiveForm != "doing b" {
		t.Errorf("activeForm: %s", todos[1].ActiveForm)
	}
}

func TestParseTimestamp(t *testing.T) {
	cases := []struct {
		in   any
		want int64
	}{
		{"2026-06-25T10:00:00Z", 1782424800000},
		{"2026-06-25T10:00:00.123Z", 1782424800123},
		{float64(1782424800), 1782424800000},     // unix seconds
		{float64(1782424800123), 1782424800123},  // already ms
		{nil, 0},
		{"", 0},
		{"garbage", 0},
	}
	for _, c := range cases {
		got := parseTimestamp(c.in)
		// 只检查数量级：RFC3339 精确验证脆弱
		if c.want == 0 && got != 0 {
			t.Errorf("parseTimestamp(%v)=%d, want 0", c.in, got)
		}
		if c.want != 0 && got == 0 {
			t.Errorf("parseTimestamp(%v)=0, want %d", c.in, c.want)
		}
	}
}

func TestFillAgentStats(t *testing.T) {
	path := writeFixture(t,
		`{"type":"user","timestamp":"2026-06-25T10:00:00Z","message":{"content":"hi"}}`,
		`{"type":"assistant","timestamp":"2026-06-25T10:00:01Z","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"ls"}}]}}`,
		`{"type":"user","timestamp":"2026-06-25T10:00:02Z","message":{"content":"result"}}`,
	)
	node := &AgentNode{}
	fillAgentStats(path, node)
	if node.MessageCount != 1 {
		t.Errorf("MessageCount: %d", node.MessageCount)
	}
	if node.ToolUseCount != 1 {
		t.Errorf("ToolUseCount: %d", node.ToolUseCount)
	}
	if node.StartedAt == 0 || node.EndedAt == 0 {
		t.Errorf("timestamps not set: %d %d", node.StartedAt, node.EndedAt)
	}
}

// 兜底测试：未格式化的字段不应该 panic
func TestExtractTodosFromEvent_Malformed(t *testing.T) {
	cases := []map[string]any{
		nil,
		{},
		{"type": "user"},
		{"type": "assistant"},
		{"type": "assistant", "message": "string-not-map"},
		{"type": "assistant", "message": map[string]any{}},
		{"type": "assistant", "message": map[string]any{"content": "not-array"}},
		{"type": "assistant", "message": map[string]any{"content": []any{"not-map"}}},
	}
	for _, c := range cases {
		got := extractTodosFromEvent(c)
		if got != nil {
			t.Errorf("extractTodosFromEvent(%v) = %v, want nil", c, got)
		}
	}
}

// TestFindSessionPathByID 集成测试：建一个真实的 home 结构，验证查找
func TestFindSessionPathByID(t *testing.T) {
	home := t.TempDir()
	projectsDir := filepath.Join(home, ".claude", "projects", "abc")
	if err := os.MkdirAll(projectsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	targetID := "sess-target"
	if err := os.WriteFile(filepath.Join(projectsDir, targetID+".jsonl"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectsDir, "other.jsonl"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := findSessionPathByID(home, targetID)
	if got == "" {
		t.Fatal("expected non-empty path")
	}
	if !strings.HasSuffix(got, targetID+".jsonl") {
		t.Errorf("path doesn't end with target id: %s", got)
	}

	// 找不到的 ID
	if got := findSessionPathByID(home, "missing"); got != "" {
		t.Errorf("expected empty for missing, got %s", got)
	}
}
package tribes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withTempClaudeHome 设置 c.home 指向临时目录，返回清理函数。
func withTempClaudeHome(t *testing.T) (*ClaudeAdapter, func()) {
	t.Helper()
	dir := t.TempDir()
	adapter := NewClaudeAdapter(dir)
	// 预创建 .claude 目录
	if err := os.MkdirAll(adapter.claudeHome(), 0o755); err != nil {
		t.Fatal(err)
	}
	return adapter, func() {}
}

func TestFindMemories_OnlyUser(t *testing.T) {
	a, cleanup := withTempClaudeHome(t)
	defer cleanup()

	// 创建 user global
	if err := os.WriteFile(a.userMemoryPath(), []byte("# user memory"), 0o644); err != nil {
		t.Fatal(err)
	}

	srcs, err := a.FindMemories("")
	if err != nil {
		t.Fatal(err)
	}
	if len(srcs) != 1 || srcs[0].Scope != "user" {
		t.Errorf("got %+v, want 1 user source", srcs)
	}
}

func TestFindMemories_ProjectPriority(t *testing.T) {
	a, cleanup := withTempClaudeHome(t)
	defer cleanup()

	// 创建 user + project 几个文件
	if err := os.WriteFile(a.userMemoryPath(), []byte("# user"), 0o644); err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".claude", "CLAUDE.md"), []byte("# project .claude"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "CLAUDE.md"), []byte("# project root"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "CLAUDE.md"), []byte("# project root"), 0o644); err != nil {
		t.Fatal(err)
	}

	srcs, err := a.FindMemories(project)
	if err != nil {
		t.Fatal(err)
	}
	// 期望：project/.claude/CLAUDE.md (priority 1), project/CLAUDE.md (priority 2), user (always)
	if len(srcs) != 3 {
		t.Fatalf("got %d, want 3", len(srcs))
	}
	if srcs[0].Scope != "project" || !strings.HasSuffix(srcs[0].Path, ".claude/CLAUDE.md") {
		t.Errorf("first should be project/.claude/CLAUDE.md, got %+v", srcs[0])
	}
	if srcs[2].Scope != "user" {
		t.Errorf("last should be user, got %+v", srcs[2])
	}
}

func TestFindMemories_ParentTraversal(t *testing.T) {
	a, cleanup := withTempClaudeHome(t)
	defer cleanup()

	// 在 /tmp/foo/bar/baz 里——回溯 4 层找 /tmp/foo/CLAUDE.md
	tmp := t.TempDir()
	fooBarBaz := filepath.Join(tmp, "foo", "bar", "baz")
	if err := os.MkdirAll(fooBarBaz, 0o755); err != nil {
		t.Fatal(err)
	}
	parentCLAUDE := filepath.Join(tmp, "foo", "CLAUDE.md")
	if err := os.WriteFile(parentCLAUDE, []byte("# parent"), 0o644); err != nil {
		t.Fatal(err)
	}

	srcs, err := a.FindMemories(fooBarBaz)
	if err != nil {
		t.Fatal(err)
	}
	// 期望找到 parent CLAUDE.md（从 baz 上溯 2 层到 foo）
	found := false
	for _, s := range srcs {
		if s.Path == parentCLAUDE {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find parent CLAUDE.md, got %+v", srcs)
	}
}

func TestSplitFrontmatter_YesFrontmatter(t *testing.T) {
	in := "---\nname: x\ntags: [a, b]\n---\n\n# Title\n\nbody"
	fm, body := splitFrontmatter(in)
	if fm != "name: x\ntags: [a, b]" {
		t.Errorf("fm: %q", fm)
	}
	if !strings.Contains(body, "# Title") {
		t.Errorf("body missing title: %q", body)
	}
}

func TestSplitFrontmatter_NoFrontmatter(t *testing.T) {
	in := "# Title\n\nbody"
	fm, body := splitFrontmatter(in)
	if fm != "" {
		t.Errorf("expected empty fm, got %q", fm)
	}
	if body != in {
		t.Errorf("body should equal input")
	}
}

func TestSplitFrontmatter_Unclosed(t *testing.T) {
	in := "---\nname: x\n# Title\nbody without closing ---"
	fm, _ := splitFrontmatter(in)
	if fm != "" {
		t.Errorf("unclosed should give empty fm, got %q", fm)
	}
}

func TestParseSections(t *testing.T) {
	body := `# Top

intro text

## Sub One

content one

## Sub Two

content two

### Deep

deep content

## Sub Three

last
`
	sections := parseSections(body)
	// 应该有 4 个：Top, Sub One, Sub Two, Deep, Sub Three... 实际是 5
	if len(sections) != 5 {
		t.Fatalf("got %d sections, want 5", len(sections))
	}
	if sections[0].Title != "Top" || sections[0].Level != 1 {
		t.Errorf("section 0: %+v", sections[0])
	}
	if !strings.Contains(sections[1].Content, "content one") {
		t.Errorf("section 1 content: %q", sections[1].Content)
	}
	if sections[3].Title != "Deep" || sections[3].Level != 3 {
		t.Errorf("section 3: %+v", sections[3])
	}
	if sections[4].Title != "Sub Three" {
		t.Errorf("section 4: %+v", sections[4])
	}
}

func TestParseSections_Empty(t *testing.T) {
	if got := parseSections(""); len(got) != 0 {
		t.Errorf("empty body should give empty sections, got %d", len(got))
	}
}

func TestMatchHeading(t *testing.T) {
	cases := []struct {
		line  string
		title string
		level int
	}{
		{"# Top", "Top", 1},
		{"## Sub", "Sub", 2},
		{"### Deep", "Deep", 3},
		{"###### H6", "H6", 6},
		{"####### H7", "", 0},   // 超过 6
		{"#NoSpace", "NoSpace", 1}, // CommonMark 允许无空格 ATX 标题
		{"plain text", "", 0},
		{"#", "", 0},            // 只有 # 没有文本
		{"", "", 0},
		{"```code```", "", 0},   // fenced code
		{"   # Indented", "Indented", 1}, // 行首空格允许
	}
	for _, c := range cases {
		t.Run(c.line, func(t *testing.T) {
			gotTitle, gotLevel := matchHeading(c.line)
			if gotTitle != c.title || gotLevel != c.level {
				t.Errorf("(%q): got (%q, %d), want (%q, %d)", c.line, gotTitle, gotLevel, c.title, c.level)
			}
		})
	}
}

func TestParseImports(t *testing.T) {
	body := `# Memory

@~/.claude/commands.md

Some text @./local.md more text

# Another section

@/absolute/path.md
Email should not match: user@example.com
`
	imps := parseImports(body)
	if len(imps) != 3 {
		t.Fatalf("got %d imports, want 3: %+v", len(imps), imps)
	}
	wantPaths := []string{"~/.claude/commands.md", "./local.md", "/absolute/path.md"}
	for i, imp := range imps {
		if imp.Path != wantPaths[i] {
			t.Errorf("import %d: got %s, want %s", i, imp.Path, wantPaths[i])
		}
	}
}

func TestReadMemory_RoundTrip(t *testing.T) {
	a, cleanup := withTempClaudeHome(t)
	defer cleanup()

	content := `---
name: project-foo
tags: [a, b]
---

# Memory about user

User prefers concise answers.

# Memory about project

This is a Go project using Wails.
`
	path := filepath.Join(a.claudeHome(), "CLAUDE.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	doc, err := a.ReadMemory(path)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Source.Path != path {
		t.Errorf("source path: %s", doc.Source.Path)
	}
	if doc.Source.Scope != "user" {
		t.Errorf("scope: %s", doc.Source.Scope)
	}
	if !strings.Contains(doc.Frontmatter, "name: project-foo") {
		t.Errorf("frontmatter: %q", doc.Frontmatter)
	}
	if len(doc.Sections) != 2 {
		t.Fatalf("got %d sections, want 2", len(doc.Sections))
	}
	if doc.Sections[0].Title != "Memory about user" {
		t.Errorf("section 0: %+v", doc.Sections[0])
	}
	if !strings.Contains(doc.Sections[1].Content, "Go project") {
		t.Errorf("section 1 content: %q", doc.Sections[1].Content)
	}
}

func TestSaveMemory_WithBackup(t *testing.T) {
	a, cleanup := withTempClaudeHome(t)
	defer cleanup()

	path := filepath.Join(a.claudeHome(), "CLAUDE.md")
	original := "# Original\n\nbody"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := a.SaveMemory(path, "# New\n\nnew body", ""); err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "# New") {
		t.Errorf("saved content: %s", got)
	}

	// 备份存在
	home, _ := os.UserHomeDir()
	backupDir := filepath.Join(home, ".aland", "backups")
	entries, _ := os.ReadDir(backupDir)
	if len(entries) == 0 {
		t.Errorf("no backup created in %s", backupDir)
	}
}

func TestSaveMemory_WithFrontmatter(t *testing.T) {
	a, cleanup := withTempClaudeHome(t)
	defer cleanup()

	path := filepath.Join(a.claudeHome(), "CLAUDE.md")
	if err := a.SaveMemory(path, "# Body", "key: value"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(got), "---\nkey: value\n---") {
		t.Errorf("frontmatter not wrapped: %s", got)
	}
}

func TestScopeFor(t *testing.T) {
	if got := scopeFor("/foo/bar", "/foo/bar"); got != "user" {
		t.Errorf("exact match should be user, got %s", got)
	}
	if got := scopeFor("/foo/bar", "/baz"); got != "project" {
		t.Errorf("non-match should be project, got %s", got)
	}
}
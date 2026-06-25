// Claude Code CLAUDE.md 解析与编辑。
//
// 探测位置（按优先级）：
//   1. <cwd>/.claude/CLAUDE.md   (项目本地 .claude 目录)
//   2. <cwd>/CLAUDE.md           (项目根)
//   3. <cwd>/../CLAUDE.md        (向上回溯，monorepo 友好)
//   4. ~/.claude/CLAUDE.md       (user global)
//
// 解析：
//   - YAML frontmatter (--- ... ---)
//   - # / ## / ### 章节
//   - @path 导入引用
//
// 写回：保留 frontmatter + 章节顺序，按 section 替换 content。
package tribes

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/narcilee7/aland/backend/core"
)

// maxParents 控制向上回溯的层数。
const maxParents = 4

// userMemoryPath 用户全局 CLAUDE.md。
func (c *ClaudeAdapter) userMemoryPath() string {
	return filepath.Join(c.claudeHome(), "CLAUDE.md")
}

// FindMemories 探测所有可用的 CLAUDE.md。
//
// cwd 为空时只用 user global；有 cwd 时按上面的优先级列表返回。
func (c *ClaudeAdapter) FindMemories(cwd string) ([]MemorySource, error) {
	var out []MemorySource
	seen := map[string]bool{}

	add := func(path, scope string) {
		if seen[path] {
			return
		}
		if _, err := os.Stat(path); err != nil {
			return
		}
		seen[path] = true
		out = append(out, MemorySource{Path: path, Scope: scope})
	}

	if cwd != "" {
		// 当前目录 + .claude
		add(filepath.Join(cwd, ".claude", "CLAUDE.md"), "project")
		add(filepath.Join(cwd, "CLAUDE.md"), "project")

		// 向上回溯
		dir := cwd
		for i := 0; i < maxParents; i++ {
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
			add(filepath.Join(dir, ".claude", "CLAUDE.md"), "project")
			add(filepath.Join(dir, "CLAUDE.md"), "project")
		}
	}

	// user global 永远返回（如果存在）
	add(c.userMemoryPath(), "user")

	return out, nil
}

// ReadMemory 解析单个 CLAUDE.md 文件。
func (c *ClaudeAdapter) ReadMemory(path string) (*MemoryDoc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read memory: %w", err)
	}
	stat, _ := os.Stat(path)

	doc := &MemoryDoc{
		Source: MemorySource{
			Path:  path,
			Scope: scopeFor(path, c.userMemoryPath()),
		},
		Body:       string(data),
		ModifiedAt: stat.ModTime().UnixMilli(),
		SizeBytes:  stat.Size(),
	}

	// 1. 解析 frontmatter（如果存在）
	fm, body := splitFrontmatter(string(data))
	doc.Frontmatter = fm
	doc.Body = body

	// 2. 解析章节
	doc.Sections = parseSections(body)

	// 3. 解析 @imports
	doc.Imports = parseImports(body)

	return doc, nil
}

// SaveMemory 写回 CLAUDE.md。body 是完整正文（含标题），frontmatter 可为空。
// 自动备份原文件到 ~/.aland/backups/CLAUDE-{unix}.md。
func (c *ClaudeAdapter) SaveMemory(path, body, frontmatter string) error {
	var content string
	if frontmatter != "" {
		content = "---\n" + frontmatter + "\n---\n\n" + body
	} else {
		content = body
	}

	// 备份
	if _, err := os.Stat(path); err == nil {
		if err := backupMemory(path); err != nil {
			core.Log.Warn("memory backup failed", "err", err)
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	core.Log.Info("memory saved", "path", path, "size", len(content))
	return os.WriteFile(path, []byte(content), 0o644)
}

func backupMemory(src string) error {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".aland", "backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	stamp := time.Now().Unix()
	dst := filepath.Join(dir, fmt.Sprintf("CLAUDE-%d.md", stamp))
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

// scopeFor 根据路径判断 scope（user / project）。
func scopeFor(path, userGlobal string) string {
	if path == userGlobal {
		return "user"
	}
	return "project"
}

// splitFrontmatter 拆分 YAML frontmatter 和正文。
// 如果首行是 `---`，读到下一个 `---` 为止作为 frontmatter；
// 否则 frontmatter 为空。
func splitFrontmatter(content string) (fm, body string) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	if !scanner.Scan() {
		return "", content
	}
	first := strings.TrimSpace(scanner.Text())
	if first != "---" {
		return "", content
	}

	var fmLines []string
	closed := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			closed = true
			break
		}
		fmLines = append(fmLines, line)
	}
	if !closed {
		return "", content
	}
	fm = strings.Join(fmLines, "\n")
	body = strings.TrimPrefix(content, scanner.Text()+"\n")
	// 跳过 frontmatter 结尾的 --- + 一个空行
	body = strings.TrimPrefix(body, "---\n")
	body = strings.TrimPrefix(body, "\n")
	return fm, body
}

// parseSections 按 # / ## / ### 切分章节。
// 章节 content = 标题行 + 下方正文，直到下一个同级或更高级标题。
func parseSections(body string) []MemorySection {
	var out []MemorySection
	scanner := bufio.NewScanner(strings.NewReader(body))
	var current *MemorySection
	var contentLines []string
	order := 0

	flush := func() {
		if current == nil {
			return
		}
		current.Content = strings.TrimSpace(strings.Join(contentLines, "\n"))
		out = append(out, *current)
		current = nil
		contentLines = nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		if title, level := matchHeading(line); level > 0 {
			flush()
			current = &MemorySection{
				Title: title,
				Level: level,
				Order: order,
			}
			order++
			continue
		}
		if current != nil {
			contentLines = append(contentLines, line)
		}
	}
	flush()
	return out
}

// matchHeading 检查一行是不是标题。
// 返回 (title, level)；level=0 表示不是标题。
func matchHeading(line string) (string, int) {
	trimmed := strings.TrimLeft(line, " ")
	if strings.HasPrefix(trimmed, "#") {
		// 数前导 #
		level := 0
		for _, c := range trimmed {
			if c == '#' {
				level++
			} else {
				break
			}
		}
		if level > 6 || level == 0 {
			return "", 0
		}
		rest := strings.TrimSpace(trimmed[level:])
		// 标题后面必须有空格 + 文本，否则视为普通文本
		if rest == "" {
			return "", 0
		}
		// 跳过 fenced code
		if strings.HasPrefix(rest, "```") {
			return "", 0
		}
		return rest, level
	}
	return "", 0
}

// parseImports 找所有 @path 形式的导入引用。
func parseImports(body string) []MemoryImport {
	var out []MemoryImport
	scanner := bufio.NewScanner(strings.NewReader(body))
	line := 0
	for scanner.Scan() {
		line++
		text := scanner.Text()
		// 简单的 @path 模式：以 @ 开头、空格分隔（不含行内 @user/email 之类）
		for _, tok := range strings.Fields(text) {
			if !strings.HasPrefix(tok, "@") {
				continue
			}
			path := strings.TrimPrefix(tok, "@")
			if path == "" || strings.ContainsAny(path, " \t") {
				continue
			}
			// 过滤常见误识别：@ 后是 email 字符
			if strings.Contains(path, "@") {
				continue
			}
			out = append(out, MemoryImport{Path: path, Line: line})
		}
	}
	return out
}
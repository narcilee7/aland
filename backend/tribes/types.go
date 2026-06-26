// Package tribes 包含部落（AI CLI）领域的所有类型与适配器。
//
// 设计原则：
//   - 小接口，按"消费方需要什么就暴露什么"切分（Detector/Launcher/Reader/Identity）
//   - Tribe 是纯数据结构；具体能力通过接口字段组合到 Land 中
//   - 适配器只需实现自己关心的方法，结构性满足即可（Go 的隐式接口）
package tribes

// ProcessInfo 进程信息快照。
type ProcessInfo struct {
	PID       int     `json:"pid"`
	Name      string  `json:"name"`
	CmdLine   string  `json:"cmdLine"`
	CWD       string  `json:"cwd"`
	CPU       float64 `json:"cpu"`
	Memory    float64 `json:"memory"`
	StartTime int64   `json:"startTime"` // unix seconds
}

// TokenUsage 一个时间窗内的 Token 消耗。
type TokenUsage struct {
	InputTokens  int64   `json:"inputTokens"`
	OutputTokens int64   `json:"outputTokens"`
	Model        string  `json:"model"`
	CostUSD      float64 `json:"costUsd"`
}

// Meta 部落的前端元信息。
// 注册时由 Identity 适配器一次性填充，运行期不变。
type Meta struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Eco         string `json:"eco"`
	ThemeColor  string `json:"themeColor"`
	AccentColor string `json:"accentColor"`
}

// SessionShard 一个历史会话（"记忆碎片"）。
type SessionShard struct {
	ID           string `json:"id"`
	Tribe        string `json:"tribe"`
	Timestamp    int64  `json:"timestamp"`   // unix ms
	MessageCount int    `json:"messageCount"`
	TokenCount   int64  `json:"tokenCount"`
	Model        string `json:"model"`
	CWD          string `json:"cwd"`
	Project      string `json:"project"` // 解码后的项目路径
	FilePath     string `json:"filePath"`
	SizeBytes    int64  `json:"sizeBytes"`
	Summary      string `json:"summary"` // ai-title 或首条 user 消息
}

// ConfigItem 一个配置项。Sensitive=true 表示 API Key 等，UI 要走锁芯。
type ConfigItem struct {
	Key       string `json:"key"`
	Value     any    `json:"value"`
	DefaultV  any    `json:"defaultValue,omitempty"`
	Type      string `json:"type"` // string | number | boolean | json
	Layer     string `json:"layer"`
	Sensitive bool   `json:"sensitive,omitempty"`
}

// ConfigField 一个配置项的 schema 元信息。
// 让前端能根据类型渲染不同的 UI（string → text input，secret → masked，enum → select）。
type ConfigField struct {
	Key         string   `json:"key"`
	Label       string   `json:"label,omitempty"`
	Description string   `json:"description,omitempty"`
	Type        string   `json:"type"` // string | number | boolean | enum | secret | json
	EnumValues  []string `json:"enumValues,omitempty"`
	Sensitive   bool     `json:"sensitive"`
	Editable    bool     `json:"editable"`
	Default     any      `json:"default,omitempty"`
}

// ConfigSchema 整个配置的 schema（每条 key 一个 Field）。
// 让"任意 adapter 的配置"都能被同一套 UI 渲染——可扩展架构的关键。
type ConfigSchema struct {
	Fields []ConfigField `json:"fields"`
}

// ConfigDNA 三层配置：surface（运行时） / middle（API & 权限） / deep（其他）。
// Schema 是可扩展渲染入口；Items 是当前值。
type ConfigDNA struct {
	Source  string       `json:"source"`
	Schema  ConfigSchema `json:"schema"`
	Surface []ConfigItem `json:"surface"`
	Middle  []ConfigItem `json:"middle"`
	Deep    []ConfigItem `json:"deep"`
}

// —— 19 项能力对应的新类型 ——

// MCPServer Model Context Protocol server 配置。
// 来自 ~/.claude/.mcp.json 或 settings.json 里的 mcpServers。
type MCPServer struct {
	Name      string            `json:"name"`
	Command   string            `json:"command"`
	Args      []string          `json:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Transport string            `json:"transport,omitempty"` // stdio / sse / http
	Source    string            `json:"source"`             // ".mcp.json" / "settings.json"
	Enabled   bool              `json:"enabled"`
}

// Skill 自定义 skill（slash command）定义。
// 来自 ~/.claude/skills/<name>/SKILL.md。
type Skill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
	Content     string `json:"content"`
}

// PlanFile Plan 模式的产物。
// 来自 ~/.claude/plans/*.md。
type PlanFile struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Size       int64  `json:"size"`
	ModifiedAt int64  `json:"modifiedAt"`
	Summary    string `json:"summary"`
}

// FileEdit Claude 修改过的文件历史（来自 ~/.claude/file-history/）。
// 配合 backup 路径，可以做 restore。
type FileEdit struct {
	Path         string `json:"path"`
	BackupPath   string `json:"backupPath"`
	Timestamp    int64  `json:"timestamp"`
	OriginalHash string `json:"originalHash,omitempty"`
	Version      int    `json:"version"`
}

// Plugin 启用的 plugin（来自 settings.json 的 enabledPlugins）。
type Plugin struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Source  string `json:"source"` // settings.json
}

// DailyActivity 来自 ~/.claude/stats-cache.json。
// 每天的消息数 / session 数 / tool 调用数。
type DailyActivity struct {
	Date         string `json:"date"`
	MessageCount int    `json:"messageCount"`
	SessionCount int    `json:"sessionCount"`
	ToolCallCount int   `json:"toolCallCount"`
}

// ModelTokenUsage 单 model 单日 token 使用。
type ModelTokenUsage struct {
	Model        string `json:"model"`
	Date         string `json:"date"`
	InputTokens  int64  `json:"inputTokens"`
	OutputTokens int64  `json:"outputTokens"`
	CacheRead    int64  `json:"cacheRead,omitempty"`
	CacheWrite   int64  `json:"cacheWrite,omitempty"`
}

// SlashCommand 来自 ~/.claude/history.jsonl 的最近 slash 命令。
type SlashCommand struct {
	Command   string `json:"command"`
	Args     string `json:"args,omitempty"`
	Timestamp int64  `json:"timestamp"`
	CWD      string `json:"cwd,omitempty"`
}

// SessionEvent 单条 session 事件（实时 tail / 全文读取都用这个）。
// 简化版：保留 type + 关键字段，不完整镜像 jsonl 全部 schema。
type SessionEvent struct {
	Type      string             `json:"type"`
	Subtype   string             `json:"subtype,omitempty"`
	Timestamp int64              `json:"timestamp"`
	Role      string             `json:"role,omitempty"` // user / assistant
	Content   string             `json:"content,omitempty"`
	Thinking  string             `json:"thinking,omitempty"`
	Model     string             `json:"model,omitempty"`
	Tokens    *SessionTokenDelta `json:"tokens,omitempty"`
	Tool      *SessionToolUse    `json:"tool,omitempty"`
	Error     string             `json:"error,omitempty"`
}

// SessionTokenDelta 一次 assistant 调用的 token 增量。
type SessionTokenDelta struct {
	Input  int64 `json:"input"`
	Output int64 `json:"output"`
	Cache  int64 `json:"cache,omitempty"`
}

// SessionToolUse 一次 tool 调用。
type SessionToolUse struct {
	Name   string `json:"name"`
	Input  string `json:"input,omitempty"`
	Output string `json:"output,omitempty"`
	Status string `json:"status,omitempty"` // ok / error
}

// TodoStatus todo 的状态。
type TodoStatus string

const (
	TodoPending    TodoStatus = "pending"
	TodoInProgress TodoStatus = "in_progress"
	TodoCompleted  TodoStatus = "completed"
)

// Todo 来自 Claude Code 的 TodoWrite 工具调用。
// 每个 session 的最新 TodoWrite 事件决定当前可见的 todo 列表。
type Todo struct {
	Content    string     `json:"content"`    // 任务描述
	Status     TodoStatus `json:"status"`     // pending / in_progress / completed
	ActiveForm string     `json:"activeForm"` // 进行中时的动词形式
}

// AgentNode Subagent 树的一个节点。
//
// Claude Code 用 Task 工具派生子 agent，每个子 agent 有自己的 session。
// 树结构：父节点 = 发起 Task 的 assistant 消息；子节点 = 该 Task 派生的 agent。
type AgentNode struct {
	ID          string       `json:"id"`          // 子 agent 的 session ID（或 Task 的 tool_use_id）
	Type        string       `json:"type"`        // agent_type: general-purpose / Explore / Plan 等
	Description string       `json:"description"` // 用户提供的描述
	Prompt      string       `json:"prompt,omitempty"`
	Status      string       `json:"status"`      // running / completed / error / unknown
	StartedAt   int64        `json:"startedAt"`   // unix ms
	EndedAt     int64        `json:"endedAt"`     // unix ms；0 表示还在跑
	MessageCount int         `json:"messageCount"`
	ToolUseCount int         `json:"toolUseCount"`
	Children    []*AgentNode `json:"children"`
}

// CompactEvent 上下文压缩事件。
type CompactEvent struct {
	SessionID  string `json:"sessionId"`
	Trigger    string `json:"trigger"`    // "manual" | "auto"
	PreTokens  int64  `json:"preTokens"`  // 压缩前的 token 数
	Timestamp  int64  `json:"timestamp"`  // unix ms
	At         int64  `json:"at"`         // 同上冗余字段，方便前端
}

// MemorySource CLAUDE.md 的来源。
type MemorySource struct {
	Path  string `json:"path"`  // 绝对路径
	Scope string `json:"scope"` // "user" | "project" | "local"
}

// MemorySection CLAUDE.md 的一个章节（# 标题 + 下方正文）。
type MemorySection struct {
	Title   string `json:"title"`   // 去掉 # 的标题
	Level   int    `json:"level"`   // 1-6
	Content string `json:"content"` // 该章节下的 markdown（不含标题本身）
	Order   int    `json:"order"`   // 章节顺序（0-indexed）
}

// MemoryImport @file 导入引用。
type MemoryImport struct {
	Path string `json:"path"` // @ 后面的路径
	Line int    `json:"line"` // 出现在文件的哪一行
}

// MemoryDoc 解析后的 CLAUDE.md 完整结构。
type MemoryDoc struct {
	Source      MemorySource   `json:"source"`
	Frontmatter string         `json:"frontmatter"` // YAML（不含 --- 包裹符）
	Sections    []MemorySection `json:"sections"`
	Imports     []MemoryImport  `json:"imports"`
	Body        string         `json:"body"`        // 完整正文（含标题）
	ModifiedAt  int64          `json:"modifiedAt"`  // unix ms
	SizeBytes   int64          `json:"sizeBytes"`
}

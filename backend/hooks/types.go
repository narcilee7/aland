// Package hooks 把 Claude Code 的 Hooks 系统接到 Aland。
//
// 工作流：
//   1. Aland 启动时启动 HTTP server (127.0.0.1:38765-38774)
//   2. 同时把自己的命令注册到 ~/.claude/settings.json 的 hooks 字段
//      (PreToolUse/PostToolUse/Notification/Stop/SubagentStop/UserPromptSubmit)
//   3. Claude Code 在事件发生时调 `aland hook <event>` 子命令
//   4. 该子命令读 stdin JSON → POST 到 Aland server
//   5. Server 解析后 emit claude:hook Wails 事件，前端订阅做实时可视化
//
// 设计取舍：
//   - 走 HTTP 而不是 unix socket：跨平台、debug 容易（curl 即可测试）
//   - 端口固定 38765+回退：避免每次都改 hook command 里的端口
//   - 写 port 到 ~/.aland/hook.port：方便其他工具探测
//   - settings.json 改动走 backup：跟 config write 同样谨慎
package hooks

// Event Claude Code hook 的事件名常量。
//
// 这些是 Claude Code 在调用 hook 时传给子命令的第一个参数。
// 完整的事件名（如 PreToolUse 的子类型）作为第二个参数。
const (
	EventPreToolUse     = "PreToolUse"
	EventPostToolUse    = "PostToolUse"
	EventNotification   = "Notification"
	EventStop           = "Stop"
	EventSubagentStop   = "SubagentStop"
	EventUserPrompt     = "UserPromptSubmit"
	EventPreCompact     = "PreCompact"
	EventSessionStart   = "SessionStart"
	EventSessionEnd     = "SessionEnd"
)

// AllEvents 所有 Aland 注册的事件。Install 时按这个列表展开。
var AllEvents = []string{
	EventPreToolUse,
	EventPostToolUse,
	EventNotification,
	EventStop,
	EventSubagentStop,
	EventUserPrompt,
	EventPreCompact,
}

// HookPayload Claude Code 通过 stdin 传给 hook 的 JSON。
//
// 字段是"广覆盖"——所有事件共有的字段 + 各事件特有的字段。
// Specific 字段按事件名动态填充。
type HookPayload struct {
	// 共有
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	CWD            string `json:"cwd"`
	HookEventName  string `json:"hook_event_name"` // e.g. "PreToolUse"

	// PreToolUse / PostToolUse 特有
	ToolName     string         `json:"tool_name,omitempty"`
	ToolInput    map[string]any `json:"tool_input,omitempty"`
	ToolResponse map[string]any `json:"tool_response,omitempty"` // 仅 PostToolUse

	// Notification 特有
	NotificationType string `json:"notification_type,omitempty"` // e.g. "permission_prompt"
	NotificationMsg  string `json:"message,omitempty"`

	// Stop / SubagentStop 特有
	StopReason    string `json:"stop_reason,omitempty"`
	AgentID       string `json:"agent_id,omitempty"`
	AgentType     string `json:"agent_type,omitempty"`
	AgentTranscript string `json:"agent_transcript_path,omitempty"`

	// UserPromptSubmit 特有
	UserPrompt string `json:"user_prompt,omitempty"`

	// PreCompact 特有
	CompactTrigger string `json:"trigger,omitempty"` // "manual" | "auto"
	CustomInstructions string `json:"custom_instructions,omitempty"`
}

// HookResponse hook 可选的 stdout 输出。
//
// 仅 PreToolUse 支持决策（approve / deny / 修改 input）。其他事件一般不输出。
// 字段对应 Claude Code 的 exit code + JSON stdout 协议。
type HookResponse struct {
	// Continue 决定是否继续执行（false = 阻断）
	Continue bool `json:"continue,omitempty"`
	// StopReason 当 Continue=false 时显示给 Claude 的原因
	StopReason string `json:"stopReason,omitempty"`
	// Decision PreToolUse 的决策：approve / deny / ask
	Decision string `json:"decision,omitempty"`
	// Reason 决策理由
	Reason string `json:"reason,omitempty"`
	// ModifiedInput 修改后的工具输入（仅 PreToolUse 支持）
	ModifiedInput map[string]any `json:"modifiedInput,omitempty"`
}
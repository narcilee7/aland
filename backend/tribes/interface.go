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

// TribeAdapter 每个 CLI 的统一抽象。
// 接入新 CLI = 实现这个接口 + Register。
type TribeAdapter interface {
	// 身份
	ID() string         // "claude" | "cursor" | ...
	Name() string       // 显示名
	EcoType() string    // 生态类型，决定地形风格

	// 进程
	DetectProcess() (*ProcessInfo, error)
	Launch(cwd string, args []string) error
	Kill(pid int) error

	// 配置
	ConfigPaths() []string
	ParseConfig() (map[string]interface{}, error)

	// 前端元数据
	ThemeColor() string // 主色 #d4a853
	AccentColor() string // 荧光 #ffdf80
}

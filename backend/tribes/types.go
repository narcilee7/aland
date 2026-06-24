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

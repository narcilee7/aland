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

// ConfigDNA 三层配置：surface（运行时） / middle（API & 权限） / deep（其他）。
// 来源标识 Source 用于 UI 展示"配置自哪"。
type ConfigDNA struct {
	Source  string       `json:"source"`
	Surface []ConfigItem `json:"surface"`
	Middle  []ConfigItem `json:"middle"`
	Deep    []ConfigItem `json:"deep"`
}

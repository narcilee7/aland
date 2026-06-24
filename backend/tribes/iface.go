package tribes

// 这里是"部落能力"的小接口集合。
// 每个接口只暴露一种能力，由消费方按需持有。
// 适配器不需显式声明实现——Go 的结构性接口让任何匹配签名的类型自动满足。

// Identity 一个部落的身份信息。注册时由 Registry 调用一次。
type Identity interface {
	ID() string
	Name() string
	EcoType() string
	ThemeColor() string
	AccentColor() string
}

// Detector 进程检测能力。infra.ProcManager 持有。
type Detector interface {
	DetectProcess() (*ProcessInfo, error)
}

// Launcher 进程启动/终止能力。Wails 绑定层持有。
type Launcher interface {
	Launch(cwd string, args []string) error
	Kill(pid int) error
}

// Reader 配置读取能力。Wails 绑定层持有。
type Reader interface {
	ConfigPaths() []string
	ParseConfig() (map[string]interface{}, error)
}

// TokenStatReader Token 消耗解析能力。Forge 持有，刷新熔炉液面。
type TokenStatReader interface {
	ParseTokenUsage() (*TokenUsage, error)
}

// SessionLister 历史会话列表能力。TribeView 持有，渲染记忆碎片。
type SessionLister interface {
	ParseSessions() ([]SessionShard, error)
}

// ConfigParser 三层结构化配置解析。TribeView 持有，渲染 Config DNA。
type ConfigParser interface {
	ParseConfigDNA() (ConfigDNA, error)
}

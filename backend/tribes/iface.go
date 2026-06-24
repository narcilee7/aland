package tribes

import "context"

// 这里是"部落能力"的小接口集合。
// 每个接口只暴露一种能力，由消费方按需持有。
// 适配器不需显式声明实现——Go 的结构性接口让任何匹配签名的类型自动满足。

// Identity 一个部落的身份信息。注册时由 Registry 调用一次。
// Capabilities() 是"可扩展架构"的契约入口——adapter 必报家门。
type Identity interface {
	ID() string
	Name() string
	EcoType() string
	ThemeColor() string
	AccentColor() string
	Capabilities() Capabilities
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

// ConfigWriter 配置写回。修改前必须自动备份到 ~/.aland/backups/。
// 跟 Reader 配对存在——能 Config 不一定能 ConfigEdit。
type ConfigWriter interface {
	WriteConfig(dna ConfigDNA) error
}

// MCPServerLister Model Context Protocol 服务器列表。
type MCPServerLister interface {
	ListMCPServers() ([]MCPServer, error)
}

// SkillsLister 自定义 skill 列表。
type SkillsLister interface {
	ListSkills() ([]Skill, error)
}

// PlanLister Plan 文件列表。
type PlanLister interface {
	ListPlans() ([]PlanFile, error)
}

// FileHistoryLister 文件编辑历史。
type FileHistoryLister interface {
	ListFileHistory() ([]FileEdit, error)
	// RestoreFile 恢复到指定 FileEdit 的 backup 版本。
	RestoreFile(edit FileEdit) error
}

// StatsProvider 每日活跃 + 按 model token 使用 + slash command 历史。
type StatsProvider interface {
	DailyActivity() ([]DailyActivity, error)
	ModelTokenUsage() ([]ModelTokenUsage, error)
	RecentSlashCommands(n int) ([]SlashCommand, error)
	// ListPlugins 从 settings.enabledPlugins 读。
	ListPlugins() ([]Plugin, error)
}

// SessionStreamer 实时 tail 当前 session。
// 启动后通过回调推送新事件；Stop 终止。
type SessionStreamer interface {
	StreamLatest(ctx context.Context, cb func(SessionEvent)) error
	StopStream()
}

// SessionReader 完整读取一个 session（用于回看）。
type SessionReader interface {
	ReadSession(id string) ([]SessionEvent, error)
}

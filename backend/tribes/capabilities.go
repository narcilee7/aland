package tribes

// Capabilities 一个 adapter 自报家门的能力清单。
//
// 这是"可扩展架构"的核心契约。
//
// 原则：
//   - 能力是 *true/false 声明*，不是接口实现检查。
//     这样 IDE 类的工具（Cursor / Trae）能"诚实地说"自己不支持 Sessions。
//   - 能力按"用户能感知到的事"分，不是按技术实现分。
//     比如 SessionTail 是用户能看到的，StreamJSON 是用户看不到的——后者不暴露。
//   - 能力是 *static* 的，运行时不变。启动时查询，UI 一次构建。
type Capabilities struct {
	// 进程
	Process bool // DetectProcess：能告诉你 CLI 跑没跑、PID/资源占用

	// 启停
	Launch bool // Launch / Kill：能从 Aland 启动/终止

	// 配置
	Config     bool // ParseConfig / ParseConfigDNA：能读
	ConfigEdit bool // WriteConfig：能写回 + 备份

	// 会话
	Sessions    bool // ParseSessions：能列出历史会话
	SessionTail bool // StreamSession：能实时观察当前会话（killer feature）

	// Token / 成本
	Tokens     bool // ParseTokenUsage：能算历史/今日 token
	TokensLive bool // StreamTokenUsage：能看实时单次调用 cost

	// 扩展 feature——CLI 特定的能力
	// 比如 Claude 的 "mcp_servers"、Cursor 的 "extensions"
	// 用 Feature.ID 路由到具体 UI
	Features []Feature
}

// Feature 一个 CLI 特定的扩展能力。
// 跟标准能力分开，因为它们没有跨 CLI 的统一语义。
type Feature struct {
	ID          string `json:"id"`          // 唯一标识：路由 + 持久化
	Label       string `json:"label"`       // UI 显示名
	Description string `json:"description"` // 悬停解释
	HasData     bool   `json:"hasData"`     // 是否有现成数据可看（false = 占位）
}

// FeatureID 预定义的扩展 feature 标识。
// 这些是跨 CLI 共享语义的 feature，UI 可以针对它们做特殊处理。
const (
	// Claude 特有
	FeatureMCPServers = "mcp_servers"
	FeatureSkills     = "skills"
	FeaturePlugins    = "plugins"
	FeaturePlans      = "plans"
	FeatureFileHistory = "file_history"

	// IDE 通用
	FeatureExtensions = "extensions"
	FeatureWorkspaces = "workspaces"
)

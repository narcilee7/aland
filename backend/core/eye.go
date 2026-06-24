package core

type EyeStateMode string

const (
	DoramantMode EyeStateMode = "dormant"
	ActiveMode EyeStateMode = "active"
	StormMode EyeStateMode = "storm"
	AlertMode EyeStateMode = "alert"
)

// EyeState 灵动岛（系统级悬浮胶囊）的状态。
// M0 只保留数据结构，真正的悬浮窗在 v1 阶段接入。
type EyeState struct {
	Mode    EyeStateMode   `json:"mode"`    // dormant | active | storm | alert
	Running []string `json:"running"` // 当前运行的部落 ID
}

// NewEyeState 创建一个沉睡的灵动岛。
func NewEyeState() *EyeState {
	return &EyeState{
		Mode:    DoramantMode,
		Running: []string{},
	}
}

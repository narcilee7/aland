package core

// Forge Token 熔炉——成本与 Token 的计算中心。
// M0 只做占位，v1 之后接入真实日志解析。
type Forge struct {
	DailyBudget int64            `json:"dailyBudget"`
	TodaySpent  int64            `json:"todaySpent"`
	ByTribe     map[string]int64 `json:"byTribe"`
	ByModel     map[string]int64 `json:"byModel"`
}

// NewForge 创建一个空的熔炉。
func NewForge() *Forge {
	return &Forge{
		ByTribe: make(map[string]int64),
		ByModel: make(map[string]int64),
	}
}

// Snapshot 返回当前熔炉状态。
func (f *Forge) Snapshot() Forge {
	return Forge{
		DailyBudget: f.DailyBudget,
		TodaySpent:  f.TodaySpent,
		ByTribe:     f.ByTribe,
		ByModel:     f.ByModel,
	}
}

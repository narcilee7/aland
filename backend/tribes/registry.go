package tribes

import "github.com/narcilee7/aland/backend/core"

// RegisterAll 把所有已实现的部落登记到大陆。
// 新增适配器时，只需在这里加一行。
func RegisterAll(land *core.Land) {
	adapters := []TribeAdapter{
		NewClaudeAdapter(),
		// NewCursorAdapter(),  // v1
		// NewTraeAdapter(),    // v1
		// NewKimiAdapter(),    // v1
	}
	for _, a := range adapters {
		land.Register(&core.Tribe{
			ID:      a.ID(),
			Name:    a.Name(),
			Eco:     a.EcoType(),
			Status:  core.StatusIdle,
			Adapter: a,
		})
	}
}

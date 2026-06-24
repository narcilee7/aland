package tribes

import "github.com/narcilee7/aland/backend/core"

// AsAdapter 把 core.Tribe.Adapter 安全地还原成 TribeAdapter。
// 失败时返回 nil，调用方需要做 nil 检查。
// 提供这个 helper 是因为 core.Tribe.Adapter 字段是 any，
// 避免在多个调用点重复写类型断言。
func AsAdapter(t *core.Tribe) TribeAdapter {
	if t == nil {
		return nil
	}
	a, _ := t.Adapter.(TribeAdapter)
	return a
}

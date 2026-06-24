// Package events 定义 Aland 的类型化事件。
//
// 为什么不用 string literal：
//   - 编译期检查：typo 立刻被编译器抓到
//   - IDE 重命名支持
//   - payload 结构可被消费方静态推断
//
// 命名规则：`{domain}:{verb}`，如 `tribe:born`、`spotlight:toggle`。
// payload 类型紧邻事件名定义，方便阅读。
package events

import "github.com/narcilee7/aland/backend/tribes"

// 事件名常量。前端通过 mirror 常量匹配。
const (
	// TribeVital 每 1s 推送一次大陆全量快照。
	TribeVital = "tribe:vital"
	// TribeBorn 部落进程首次出现时推送。
	TribeBorn = "tribe:born"
	// TribeDeath 部落进程消失时推送。
	TribeDeath = "tribe:death"
	// FSChange 配置文件变更时推送（500ms 防抖）。
	FSChange = "fs:change"
	// SpotlightToggle 全局快捷键 Cmd+Shift+A 触发时推送。
	SpotlightToggle = "spotlight:toggle"
)

// TribeLifecycleEvent tribe:born / tribe:death 的 payload。
type TribeLifecycleEvent struct {
	ID   string `json:"id"`
	PID  int    `json:"pid"`
	Name string `json:"name"`
}

// FSChangeEvent fs:change 的 payload。
type FSChangeEvent struct {
	Path string `json:"path"`
	Op   string `json:"op"`
}

// SpotlightToggleEvent spotlight:toggle 的 payload。
// Action 是 "open" / "close" / "toggle"，由快捷键层决定。
type SpotlightToggleEvent struct {
	Action string `json:"action"`
}

// TribeSnapshotMap 是 tribe:vital 的 payload 类型。
// 用类型别名（而非 struct）让 Emit 接受任意 map，序列化时与 Wails 一致。
type TribeSnapshotMap = map[string]tribes.Tribe

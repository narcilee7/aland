package eye

import (
	"testing"

	"github.com/narcilee7/aland/backend/core"
)

// TestIconPerModeUnique 验证 4 个 mode 产出不同的图标字节。
func TestIconPerModeUnique(t *testing.T) {
	seen := map[string]core.EyeStateMode{}
	for _, m := range []core.EyeStateMode{
		core.DormantMode,
		core.ActiveMode,
		core.StormMode,
		core.AlertMode,
	} {
		icon := iconForMode(m)
		if len(icon) == 0 {
			t.Errorf("mode %s: empty icon", m)
		}
		// PNG magic number
		if len(icon) < 8 || icon[0] != 0x89 || icon[1] != 'P' {
			t.Errorf("mode %s: not a valid PNG (got %x...)", m, icon[:4])
		}
		if other, dup := seen[string(icon)]; dup {
			t.Errorf("mode %s and %s produced identical icon", m, other)
		}
		seen[string(icon)] = m
	}
}

// TestIconPerFlashUnique flash 类型也应该各异。
func TestIconPerFlashUnique(t *testing.T) {
	seen := map[string]core.FlashType{}
	flashTypes := []core.FlashType{
		core.FlashComplete,
		core.FlashBorn,
		core.FlashDeath,
		core.FlashError,
		core.FlashCostAlert,
		core.FlashConflict,
	}
	for _, f := range flashTypes {
		icon := iconForFlash(f)
		if len(icon) == 0 {
			t.Errorf("flash %s: empty icon", f)
		}
		if other, dup := seen[string(icon)]; dup {
			t.Errorf("flash %s and %s produced identical icon", f, other)
		}
		seen[string(icon)] = f
	}
}

func TestTooltipForMode(t *testing.T) {
	cases := map[core.EyeStateMode]string{
		core.DormantMode: "Aland · 沉睡守望",
		core.ActiveMode:  "Aland · 部落活跃",
		core.StormMode:   "Aland · 风暴（高负载）",
		core.AlertMode:   "Aland · 告警",
	}
	for m, want := range cases {
		if got := tooltipForMode(m); got != want {
			t.Errorf("mode %s: tooltip=%q, want %q", m, got, want)
		}
	}
	// 未知 mode 返回默认
	if got := tooltipForMode("unknown"); got != "Aland" {
		t.Errorf("unknown mode: got %q, want Aland", got)
	}
}

func TestJoinComma(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{nil, ""},
		{[]string{}, ""},
		{[]string{"a"}, "a"},
		{[]string{"a", "b"}, "a, b"},
		{[]string{"claude", "cursor", "trae"}, "claude, cursor, trae"},
	}
	for _, c := range cases {
		if got := joinComma(c.in); got != c.want {
			t.Errorf("joinComma(%v)=%q, want %q", c.in, got, c.want)
		}
	}
}

func TestNewTrayDefaultsToDormant(t *testing.T) {
	tr := New()
	if tr == nil {
		t.Fatal("New() returned nil")
	}
	if tr.current != core.DormantMode {
		t.Errorf("initial mode=%s, want dormant", tr.current)
	}
	if tr.ready {
		t.Errorf("tray should not be ready before Run()")
	}
}
package core

import (
	"testing"
)

// TestRecompute 覆盖 4 种模式的转移。
func TestRecompute(t *testing.T) {
	cases := []struct {
		name     string
		input    []TribeVitalInput
		wantMode EyeStateMode
		wantRun  []string
		wantChg  bool // 期望 Recompute 返回 changed
	}{
		{
			name:     "no tribes → dormant",
			input:    nil,
			wantMode: DormantMode,
			wantChg:  false, // 默认就是 dormant，无变化
		},
		{
			name: "one running → active",
			input: []TribeVitalInput{
				{ID: "claude", Status: "running", PID: 100, CPU: 30},
			},
			wantMode: ActiveMode,
			wantRun:  []string{"claude"},
			wantChg:  true,
		},
		{
			name: "error status → alert",
			input: []TribeVitalInput{
				{ID: "claude", Status: "error", PID: 100, CPU: 30},
			},
			wantMode: AlertMode,
			wantChg:  true,
		},
		{
			name: "high total cpu → storm",
			input: []TribeVitalInput{
				{ID: "claude", Status: "running", PID: 1, CPU: 90},
				{ID: "cursor", Status: "running", PID: 2, CPU: 80},
				{ID: "trae", Status: "running", PID: 3, CPU: 70},
			},
			wantMode: StormMode,
			wantChg:  true,
		},
		{
			name: "only idle (no PID) → dormant",
			input: []TribeVitalInput{
				{ID: "claude", Status: "idle", PID: 0},
			},
			wantMode: DormantMode,
			wantChg:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := NewEyeState()
			// 第一次 Recompute：模式从默认 Dormant 可能切换
			changed := e.Recompute(tc.input)
			got := e.Snapshot()
			if got.Mode != tc.wantMode {
				t.Errorf("mode: got %s, want %s", got.Mode, tc.wantMode)
			}
			if tc.wantRun != nil && !sliceEq(got.Running, tc.wantRun) {
				t.Errorf("running: got %v, want %v", got.Running, tc.wantRun)
			}
			if changed != tc.wantChg {
				t.Errorf("first Recompute changed=%v, want %v", changed, tc.wantChg)
			}
			// 第二次 Recompute：相同输入 → 不变化
			changed2 := e.Recompute(tc.input)
			if changed2 {
				t.Errorf("second Recompute with same input should not change")
			}
		})
	}
}

func TestPushFlashAndConsume(t *testing.T) {
	e := NewEyeState()

	f1 := e.PushFlash(FlashComplete, "claude", "session abc done")
	f2 := e.PushFlash(FlashError, "cursor", "build failed")
	f3 := e.PushFlash(FlashCostAlert, "", "budget 80%")

	if got := len(e.Snapshot().Flashing); got != 3 {
		t.Fatalf("flashing len: got %d, want 3", got)
	}
	if f1.ID == f2.ID || f2.ID == f3.ID || f1.ID == f3.ID {
		t.Errorf("flash IDs should be unique: %s %s %s", f1.ID, f2.ID, f3.ID)
	}

	if !e.ConsumeFlash(f2.ID) {
		t.Errorf("consume f2 should succeed")
	}
	left := e.Snapshot().Flashing
	if len(left) != 2 {
		t.Errorf("after consume: len=%d, want 2", len(left))
	}
	for _, f := range left {
		if f.ID == f2.ID {
			t.Errorf("f2 still in queue")
		}
	}

	// consume 不存在的 ID
	if e.ConsumeFlash("nonexistent") {
		t.Errorf("consume missing should return false")
	}
}

func TestPushFlashOverflow(t *testing.T) {
	e := NewEyeState()
	// 推 maxFlashing + 5 条，最旧的应该被丢弃
	for i := 0; i < maxFlashing+5; i++ {
		e.PushFlash(FlashComplete, "", "")
	}
	got := len(e.Snapshot().Flashing)
	if got != maxFlashing {
		t.Errorf("overflow: len=%d, want %d", got, maxFlashing)
	}
}

func TestSnapshotDeepCopy(t *testing.T) {
	e := NewEyeState()
	e.PushFlash(FlashComplete, "claude", "x")

	s1 := e.Snapshot()
	// 修改 s1 的 Flashing 切片不应该影响 e
	s1.Flashing[0].Content = "MUTATED"
	s2 := e.Snapshot()
	if s2.Flashing[0].Content == "MUTATED" {
		t.Errorf("snapshot not deep-copied: mutation leaked back")
	}
	// 修改 s1.Running 也不应影响 e
	s1.Running = append(s1.Running, "evil")
	s3 := e.Snapshot()
	for _, id := range s3.Running {
		if id == "evil" {
			t.Errorf("Running slice not deep-copied")
		}
	}
}

func TestClearFlashes(t *testing.T) {
	e := NewEyeState()
	e.PushFlash(FlashComplete, "", "a")
	e.PushFlash(FlashComplete, "", "b")
	e.ClearFlashes()
	if got := len(e.Snapshot().Flashing); got != 0 {
		t.Errorf("after clear: len=%d, want 0", got)
	}
}
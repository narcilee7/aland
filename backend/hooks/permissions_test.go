package hooks

import (
	"testing"
)

func TestPermissionsRoundtrip(t *testing.T) {
	_, restore := withTempHome(t)
	defer restore()

	original := Permissions{
		Allow: []string{"Bash(npm:*)", "Read(./src/**)"},
		Deny:  []string{"Bash(rm:*)"},
		Ask:   []string{"Bash(git push:*)"},
	}
	if err := WritePermissions(original); err != nil {
		t.Fatal(err)
	}
	got, err := ReadPermissions()
	if err != nil {
		t.Fatal(err)
	}
	if !sliceEqStr(got.Allow, original.Allow) {
		t.Errorf("Allow: got %v, want %v", got.Allow, original.Allow)
	}
	if !sliceEqStr(got.Deny, original.Deny) {
		t.Errorf("Deny: got %v, want %v", got.Deny, original.Deny)
	}
	if !sliceEqStr(got.Ask, original.Ask) {
		t.Errorf("Ask: got %v, want %v", got.Ask, original.Ask)
	}
}

func TestPermissions_Empty(t *testing.T) {
	_, restore := withTempHome(t)
	defer restore()

	p, err := ReadPermissions()
	if err != nil {
		t.Fatal(err)
	}
	if !p.Empty() {
		t.Error("expected empty permissions when no file")
	}
}

func TestTogglePermission_Add(t *testing.T) {
	_, restore := withTempHome(t)
	defer restore()

	got, err := TogglePermission("allow", "Bash(npm:*)")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Allow) != 1 || got.Allow[0] != "Bash(npm:*)" {
		t.Errorf("Allow after add: %v", got.Allow)
	}
}

func TestTogglePermission_Remove(t *testing.T) {
	_, restore := withTempHome(t)
	defer restore()

	_, _ = TogglePermission("deny", "Bash(rm:*)")
	got, err := TogglePermission("deny", "Bash(rm:*)")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Deny) != 0 {
		t.Errorf("Deny after toggle twice: %v, want empty", got.Deny)
	}
}

func TestTogglePermission_InvalidCategory(t *testing.T) {
	_, restore := withTempHome(t)
	defer restore()

	_, err := TogglePermission("bogus", "rule")
	if err == nil {
		t.Error("expected error for invalid category")
	}
}

func TestPermissions_PreservesOtherFields(t *testing.T) {
	_, restore := withTempHome(t)
	defer restore()

	writeSettings(t, SettingsPath(), `{
  "model": "claude-opus-4",
  "permissions": {
    "allow": ["Bash(*)"]
  }
}`)

	if err := WritePermissions(Permissions{Deny: []string{"Bash(curl:*)"}}); err != nil {
		t.Fatal(err)
	}

	raw := readSettings(t, SettingsPath())
	if raw["model"] != "claude-opus-4" {
		t.Errorf("model lost: %v", raw["model"])
	}
	perms := raw["permissions"].(map[string]any)
	if perms["allow"] == nil {
		t.Errorf("allow category lost")
	}
}

func sliceEqStr(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
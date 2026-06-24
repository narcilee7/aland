package tribes

// RegisterAll 把所有已实现的部落登记到大陆。
// 新增适配器时，只需在这里加一行。
func RegisterAll(land *Land) error {
	adapters := []any{
		NewClaudeAdapter(""),
		// NewCursorAdapter(""),
		// NewTraeAdapter(""),
		// NewKimiAdapter(""),
	}
	for _, a := range adapters {
		if err := land.Register(a); err != nil {
			return err
		}
	}
	return nil
}

package memoriser

// Exported as a package because this is something
// that people really might want to forcefully change
// themselves
type Memoriser interface {
	Save(string, []any) bool
	Retrieve(string) ([]any, error)
}

type NoOpMemoriser struct {
}

func (no *NoOpMemoriser) Save(string, []any) bool {
	return true
}

func (no *NoOpMemoriser) Retrieve(string) ([]any, error) {
	return []any{}, nil
}

package memoriser

// Exported as a package because this is something
// that people really might want to forcefully change
// themselves
type Memoriser interface {
	Save(string, []string) bool
	Retrieve(string) ([]string, error)
}

type NoOpMemoriser struct {
}

func (no *NoOpMemoriser) Save(string, []string) bool {
	return true
}

func (no *NoOpMemoriser) Retrieve(string) ([]string, error) {
	return make([]string, 0), nil
}

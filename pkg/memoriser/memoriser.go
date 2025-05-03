package memoriser

import "encoding/json"

// Exported as a package because this is something
// that people really might want to forcefully change
// themselves
type Memoriser interface {
	Save(string, json.RawMessage) bool
	Retrieve(string) (json.RawMessage, error)
}

type NoOpMemoriser struct {
}

func (no *NoOpMemoriser) Save(string, json.RawMessage) bool {
	return true
}

func (no *NoOpMemoriser) Retrieve(string) (json.RawMessage, error) {
	return make(json.RawMessage, 0), nil
}

package memoriser

import (
	"encoding/json"
	"errors"
	"sync"
)

type InMemoryMemoriser struct {
	mux     sync.RWMutex
	history map[string]json.RawMessage
}

func (in *InMemoryMemoriser) Save(id string, latest json.RawMessage) bool {
	in.mux.Lock()
	defer in.mux.Unlock()

	in.history[id] = latest

	return true
}

func (in *InMemoryMemoriser) Retrieve(id string) (json.RawMessage, error) {
	in.mux.RLock()
	defer in.mux.RUnlock()

	hist, ok := in.history[id]
	if !ok {
		return nil, errors.New("not found")
	}

	return hist, nil
}

func NewInMemoryMemoriser() *InMemoryMemoriser {
	m := &InMemoryMemoriser{
		history: make(map[string]json.RawMessage, 0),
	}

	return m
}

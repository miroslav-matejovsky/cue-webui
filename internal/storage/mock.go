package storage

import (
	"context"
	"sync"
)

// Mock is an in-memory Store implementation intended for tests and examples.
type Mock struct {
	mu      sync.RWMutex
	values  map[string]string
	loadErr error
	saveErr error
}

// NewMock returns a mock store preloaded with the provided values.
func NewMock(initialValues map[string]string) *Mock {
	return &Mock{values: cloneValues(initialValues)}
}

// LoadMap returns a copy of the currently stored values.
func (m *Mock) LoadMap(_ context.Context) (map[string]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.loadErr != nil {
		return nil, m.loadErr
	}
	return cloneValues(m.values), nil
}

// SaveMap replaces the currently stored values with a copy of the submitted set.
func (m *Mock) SaveMap(_ context.Context, values map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.saveErr != nil {
		return m.saveErr
	}
	m.values = cloneValues(values)
	return nil
}

// Snapshot returns a copy of the current in-memory state.
func (m *Mock) Snapshot() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return cloneValues(m.values)
}

// SetLoadError configures the error returned by Load.
func (m *Mock) SetLoadError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.loadErr = err
}

// SetSaveError configures the error returned by Save.
func (m *Mock) SetSaveError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.saveErr = err
}

func cloneValues(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

package main

import "sync"

// MapSlice -
type MapSlice struct {
	Map map[string][]string
	mu  sync.Mutex
}

// MapSliceNew -
func MapSliceNew() *MapSlice {
	return &MapSlice{
		Map: make(map[string][]string),
	}
}

// Get -
func (m *MapSlice) Get(k string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Map[k]
}

// Len -
func (m *MapSlice) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Map)
}

// Keys -
func (m *MapSlice) Keys() []string {
	m.mu.Lock()
	s := make([]string, 0, len(m.Map))
	defer m.mu.Unlock()
	for k := range m.Map {
		s = append(s, k)
	}
	return s
}

// Set -
func (m *MapSlice) Set(k string, s []string) {
	m.mu.Lock()
	m.Map[k] = s
	m.mu.Unlock()
}

// Clear -
func (m *MapSlice) Clear() {
	m.mu.Lock()
	clear(m.Map)
	m.mu.Unlock()
}

package inventory

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type Store struct {
	mu    sync.RWMutex
	path  string
	state State
}

func OpenStore(path string) (*Store, error) {
	s := &Store{path: path, state: emptyState()}
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read state: %w", err)
	}
	if err == nil {
		if err := json.Unmarshal(b, &s.state); err != nil {
			return nil, fmt.Errorf("decode state: %w", err)
		}
		if s.state.Items == nil {
			s.state.Items = make(map[string]Item)
		}
		if s.state.Alerts == nil {
			s.state.Alerts = []Alert{}
		}
		if s.state.Runs == nil {
			s.state.Runs = []Run{}
		}
	}
	return s, nil
}

// Update commits a complete snapshot atomically. A failed disk write leaves memory unchanged.
func (s *Store) Update(fn func(*State) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.state.clone()
	if err := fn(&next); err != nil {
		return err
	}
	if err := persist(s.path, next); err != nil {
		return err
	}
	s.state = next
	return nil
}

func persist(path string, state State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".inventory-state-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(0600); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func (s *Store) Snapshot() (items []Item, alerts []Alert, runs []Run) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.state.Items {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].SKU < items[j].SKU })
	alerts = append([]Alert{}, s.state.Alerts...)
	runs = append([]Run{}, s.state.Runs...)
	return
}

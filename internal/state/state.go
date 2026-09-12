package state

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

type payload struct {
	DefaultBranches map[string]string `json:"default_branches"`
}

type Store struct {
	path  string
	mu    sync.Mutex
	data  map[string]string
	dirty bool
}

func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return filepath.Join(home, ".local", "mono-vcs", "state.json")
}

func Open(path string) (*Store, error) {
	s := &Store{path: path, data: map[string]string{}}
	if path == "" {
		return s, errors.New("state db path is empty")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return s, nil
		}
		return s, fmt.Errorf("failed to read %s: %v", path, err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return s, nil
	}
	var p payload
	if err := json.Unmarshal(raw, &p); err != nil {
		return s, fmt.Errorf("failed to parse %s: %v", path, err)
	}
	for k, v := range p.DefaultBranches {
		if v != "" {
			s.data[k] = v
		}
	}
	return s, nil
}

func key(repo string) string {
	if abs, err := filepath.Abs(repo); err == nil {
		return abs
	}
	return repo
}

func (s *Store) DefaultBranch(repo string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.data[key(repo)]
	return b, ok && b != ""
}

func (s *Store) SetDefaultBranch(repo, branch string) bool {
	if branch == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key(repo)
	if s.data[k] == branch {
		return false
	}
	s.data[k] = branch
	s.dirty = true
	return true
}

func (s *Store) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.dirty {
		return nil
	}
	if s.path == "" {
		return errors.New("state db path is empty")
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create %s: %v", dir, err)
	}
	data, err := json.MarshalIndent(payload{DefaultBranches: s.data}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(dir, ".state-*.json")
	if err != nil {
		return fmt.Errorf("failed to write %s: %v", s.path, err)
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(name)
		return fmt.Errorf("failed to write %s: %v", s.path, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return fmt.Errorf("failed to write %s: %v", s.path, err)
	}
	if err := os.Chmod(name, 0o644); err != nil {
		os.Remove(name)
		return fmt.Errorf("failed to write %s: %v", s.path, err)
	}
	if err := os.Rename(name, s.path); err != nil {
		os.Remove(name)
		return fmt.Errorf("failed to write %s: %v", s.path, err)
	}
	s.dirty = false
	return nil
}

package aliases

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
)

const KindRemote = "remote"

var Kinds = []string{KindRemote}

type payload struct {
	Remotes map[string]string `json:"remotes,omitempty"`
}

type Store struct {
	path  string
	mu    sync.Mutex
	data  map[string]map[string]string
	dirty bool
}

func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return filepath.Join(home, ".local", "mono-vcs", "aliases.json")
}

func Open(path string) (*Store, error) {
	s := &Store{path: path, data: map[string]map[string]string{}}
	for _, k := range Kinds {
		s.data[k] = map[string]string{}
	}
	if path == "" {
		return s, errors.New("aliases path is empty")
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
	for k, v := range p.Remotes {
		if k != "" && v != "" {
			s.data[KindRemote][k] = v
		}
	}
	return s, nil
}

func ValidName(name string) error {
	if name == "" {
		return errors.New("alias name is empty")
	}
	if strings.ContainsAny(name, "/:, \t") {
		return fmt.Errorf("alias name %q must not contain spaces, commas, %q or %q", name, "/", ":")
	}
	return nil
}

func KnownKind(kind string) bool {
	return slices.Contains(Kinds, kind)
}

func (s *Store) Get(kind, name string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[kind][name]
	return v, ok && v != ""
}

func (s *Store) Set(kind, name, value string) bool {
	if value == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data[kind] == nil {
		s.data[kind] = map[string]string{}
	}
	if s.data[kind][name] == value {
		return false
	}
	s.data[kind][name] = value
	s.dirty = true
	return true
}

func (s *Store) Delete(kind, name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[kind][name]; !ok {
		return false
	}
	delete(s.data[kind], name)
	s.dirty = true
	return true
}

func (s *Store) All(kind string) map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]string, len(s.data[kind]))
	maps.Copy(out, s.data[kind])
	return out
}

func (s *Store) Names(kind string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.data[kind]))
	for k := range s.data[kind] {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (s *Store) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.dirty {
		return nil
	}
	if s.path == "" {
		return errors.New("aliases path is empty")
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create %s: %v", dir, err)
	}
	data, err := json.MarshalIndent(payload{Remotes: s.data[KindRemote]}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(dir, ".aliases-*.json")
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

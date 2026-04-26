package filestore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type rulesFile struct {
	Rules []string `json:"rules"`
}

// FileStore implements core.RuleStorePort backed by a JSON file.
type FileStore struct {
	mu   sync.Mutex
	path string
}

func New(path string) *FileStore { return &FileStore{path: path} }

// Save writes rules atomically: write to .tmp then rename.
func (s *FileStore) Save(rules []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rules == nil {
		rules = []string{}
	}
	data, err := json.MarshalIndent(rulesFile{Rules: rules}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("creating rules dir: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("writing tmp: %w", err)
	}
	return os.Rename(tmp, s.path)
}

// Load reads the rules file. Returns empty slice if file doesn't exist.
func (s *FileStore) Load() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading rules file: %w", err)
	}
	var rf rulesFile
	if err := json.Unmarshal(data, &rf); err != nil {
		return nil, fmt.Errorf("parsing rules file: %w", err)
	}
	if rf.Rules == nil {
		return []string{}, nil
	}
	return rf.Rules, nil
}

package profiles

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Store manages UnitProfile JSON files in a directory.
type Store struct {
	dir string
}

// NewStore creates a Store backed by the given directory path.
func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

// Load reads a UnitProfile from a JSON file by ID.
func (s *Store) Load(id string) (*UnitProfile, error) {
	path := filepath.Join(s.dir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("profiles: load %q: %w", id, err)
	}
	var p UnitProfile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("profiles: parse %q: %w", id, err)
	}
	if p.ID == "" {
		p.ID = id
	}
	return &p, nil
}

// Save writes a UnitProfile to a JSON file.
func (s *Store) Save(p *UnitProfile) error {
	if p.ID == "" {
		return fmt.Errorf("profiles: save: profile has no ID")
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dir, p.ID+".json"), data, 0o644)
}

// List returns all valid UnitProfiles found in the store directory.
func (s *Store) List() ([]*UnitProfile, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []*UnitProfile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		p, err := s.Load(id)
		if err != nil {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

// Delete removes a profile by ID.
func (s *Store) Delete(id string) error {
	return os.Remove(filepath.Join(s.dir, id+".json"))
}

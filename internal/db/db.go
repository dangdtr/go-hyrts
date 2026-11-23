// Package db stores what a collection run learned: the module snapshot and,
// for every test, the functions it executed and the files it read.
package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dangdtr/go-hyrts/internal/snapshot"
)

// Version is bumped whenever the format changes; older databases are ignored.
const Version = 1

// Test is what one top-level test did during collection.
type Test struct {
	Pkg     string   `json:"pkg"`
	Name    string   `json:"name"`
	Status  string   `json:"status"` // pass, fail, skip, norun
	Seconds float64  `json:"sec"`
	Funcs   []int    `json:"funcs,omitempty"` // indexes into DB.Funcs
	Paths   []string `json:"paths,omitempty"` // non-directory files it opened or stat'ed
	Dirs    []string `json:"dirs,omitempty"`  // directories it opened or stat'ed
	Env     []string `json:"env,omitempty"`   // environment variables it read
}

// ID is the key of a test: package import path and name.
func ID(pkg, name string) string { return pkg + "." + name }

// DB is the persisted collection result.
type DB struct {
	Version   int               `json:"version"`
	GoVersion string            `json:"go"`
	Tags      string            `json:"tags,omitempty"`
	Module    string            `json:"module"`
	Commit    string            `json:"commit,omitempty"`
	Collected time.Time         `json:"collected"`
	Files     snapshot.Snapshot `json:"files"`
	Funcs     []string          `json:"func_ids"` // "rel/path.go#Key"
	Tests     map[string]*Test  `json:"tests"`
}

// DefaultPath is where the database lives relative to the module root.
const DefaultPath = ".hyrts/db.json"

// Load reads a database; it returns an error if it is missing or stale.
func Load(path string) (*DB, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var d DB
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	if d.Version != Version {
		return nil, fmt.Errorf("%s: format version %d, want %d; run hyrts collect", path, d.Version, Version)
	}
	return &d, nil
}

// Save writes the database atomically, with a .gitignore so it is not committed by accident.
func (d *DB) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	gi := filepath.Join(filepath.Dir(path), ".gitignore")
	if _, err := os.Stat(gi); os.IsNotExist(err) {
		_ = os.WriteFile(gi, []byte("*\n"), 0o644)
	}
	data, err := json.Marshal(d)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

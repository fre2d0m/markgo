// Package session keeps recovery copies separate from the user's documents.
package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"
)

const (
	SchemaVersion = 1
	MaxBytes      = 64 << 20
	MaxDocuments  = 100
	MaxRecent     = 20
)

type Document struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Path     string  `json:"path,omitempty"`
	Dirty    bool    `json:"dirty,omitempty"`
	Source   string  `json:"source,omitempty"`
	Original string  `json:"original,omitempty"`
	Version  uint64  `json:"bufferVersion,omitempty"`
	Anchor   int     `json:"anchor,omitempty"`
	Head     int     `json:"head,omitempty"`
	ScrollY  float32 `json:"scrollY,omitempty"`
}

type State struct {
	Schema    int        `json:"schema"`
	Documents []Document `json:"documents"`
	Active    string     `json:"active,omitempty"`
	Recent    []string   `json:"recent,omitempty"`
	Folder    string     `json:"folder,omitempty"`
	Sidebar   bool       `json:"sidebar"`
	Reading   bool       `json:"reading"`
	Source    bool       `json:"sourceMode"`
	Outline   bool       `json:"outline"`
}

// Store is used by one application instance. Requests can arrive out of order;
// an older worker must never resurrect a discarded or already-saved revision.
type Store struct {
	Path    string
	mu      sync.Mutex
	latest  uint64
	replace func(string, string) error
}

func (s *Store) Load() (State, error) {
	state := State{Schema: SchemaVersion, Sidebar: true}
	info, err := os.Lstat(s.Path)
	if errors.Is(err, fs.ErrNotExist) {
		s.cleanOrphans()
		return state, nil
	}
	if err != nil {
		return state, err
	}
	if !info.Mode().IsRegular() || info.Size() > MaxBytes {
		return state, errors.New("invalid or oversized recovery file; it was left untouched")
	}
	f, err := os.Open(s.Path)
	if err != nil {
		return state, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxBytes+1))
	if err != nil {
		return state, err
	}
	if len(data) > MaxBytes || !utf8.Valid(data) {
		return state, errors.New("invalid recovery data; it was left untouched")
	}
	state = State{}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, fmt.Errorf("read recovery data: %w", err)
	}
	if err := validate(state); err != nil {
		return state, err
	}
	s.cleanOrphans()
	return state, nil
}

func (s *Store) Save(revision uint64, state State) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if revision <= s.latest {
		return nil
	}
	s.latest = revision
	if err := validate(state); err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if len(data) > MaxBytes {
		return errors.New("recovery exceeds 64 MiB; save your documents to disk")
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	if info, err := os.Lstat(s.Path); err == nil && !info.Mode().IsRegular() {
		return errors.New("recovery destination is not a regular file")
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.Path), ".recovery-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	replace := s.replace
	if replace == nil {
		replace = os.Rename
	}
	return replace(f.Name(), s.Path)
}

// Called only during startup, after acquiring the application instance lock.
// Unreadable journals keep temporary data intact for manual recovery.
func (s *Store) cleanOrphans() {
	entries, err := os.ReadDir(filepath.Dir(s.Path))
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.Type().IsRegular() && strings.HasPrefix(entry.Name(), ".recovery-") && strings.HasSuffix(entry.Name(), ".tmp") {
			_ = os.Remove(filepath.Join(filepath.Dir(s.Path), entry.Name()))
		}
	}
}

func validate(state State) error {
	if state.Schema != SchemaVersion {
		return errors.New("unsupported recovery format; it was left untouched")
	}
	if len(state.Documents) > MaxDocuments || len(state.Recent) > MaxRecent {
		return errors.New("recovery supports up to 100 open and 20 recent documents")
	}
	ids, paths := map[string]bool{}, map[string]bool{}
	size := 0
	for _, doc := range state.Documents {
		if doc.ID == "" || ids[doc.ID] || doc.Name == "" || !validPath(doc.Path) || (doc.Path != "" && paths[doc.Path]) {
			return errors.New("invalid or duplicate recovery document")
		}
		ids[doc.ID], paths[doc.Path] = true, true
		if doc.Anchor < 0 || doc.Head < 0 || doc.ScrollY < 0 || math.IsInf(float64(doc.ScrollY), 0) || math.IsNaN(float64(doc.ScrollY)) {
			return errors.New("invalid recovery position")
		}
		if !utf8.ValidString(doc.Source) || !utf8.ValidString(doc.Original) {
			return errors.New("invalid recovery text")
		}
		size += len(doc.Source) + len(doc.Original)
	}
	if size > MaxBytes {
		return errors.New("recovery exceeds 64 MiB; save your documents to disk")
	}
	seenRecent := map[string]bool{}
	for _, path := range state.Recent {
		if path == "" || !validPath(path) || seenRecent[path] {
			return errors.New("invalid recovery path")
		}
		seenRecent[path] = true
	}
	if !validPath(state.Folder) {
		return errors.New("invalid recovery folder")
	}
	return nil
}

func validPath(path string) bool { return path == "" || filepath.IsAbs(path) }

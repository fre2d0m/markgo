// Package preferences persists small application preferences independently of documents.
package preferences

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

type Store struct {
	Path   string
	mu     sync.Mutex
	latest uint64
}

func (s *Store) Load() ([]byte, error) {
	info, err := os.Lstat(s.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > 64*1024 {
		return nil, errors.New("invalid preferences file")
	}
	return os.ReadFile(s.Path)
}

// Save rejects older requests even when their workers arrive out of order.
func (s *Store) Save(revision uint64, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if revision <= s.latest {
		return nil
	}
	s.latest = revision
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.Path), ".appearance-*.tmp")
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
	return os.Rename(f.Name(), s.Path)
}

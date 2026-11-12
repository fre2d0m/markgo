// Package storage reads and writes Markdown files without changing their bytes
// or silently overwriting changes made by another application.
package storage

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"unicode/utf8"
)

var (
	ErrExternalChange = errors.New("the file changed outside MarkGo")
	ErrNotRegular     = errors.New("the path is not a regular file")
	ErrInvalidUTF8    = errors.New("the file is not valid UTF-8")
	ErrAlreadyExists  = errors.New("the destination already exists")
)

// Snapshot identifies the exact file content most recently opened or saved.
// Keep it with the document; do not construct one from a pathname alone.
type Snapshot struct {
	Path   string
	digest [sha256.Size]byte
	mode   fs.FileMode
}

// Open reads an existing regular UTF-8 file, preserving its original bytes.
// Symbolic links are rejected so atomic replacement cannot replace a link
// instead of the file the user intended to edit.
func Open(path string) (Snapshot, []byte, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Snapshot{}, nil, err
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return Snapshot{}, nil, err
	}
	if !info.Mode().IsRegular() {
		return Snapshot{}, nil, ErrNotRegular
	}
	data, err := os.ReadFile(absolute)
	if err != nil {
		return Snapshot{}, nil, err
	}
	if !utf8.Valid(data) {
		return Snapshot{}, nil, ErrInvalidUTF8
	}
	return Snapshot{Path: absolute, digest: sha256.Sum256(data), mode: info.Mode().Perm()}, data, nil
}

// Save replaces an existing file only if it still matches the snapshot. The
// contents are written to a temporary file in the same directory first.
func Save(snapshot Snapshot, data []byte) (Snapshot, error) {
	if snapshot.Path == "" {
		return Snapshot{}, errors.New("missing original file snapshot")
	}
	if !utf8.Valid(data) {
		return Snapshot{}, ErrInvalidUTF8
	}
	if snapshot.mode.Perm()&0o222 == 0 {
		return Snapshot{}, fs.ErrPermission
	}
	if err := unchanged(snapshot); err != nil {
		return Snapshot{}, err
	}
	name, err := writeTemp(snapshot.Path, data, snapshot.mode)
	if err != nil {
		return Snapshot{}, err
	}
	defer os.Remove(name)
	// A save can take time, so check again immediately before replacement.
	if err := unchanged(snapshot); err != nil {
		return Snapshot{}, err
	}
	if err := os.Rename(name, snapshot.Path); err != nil {
		return Snapshot{}, err
	}
	return Snapshot{Path: snapshot.Path, digest: sha256.Sum256(data), mode: snapshot.mode}, nil
}

// Check reports whether the file still matches a previously loaded snapshot.
func Check(snapshot Snapshot) error {
	if snapshot.Path == "" {
		return errors.New("missing original file snapshot")
	}
	return unchanged(snapshot)
}

// Create saves a new file without overwriting an existing path, even if the
// path is created by another process while this save is in progress.
func Create(path string, data []byte) (Snapshot, error) {
	if !utf8.Valid(data) {
		return Snapshot{}, ErrInvalidUTF8
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Snapshot{}, err
	}
	name, err := writeTemp(absolute, data, 0o600)
	if err != nil {
		return Snapshot{}, err
	}
	defer os.Remove(name)
	// Linking a completed temporary file makes creation exclusive and avoids
	// exposing a partially written document. Both paths are in the same folder.
	if err := os.Link(name, absolute); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return Snapshot{}, ErrAlreadyExists
		}
		return Snapshot{}, err
	}
	return Snapshot{Path: absolute, digest: sha256.Sum256(data), mode: 0o600}, nil
}

func unchanged(snapshot Snapshot) error {
	info, err := os.Lstat(snapshot.Path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ErrExternalChange
		}
		return err
	}
	if !info.Mode().IsRegular() {
		return ErrExternalChange
	}
	data, err := os.ReadFile(snapshot.Path)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	if !bytes.Equal(digest[:], snapshot.digest[:]) || info.Mode().Perm() != snapshot.mode {
		return ErrExternalChange
	}
	return nil
}

func writeTemp(path string, data []byte, mode fs.FileMode) (string, error) {
	f, err := os.CreateTemp(filepath.Dir(path), ".markgo-*.tmp")
	if err != nil {
		return "", err
	}
	name := f.Name()
	failed := func(err error) (string, error) {
		_ = f.Close()
		_ = os.Remove(name)
		return "", err
	}
	if err := f.Chmod(mode.Perm()); err != nil {
		return failed(err)
	}
	if _, err := f.Write(data); err != nil {
		return failed(err)
	}
	if err := f.Sync(); err != nil {
		return failed(err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(name)
		return "", fmt.Errorf("close temporary file: %w", err)
	}
	return name, nil
}

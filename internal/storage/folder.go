package storage

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Entry is a directory or Markdown file in a local workspace.
type Entry struct {
	Name      string
	Path      string
	Directory bool
	Children  []Entry
}

var ErrWorkspaceTooLarge = errors.New("the folder contains too many entries to display")

// ListFolder returns a stable tree without following symbolic links. Hidden
// entries are omitted. Call it off the UI thread for user-selected folders.
func ListFolder(path string) (Entry, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Entry{}, err
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return Entry{}, err
	}
	if !info.IsDir() {
		return Entry{}, ErrNotRegular
	}
	root := Entry{Name: filepath.Base(absolute), Path: absolute, Directory: true}
	count := 0
	root.Children, err = folderChildren(absolute, &count)
	return root, err
}

func folderChildren(path string, count *int) ([]Entry, error) {
	items, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var entries []Entry
	for _, item := range items {
		if strings.HasPrefix(item.Name(), ".") || item.Type()&os.ModeSymlink != 0 {
			continue
		}
		*count++
		if *count > 10000 {
			return nil, ErrWorkspaceTooLarge
		}
		child := Entry{Name: item.Name(), Path: filepath.Join(path, item.Name()), Directory: item.IsDir()}
		if child.Directory {
			child.Children, err = folderChildren(child.Path, count)
			if err != nil {
				return nil, err
			}
		} else {
			extension := strings.ToLower(filepath.Ext(item.Name()))
			if extension != ".md" && extension != ".markdown" {
				continue
			}
		}
		entries = append(entries, child)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Directory != entries[j].Directory {
			return entries[i].Directory
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	return entries, nil
}

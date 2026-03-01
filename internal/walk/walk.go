package walk

import (
	"io/fs"
	"path/filepath"
)

// Based on https://github.com/golang/go/issues/64341

type DirEntry struct {
	Path    string
	Entry   fs.DirEntry
	skipDir *bool
}

func (entry DirEntry) SkipDir() {
	*entry.skipDir = true
}

func DirIter(root string) func(func(DirEntry, error) bool) {
	var skipDir bool // outside the loop so we'll only allocate once.
	return func(yield func(DirEntry, error) bool) {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			info := DirEntry{
				Path:    path,
				Entry:   d,
				skipDir: &skipDir,
			}
			skipDir = false
			if !yield(info, err) {
				return fs.SkipAll
			}
			if skipDir {
				return fs.SkipDir
			}
			return nil
		})
		if err != nil {
			yield(DirEntry{}, err)
		}
	}
}

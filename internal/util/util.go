package util

import (
	"io/fs"
	"path/filepath"
)

// Based on https://github.com/golang/go/issues/64341

type WalkDirEntry struct {
	Path    string
	Entry   fs.DirEntry
	skipDir *bool
}

func (entry WalkDirEntry) SkipDir() {
	*entry.skipDir = true
}

func WalkDirIter(root string) func(func(WalkDirEntry, error) bool) {
	var skipDir bool // outside the loop so we'll only allocate once.
	return func(yield func(WalkDirEntry, error) bool) {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			info := WalkDirEntry{
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
			yield(WalkDirEntry{}, err)
		}
	}
}

func Keys[K comparable, V any](m map[K]V) []K {
	var ks []K
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

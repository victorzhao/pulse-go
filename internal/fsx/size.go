package fsx

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
)

// DirSize sums the size of the files under path, skipping symlinks. It
// reports how many files it counted. It stops when ctx is done.
func DirSize(ctx context.Context, path string, limit int) (size int64, files int, err error) {
	size, files, _, err = DirStats(ctx, path, limit)
	return size, files, err
}

// DirStats sums a folder recursively, counting its files and folders. It
// skips symlinks and stops at limit files, or when ctx is done.
func DirStats(ctx context.Context, path string, limit int) (size int64, files, dirs int, err error) {
	root := filepath.Clean(path)
	err = filepath.WalkDir(path, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			if filepath.Clean(p) != root {
				dirs++
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if info, e := d.Info(); e == nil {
			size += info.Size()
			files++
			if limit > 0 && files >= limit {
				return fs.SkipAll
			}
		}
		return nil
	})
	if err == context.Canceled || err == context.DeadlineExceeded {
		err = nil
	}
	return size, files, dirs, err
}

// SearchFilter selects files by name, kind, size and modification time.
type SearchFilter struct {
	Query         string
	Root          string
	MaxDepth      int
	Kinds         []Kind
	MinSize       int64
	MaxSize       int64
	After         int64 // Unix seconds, 0 for any
	Before        int64
	IncludeHidden bool
	Limit         int
	FoldersOnly   bool
	FilesOnly     bool
}

// Search walks root and calls emit for every entry matching filter. It stops
// at filter.Limit, when ctx is done, or when emit returns false.
func Search(ctx context.Context, f SearchFilter, emit func(Entry) bool) error {
	if f.MaxDepth <= 0 {
		f.MaxDepth = 16
	}
	if f.Limit <= 0 {
		f.Limit = 5000
	}
	root := f.Root
	if root == "" {
		root = "."
	}
	found := 0
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		rel, _ := filepath.Rel(root, p)
		if rel != "." {
			depth := 0
			for _, r := range rel {
				if r == filepath.Separator {
					depth++
				}
			}
			if depth >= f.MaxDepth {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
		}
		if p == root {
			return nil
		}
		if !f.IncludeHidden && IsHidden(d.Name()) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if f.FilesOnly {
				return nil
			}
		} else if f.FoldersOnly {
			return nil
		}
		if f.Query != "" && !MatchName(d.Name(), f.Query) {
			return nil
		}
		var e Entry
		if info, err := d.Info(); err == nil {
			e = Entry{
				Name:    d.Name(),
				Path:    p,
				IsDir:   d.IsDir(),
				Size:    info.Size(),
				ModTime: info.ModTime(),
				Mode:    info.Mode(),
				Hidden:  IsHiddenPath(d.Name(), p),
				Link:    info.Mode()&os.ModeSymlink != 0,
			}
			if e.IsDir {
				e.Size = 0
				e.Kind = KindFolder
			} else {
				e.Ext = filepath.Ext(e.Name)
				if e.Ext != "" {
					e.Ext = e.Ext[1:]
				}
				e.Kind = KindOf(e.Ext)
			}
		} else {
			return nil
		}
		if f.MinSize > 0 && !e.IsDir && e.Size < f.MinSize {
			return nil
		}
		if f.MaxSize > 0 && !e.IsDir && e.Size > f.MaxSize {
			return nil
		}
		if f.After > 0 && e.ModTime.Unix() < f.After {
			return nil
		}
		if f.Before > 0 && e.ModTime.Unix() > f.Before {
			return nil
		}
		if len(f.Kinds) > 0 && !e.IsDir {
			ok := false
			for _, k := range f.Kinds {
				if e.Kind == k {
					ok = true
					break
				}
			}
			if !ok {
				return nil
			}
		}
		found++
		if !emit(e) {
			return fs.SkipAll
		}
		if found >= f.Limit {
			return fs.SkipAll
		}
		return nil
	})
	if err == context.Canceled || err == context.DeadlineExceeded {
		err = nil
	}
	return err
}

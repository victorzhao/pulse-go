package platform

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// TrashItem is a deleted item that can be restored.
type TrashItem struct {
	ID        string    `json:"id"`
	Original  string    `json:"original"`
	Name      string    `json:"name"`
	Stored    string    `json:"stored"`
	IsDir     bool      `json:"is_dir"`
	Size      int64     `json:"size"`
	DeletedAt time.Time `json:"deleted_at"`
}

// Trash is Pulse's own recycle bin: a folder under the configuration
// directory with a manifest, so that deleted items can be listed and
// restored on every platform.
type Trash struct {
	mu   sync.Mutex
	dir  string
	file string
}

// NewTrash returns the trash of this installation.
func NewTrash() *Trash {
	dir := filepath.Join(ConfigDir(), "trash")
	_ = os.MkdirAll(dir, 0o755)
	return &Trash{dir: dir, file: filepath.Join(dir, "manifest.json")}
}

// Dir returns where deleted items are stored.
func (t *Trash) Dir() string { return t.dir }

func (t *Trash) load() ([]TrashItem, error) {
	data, err := os.ReadFile(t.file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var items []TrashItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (t *Trash) save(items []TrashItem) error {
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	tmp := t.file + ".tmp"
	_ = os.Remove(tmp)
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, t.file); err != nil {
		if err2 := os.WriteFile(t.file, data, 0o644); err2 != nil {
			return err
		}
	}
	_ = os.Remove(tmp)
	return nil
}

// Delete moves a path to the trash.
func (t *Trash) Delete(path string) (TrashItem, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	info, err := os.Lstat(path)
	if err != nil {
		return TrashItem{}, err
	}
	items, err := t.load()
	if err != nil {
		return TrashItem{}, err
	}
	id := fmt.Sprintf("%d-%d", time.Now().UnixNano(), len(items))
	stored := filepath.Join(t.dir, id+"-"+filepath.Base(path))
	if err := os.Rename(path, stored); err != nil {
		// Cross-device: copy and then remove.
		if err := copyPath(path, stored); err != nil {
			return TrashItem{}, err
		}
		if err := os.RemoveAll(path); err != nil {
			_ = os.RemoveAll(stored)
			return TrashItem{}, err
		}
	}
	size := info.Size()
	if info.IsDir() {
		size = 0
	}
	item := TrashItem{
		ID:        id,
		Original:  path,
		Name:      filepath.Base(path),
		Stored:    stored,
		IsDir:     info.IsDir(),
		Size:      size,
		DeletedAt: time.Now(),
	}
	items = append(items, item)
	if err := t.save(items); err != nil {
		return item, err
	}
	return item, nil
}

// List returns the items in the trash, newest first.
func (t *Trash) List() ([]TrashItem, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	items, err := t.load()
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].DeletedAt.After(items[j].DeletedAt) })
	return items, nil
}

// Restore puts an item back where it came from.
func (t *Trash) Restore(id string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	items, err := t.load()
	if err != nil {
		return err
	}
	idx := -1
	for i, item := range items {
		if item.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return errors.New("item not found")
	}
	item := items[idx]
	target := uniquePath(item.Original)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := os.Rename(item.Stored, target); err != nil {
		if err := copyPath(item.Stored, target); err != nil {
			return err
		}
		if err := os.RemoveAll(item.Stored); err != nil {
			return err
		}
	}
	items = append(items[:idx], items[idx+1:]...)
	return t.save(items)
}

// Remove permanently deletes an item from the trash.
func (t *Trash) Remove(id string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	items, err := t.load()
	if err != nil {
		return err
	}
	idx := -1
	for i, item := range items {
		if item.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return errors.New("item not found")
	}
	if err := os.RemoveAll(items[idx].Stored); err != nil {
		return err
	}
	items = append(items[:idx], items[idx+1:]...)
	return t.save(items)
}

// Empty permanently deletes everything in the trash.
func (t *Trash) Empty() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	items, err := t.load()
	if err != nil {
		return err
	}
	for _, item := range items {
		_ = os.RemoveAll(item.Stored)
	}
	return t.save(nil)
}

// Size returns the number of items and their total known size.
func (t *Trash) Size() (count int, size int64) {
	items, err := t.List()
	if err != nil {
		return 0, 0
	}
	for _, item := range items {
		count++
		size += item.Size
	}
	return count, size
}

func uniquePath(path string) string {
	if !Exists(path) {
		return path
	}
	dir, name := filepath.Split(path)
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 1; i < 1000; i++ {
		cand := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
		if !Exists(cand) {
			return cand
		}
	}
	return path
}

func copyPath(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyTree(src, dst)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, info.Mode().Perm())
}

func copyTree(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, e := range entries {
		if err := copyPath(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

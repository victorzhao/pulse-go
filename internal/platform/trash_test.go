package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func testTrash(t *testing.T) *Trash {
	t.Helper()
	dir := t.TempDir()
	trashDir := filepath.Join(dir, "trash")
	if err := os.MkdirAll(trashDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return &Trash{dir: trashDir, file: filepath.Join(trashDir, "manifest.json")}
}

func TestTrashDeleteRestore(t *testing.T) {
	tr := testTrash(t)
	src := filepath.Join(t.TempDir(), "note.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	item, err := tr.Delete(src)
	if err != nil {
		t.Fatal(err)
	}
	if Exists(src) {
		t.Fatal("source still exists")
	}
	if !Exists(item.Stored) {
		t.Fatal("stored item is missing")
	}
	items, err := tr.List()
	if err != nil || len(items) != 1 {
		t.Fatalf("list = %v, %v", items, err)
	}
	if err := tr.Restore(item.ID); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(src)
	if err != nil || string(data) != "hello" {
		t.Fatalf("restored data = %q, %v", data, err)
	}
	if _, err := tr.List(); err != nil {
		t.Fatal(err)
	}
}

func TestTrashRemoveAndEmpty(t *testing.T) {
	tr := testTrash(t)
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	os.WriteFile(a, []byte("a"), 0o644)
	os.WriteFile(b, []byte("b"), 0o644)
	ia, _ := tr.Delete(a)
	ib, _ := tr.Delete(b)
	if err := tr.Remove(ia.ID); err != nil {
		t.Fatal(err)
	}
	items, _ := tr.List()
	if len(items) != 1 || items[0].ID != ib.ID {
		t.Fatalf("items = %v", items)
	}
	if err := tr.Empty(); err != nil {
		t.Fatal(err)
	}
	items, _ = tr.List()
	if len(items) != 0 {
		t.Fatalf("items after empty = %v", items)
	}
}

func TestTrashFolder(t *testing.T) {
	tr := testTrash(t)
	dir := t.TempDir()
	folder := filepath.Join(dir, "folder")
	os.MkdirAll(folder, 0o755)
	os.WriteFile(filepath.Join(folder, "x"), []byte("x"), 0o644)
	item, err := tr.Delete(folder)
	if err != nil {
		t.Fatal(err)
	}
	if !item.IsDir {
		t.Fatal("not a folder item")
	}
	if Exists(folder) {
		t.Fatal("folder still exists")
	}
	if err := tr.Restore(item.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(folder, "x")); err != nil {
		t.Fatal(err)
	}
}

package app

import (
	"os"
	"path/filepath"
	"testing"

	"pulse/internal/fsx"
)

func TestRenameManySwap(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	os.WriteFile(a, []byte("A"), 0o644)
	os.WriteFile(b, []byte("B"), 0o644)
	if err := renameMany([][2]string{{a, b}, {b, a}}); err != nil {
		t.Fatal(err)
	}
	da, _ := os.ReadFile(a)
	db, _ := os.ReadFile(b)
	if string(da) != "B" || string(db) != "A" {
		t.Fatalf("a=%q b=%q", da, db)
	}
}

func TestBatchPreviewAndApply(t *testing.T) {
	dir := t.TempDir()
	var entries []fsx.Entry
	for _, name := range []string{"one.txt", "two.txt"} {
		path := filepath.Join(dir, name)
		os.WriteFile(path, []byte(name), 0o644)
		e, err := fsx.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, e)
	}
	a := testApp(t)
	p := a.activePane()
	a.batch = batchRenameState{Open: true, Pane: p, Entries: entries, Prefix: "pre-", Start: 1, Digits: 2}
	a.batchPreview()
	if a.batch.Error != "" {
		t.Fatalf("preview error: %s", a.batch.Error)
	}
	if a.batch.Preview[0] != "pre-one.txt" || a.batch.Preview[1] != "pre-two.txt" {
		t.Fatalf("preview = %v", a.batch.Preview)
	}
	a.applyBatchRename()
	if !osPathExists(filepath.Join(dir, "pre-one.txt")) || !osPathExists(filepath.Join(dir, "pre-two.txt")) {
		t.Fatal("renamed files are missing")
	}
}

func TestDirFingerprint(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a"), []byte("a"), 0o644)
	entries, _ := fsx.ReadDir(dir)
	fp := dirFingerprint(entries)
	os.WriteFile(filepath.Join(dir, "b"), []byte("b"), 0o644)
	entries, _ = fsx.ReadDir(dir)
	if dirFingerprint(entries) == fp {
		t.Fatal("fingerprint did not change")
	}
}

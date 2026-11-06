package app

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyMoveAndUnique(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "b.txt")
	if err := copyPath(src, dst); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dst)
	if err != nil || string(data) != "hello" {
		t.Fatalf("copied data = %q, %v", data, err)
	}
	moved := filepath.Join(dir, "c.txt")
	if err := movePath(dst, moved); err != nil {
		t.Fatal(err)
	}
	if osPathExists(dst) {
		t.Fatal("moved source still exists")
	}
	if _, err := os.Stat(moved); err != nil {
		t.Fatal(err)
	}
	u := uniqueDest(src)
	if u == src {
		t.Fatal("uniqueDest returned the existing path")
	}
	if _, err := os.Stat(u); !os.IsNotExist(err) {
		t.Fatal("unique destination exists")
	}
}

func TestCopyFolder(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	os.MkdirAll(filepath.Join(src, "sub"), 0o755)
	os.WriteFile(filepath.Join(src, "sub", "x.txt"), []byte("x"), 0o644)
	dst := filepath.Join(dir, "dst")
	if err := copyPath(src, dst); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dst, "sub", "x.txt"))
	if err != nil || string(data) != "x" {
		t.Fatalf("copied tree = %q, %v", data, err)
	}
}

func TestRenameEntry(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "old.txt")
	os.WriteFile(src, []byte("x"), 0o644)
	p := a.activePane()
	a.renameEntry(p, src, "new.txt")
	if osPathExists(src) {
		t.Fatal("old name still exists")
	}
	if _, err := os.Stat(filepath.Join(dir, "new.txt")); err != nil {
		t.Fatal(err)
	}
}

func TestMergeCopy(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	os.MkdirAll(filepath.Join(src, "sub"), 0o755)
	os.MkdirAll(filepath.Join(dst, "sub"), 0o755)
	os.MkdirAll(filepath.Join(dst, "keep"), 0o755)
	os.WriteFile(filepath.Join(src, "a.txt"), []byte("new"), 0o644)
	os.WriteFile(filepath.Join(src, "sub", "b.txt"), []byte("b"), 0o644)
	os.WriteFile(filepath.Join(dst, "a.txt"), []byte("old"), 0o644)
	os.WriteFile(filepath.Join(dst, "keep", "k.txt"), []byte("k"), 0o644)
	if err := mergeCopy(src, dst); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dst, "a.txt"))
	if string(data) != "new" {
		t.Fatalf("a.txt = %q", data)
	}
	if _, err := os.Stat(filepath.Join(dst, "sub", "b.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dst, "keep", "k.txt")); err != nil {
		t.Fatal("existing folder was lost")
	}
}

func TestNewFile(t *testing.T) {
	a := testApp(t)
	p := a.activePane()
	p.Path = t.TempDir()
	p.Virtual = virtualNone
	a.newFolder = newFolderState{Open: true, Name: "note.txt", Pane: p, File: true}
	a.commitNewFolder()
	if !osPathExists(filepath.Join(p.Path, "note.txt")) {
		t.Fatal("new file is missing")
	}
}

func TestConflictQueue(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	src1 := filepath.Join(dir, "a.txt")
	src2 := filepath.Join(dir, "b.txt")
	dst1 := filepath.Join(dir, "dst", "a.txt")
	dst2 := filepath.Join(dir, "dst", "b.txt")
	os.MkdirAll(filepath.Dir(dst1), 0o755)
	for _, p := range []string{src1, src2, dst1, dst2} {
		os.WriteFile(p, []byte("x"), 0o644)
	}
	items := []conflictItem{{src: src1, dst: dst1}, {src: src2, dst: dst2}}
	choices := map[string]int{}
	job := transferJob{srcs: []string{src1, src2}, dstDir: filepath.Dir(dst1)}
	a.beginConflict(job, items, choices, 0)
	if !a.conflict.Open || a.conflict.Path != dst1 {
		t.Fatalf("first conflict = %+v", a.conflict)
	}
	a.conflict.Resume(1, false)
	if choices[src1] != 1 {
		t.Fatalf("choices = %v", choices)
	}
	if !a.conflict.Open || a.conflict.Path != dst2 {
		t.Fatalf("second conflict = %+v", a.conflict)
	}
	a.conflict.Resume(0, false)
	if choices[src2] != 0 {
		t.Fatalf("choices = %v", choices)
	}
}

func TestConflictApplyAll(t *testing.T) {
	a := testApp(t)
	dir := t.TempDir()
	items := []conflictItem{}
	choices := map[string]int{}
	for i := 0; i < 3; i++ {
		src := filepath.Join(dir, "src", fmt.Sprintf("%d.txt", i))
		dst := filepath.Join(dir, "dst", fmt.Sprintf("%d.txt", i))
		items = append(items, conflictItem{src: src, dst: dst})
	}
	a.beginConflict(transferJob{}, items, choices, 0)
	a.conflict.Resume(2, true)
	for _, item := range items {
		if choices[item.src] != 2 {
			t.Fatalf("choices = %v", choices)
		}
	}
}

package fsx

import (
	"context"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestPinyinMatch(t *testing.T) {
	cases := []struct {
		name, query string
		want        bool
	}{
		{"品牌设计", "sh", true},
		{"品牌设计", "pinpai", true},
		{"品牌设计", "ppsj", true},
		{"城市夜景", "cs", true},
		{"山湖晨雾", "sh", true},
		{"项目计划.md", "xiangmu", true},
		{"项目计划.md", "xm", true},
		{"report.pdf", "port", true},
		{"report.pdf", "xyz", false},
	}
	for _, c := range cases {
		if got := MatchName(c.name, c.query); got != c.want {
			t.Errorf("MatchName(%q, %q) = %v, want %v", c.name, c.query, got, c.want)
		}
	}
}

func TestMatchRanges(t *testing.T) {
	ranges, ok := MatchRanges("品牌设计", "sheji")
	if !ok || len(ranges) == 0 {
		t.Fatalf("no pinyin ranges: %v %v", ranges, ok)
	}
	ranges, ok = MatchRanges("report.pdf", "port")
	if !ok || len(ranges) != 1 || ranges[0] != [2]int{2, 6} {
		t.Fatalf("direct ranges = %v %v", ranges, ok)
	}
}

func TestFormatSize(t *testing.T) {
	cases := map[int64]string{
		183:       "183 B",
		2765:      "2.7 KB",
		232448:    "227.0 KB",
		3_600_000: "3.4 MB",
	}
	for n, want := range cases {
		if got := FormatSize(n); got != want {
			t.Errorf("FormatSize(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestDiffLines(t *testing.T) {
	rows := DiffLines("a\nb\nc\n", "a\nx\nc\n")
	_, removed, added := DiffStats(rows)
	if removed != 1 || added != 1 {
		t.Fatalf("removed %d added %d: %+v", removed, added, rows)
	}
	if rows[1].Kind != '-' || rows[1].A != "b" {
		t.Errorf("row 1 = %+v", rows[1])
	}
	if rows[2].Kind != '+' || rows[2].B != "x" {
		t.Errorf("row 2 = %+v", rows[2])
	}
}

func TestReadDirAndSize(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %v", entries)
	}
	size, files, dirs, err := DirStats(context.Background(), dir, 0)
	if err != nil {
		t.Fatal(err)
	}
	if size != 5 || files != 1 || dirs != 1 {
		t.Errorf("size %d files %d dirs %d", size, files, dirs)
	}
}

func TestSearch(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "品牌设计"), 0o755)
	os.WriteFile(filepath.Join(dir, "城市夜景.jpg"), []byte("x"), 0o644)
	var names []string
	err := Search(context.Background(), SearchFilter{Root: dir, Query: "sh", MaxDepth: 4}, func(e Entry) bool {
		names = append(names, e.Name)
		return true
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 {
		t.Fatalf("names = %v", names)
	}
}

func TestImageDimensions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "img.png")
	img := image.NewNRGBA(image.Rect(0, 0, 13, 7))
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()
	w, h, ok := ImageDimensions(path)
	if !ok || w != 13 || h != 7 {
		t.Fatalf("dimensions = %d x %d, %v", w, h, ok)
	}
}

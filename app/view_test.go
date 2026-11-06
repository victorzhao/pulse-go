package app

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"pulse/internal/fsx"
	"pulse/internal/platform"
	"pulse/internal/store"

	"github.com/egoist/mygo/ui"
)

// testApp builds an app state with a few entries, for rendering the view
// without a window.
func testApp(t *testing.T) *App {
	t.Helper()
	cfg := store.Default()
	cfg.Language = "zh-Hans"
	cfg.View.ShowHidden = true
	cfg.View.ComputeFolderSize = false
	a := &App{
		cfg:              cfg,
		trash:            platform.NewTrash(),
		locale:           "zh-Hans",
		systemDark:       true,
		sidebarOn:        true,
		sidebarWidth:     250,
		detailsOn:        true,
		detailsPercent:   70,
		detailsBasicOpen: true,
		detailsTagsOpen:  true,
		sizes:            map[string]sizeCache{},
		thumb:            thumbCache{},
	}
	a.thumb.init()
	a.side = newSideState()
	home := platform.Home()
	now := time.Now()
	dirs := []string{"产品文档", "发布", "品牌设计", "源代码", "照片"}
	entries := make([]fsx.Entry, 0, 12)
	for i, name := range dirs {
		entries = append(entries, fsx.Entry{
			Name: name, Path: filepath.Join(home, "Demo", name), IsDir: true,
			Kind: fsx.KindFolder, ModTime: now.Add(-time.Duration(i) * time.Hour),
		})
	}
	files := []struct {
		name string
		ext  string
		kind fsx.Kind
		size int64
	}{
		{"待办清单.txt", "txt", fsx.KindText, 78},
		{"季度销售数据.csv", "csv", fsx.KindSheet, 2765},
		{"项目计划.md", "md", fsx.KindText, 183},
		{"城市夜景.jpg", "jpg", fsx.KindImage, 325200},
		{"配置-v2.3.json", "json", fsx.KindCode, 412},
	}
	for i, f := range files {
		entries = append(entries, fsx.Entry{
			Name: f.name, Path: filepath.Join(home, "Demo", f.name),
			Ext: f.ext, Kind: f.kind, Size: f.size,
			ModTime: now.Add(-time.Duration(i) * 2 * time.Hour),
		})
	}
	p := &Pane{
		Path:     filepath.Join(home, "Demo"),
		View:     viewDetails,
		SortBy:   "name",
		Group:    "none",
		Selected: 0,
		Sizes:    map[string]int64{},
		Compare:  map[string]compareMark{},
	}
	p.Entries = entries
	p.History = []string{p.Path}
	a.tabs = []*Tab{{Title: "Demo", Layout: layoutSingle, Panes: []*Pane{p}, SplitA: 50, SplitB: 50, SplitV: 50}}
	a.tab = 0
	a.cfg.TagAssignments[p.Entries[5].Path] = "design"
	a.applyPane(p)
	return a
}

func TestViewSnapshot(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.View, 1280, 820)
	tt.Frame()
	if !tt.HasText("产品文档") {
		t.Fatalf("the file list is missing; texts: %v", tt.Texts())
	}
	img := tt.Image()
	if img == nil {
		t.Fatal("no image")
	}
	if err := os.MkdirAll("build", 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join("build", "view.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

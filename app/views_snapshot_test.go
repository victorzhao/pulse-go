package app

import (
	"bytes"
	"image"
	imagecolor "image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"pulse/internal/fsx"
	"pulse/internal/platform"

	"github.com/egoist/mygo/ui"
)

func snapshot(t *testing.T, a *App, name string) {
	t.Helper()
	tt := ui.NewTester(a.View, 1280, 820)
	tt.Frame()
	img := tt.Image()
	if img == nil {
		t.Fatalf("%s: no image", name)
	}
	dir := filepath.Join("build", "shots")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestViewsSnapshots(t *testing.T) {
	for _, v := range []struct {
		name string
		mode viewMode
	}{
		{"details", viewDetails},
		{"tiles", viewTiles},
		{"icons", viewIcons},
		{"list", viewList},
		{"content", viewContent},
	} {
		a := testApp(t)
		p := a.activePane()
		p.View = v.mode
		p.Group = "none"
		p.Selected = 3
		p.Selection.Add(p.Entries[3].Path)
		a.applyPane(p)
		snapshot(t, a, "view-"+v.name)
	}
}

func TestLayoutSnapshot(t *testing.T) {
	a := testApp(t)
	tb := a.activeTab()
	p1 := tb.Panes[0]
	p2 := &Pane{
		Path:     p1.Path,
		View:     viewTiles,
		SortBy:   "name",
		Selected: -1,
		Sizes:    map[string]int64{},
		Compare:  map[string]compareMark{},
	}
	p2.Entries = p1.Entries
	p2.History = []string{p2.Path}
	tb.Panes = append(tb.Panes, p2)
	tb.Layout = layoutSplitH
	tb.Active = 1
	a.applyPane(p2)
	snapshot(t, a, "layout-split")
}

func TestSettingsSnapshot(t *testing.T) {
	a := testApp(t)
	a.settings.Open = true
	a.settings.Tab = 0
	snapshot(t, a, "settings-appearance")
}

func TestSearchSnapshot(t *testing.T) {
	a := testApp(t)
	a.search.Open = true
	a.search.Query = "项目"
	a.search.Done = 3
	home := platform.Home()
	a.search.Results = []fsx.Entry{
		{Name: "项目计划.md", Path: filepath.Join(home, "Demo", "项目计划.md"), Ext: "md", Kind: fsx.KindDocument, Size: 183, ModTime: time.Now()},
		{Name: "第四季度项目计划.md", Path: filepath.Join(home, "Docs", "第四季度项目计划.md"), Ext: "md", Kind: fsx.KindDocument, Size: 2048, ModTime: time.Now()},
		{Name: "项目文档", Path: filepath.Join(home, "Demo", "项目文档"), IsDir: true, Kind: fsx.KindFolder},
	}
	a.search.Selected = 0
	snapshot(t, a, "search")
}

func TestVerticalTabsSnapshot(t *testing.T) {
	a := testApp(t)
	a.cfg.View.VerticalTabs = true
	a.sidebarOn = true
	snapshot(t, a, "vertical-tabs")
}

func TestWallpaperSnapshot(t *testing.T) {
	a := testApp(t)
	img := image.NewNRGBA(image.Rect(0, 0, 256, 160))
	for y := 0; y < 160; y++ {
		for x := 0; x < 256; x++ {
			t := float64(x+y) / 416
			img.Set(x, y, imagecolor.NRGBA{R: uint8(40 + 180*t), G: uint8(90 + 60*t), B: uint8(180 - 90*t), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	bmp, err := ui.DecodeBitmap(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	a.wp.bmp = bmp
	a.cfg.Appearance.Blur = 40
	a.cfg.Appearance.Transparency = 100
	snapshot(t, a, "wallpaper")
}

func TestMultiSelectionSnapshot(t *testing.T) {
	a := testApp(t)
	p := a.activePane()
	for i := 0; i < 3; i++ {
		p.Selection.Add(p.Entries[i].Path)
	}
	p.Selected = 0
	snapshot(t, a, "multi-selection")
}

func TestBatchRenameSnapshot(t *testing.T) {
	a := testApp(t)
	p := a.activePane()
	for i := 0; i < 3; i++ {
		p.Selection.Add(p.Entries[i].Path)
	}
	a.beginBatchRename(p)
	a.batch.Prefix = "pre-"
	a.batch.Numbering = true
	a.batchPreview()
	snapshot(t, a, "batch-rename")
}

func TestStashSnapshot(t *testing.T) {
	a := testApp(t)
	home := platform.Home()
	a.stash.Items = []stashItem{
		{Path: filepath.Join(home, "配置-v2.2.json"), Name: "配置-v2.2.json", Size: 318, Move: true},
		{Path: filepath.Join(home, "配置-v2.3.json"), Name: "配置-v2.3.json", Size: 412},
	}
	snapshot(t, a, "stash")
}

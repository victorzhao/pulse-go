package app

import (
	"context"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"pulse/internal/fsx"
	"pulse/internal/platform"
	"pulse/internal/store"

	"github.com/egoist/mygo/ui"
)

// loadPane reads a pane's folder in the background and updates the view.
func (a *App) loadPane(p *Pane) {
	if p.watchCancel != nil {
		p.watchCancel()
		p.watchCancel = nil
	}
	p.Gen++
	gen := p.Gen
	p.Loading = true
	p.Err = ""
	p.SizeBusy = false
	p.Compare = map[string]compareMark{}
	p.CompareOn = false
	p.Selected = -1
	p.Selection.Clear()
	if p.Virtual != virtualNone {
		a.loadVirtual(p)
		return
	}
	if a.Win == nil {
		p.Loading = false
		return
	}
	dir := p.Path
	a.updateTabTitle(p)
	go func() {
		entries, err := fsx.ReadDir(dir)
		a.Win.Update(func() {
			if p.Gen != gen {
				return
			}
			p.Loading = false
			if err != nil {
				p.Err = err.Error()
				p.Entries = nil
				p.Rows = nil
				return
			}
			p.Entries = entries
			p.TrashIDs = nil
			p.TrashOriginal = nil
			a.applyPane(p)
			a.computeSizes(p)
			a.addRecent(dir, true)
			a.startWatcher(p, dir, gen)
			if a.compare.Open && (p == a.compare.Left || p == a.compare.Right) && a.compare.Left != nil && a.compare.Right != nil {
				a.computeCompare(a.compare.Left, a.compare.Right)
			}
		})
	}()
}

// loadVirtual fills a pane that shows a folder Pulse makes up.
func (a *App) loadVirtual(p *Pane) {
	if p.watchCancel != nil {
		p.watchCancel()
		p.watchCancel = nil
	}
	p.Loading = false
	p.Err = ""
	p.Entries = nil
	p.TrashIDs = map[string]string{}
	p.TrashOriginal = map[string]string{}
	switch p.Virtual {
	case virtualComputer:
		p.Drives = platform.Drives()
		for _, d := range p.Drives {
			p.Entries = append(p.Entries, fsx.Entry{
				Name:  a.driveDisplay(d),
				Path:  d.Path,
				IsDir: true,
				Kind:  fsx.KindDisk,
			})
		}
	case virtualTrash:
		items, err := a.trash.List()
		if err != nil {
			p.Err = err.Error()
		}
		for _, item := range items {
			e := fsx.Entry{
				Name:    item.Name,
				Path:    item.Stored,
				IsDir:   item.IsDir,
				Size:    item.Size,
				ModTime: item.DeletedAt,
			}
			if e.IsDir {
				e.Kind = fsx.KindFolder
			} else {
				e.Ext = strings.ToLower(strings.TrimPrefix(filepath.Ext(item.Name), "."))
				e.Kind = fsx.KindOf(e.Ext)
			}
			p.Entries = append(p.Entries, e)
			p.TrashIDs[item.Stored] = item.ID
			p.TrashOriginal[item.Stored] = item.Original
		}
	case virtualTag:
		for path, id := range a.cfg.TagAssignments {
			if id != p.TagID {
				continue
			}
			if e, err := fsx.Stat(path); err == nil {
				p.Entries = append(p.Entries, e)
			}
		}
	case virtualStarred:
		for _, f := range a.cfg.Favorites {
			if e, err := fsx.Stat(f.Path); err == nil {
				p.Entries = append(p.Entries, e)
			}
		}
	case virtualRecent:
		for _, r := range a.recent {
			if e, err := fsx.Stat(r.Path); err == nil {
				p.Entries = append(p.Entries, e)
			}
		}
	case virtualNetwork:
		// Nothing to enumerate without a platform browser; the view says so.
	case virtualSearch:
		// Entries were filled by the search that opened this pane.
	}
	a.applyPane(p)
}

// startWatcher polls a folder and refreshes the pane when its contents
// change, as Pulse tracks folder changes.
func (a *App) startWatcher(p *Pane, dir string, gen int) {
	if a.Win == nil || !a.cfg.View.AutoRefresh || dir == "" || !a.paneInActiveTab(p) {
		return
	}
	if p.watchCancel != nil {
		p.watchCancel()
	}
	initial := dirFingerprint(p.Entries)
	ctx, cancel := context.WithCancel(context.Background())
	p.watchCancel = cancel
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		last := initial
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			entries, err := fsx.ReadDir(dir)
			if err != nil {
				continue
			}
			fp := dirFingerprint(entries)
			if fp == last {
				continue
			}
			last = fp
			a.Win.Update(func() {
				if ctx.Err() != nil || p.Gen != gen || p.Virtual != virtualNone {
					return
				}
				a.refreshPane(p)
			})
			return
		}
	}()
}

// paneInActiveTab reports whether a pane belongs to the tab shown.
func (a *App) paneInActiveTab(p *Pane) bool {
	t := a.activeTab()
	if t == nil {
		return false
	}
	for _, pane := range t.Panes {
		if pane == p {
			return true
		}
	}
	return false
}

// refreshWatchers watches the active tab's folders and stops the others.
func (a *App) refreshWatchers() {
	for _, t := range a.tabs {
		for _, p := range t.Panes {
			if p.watchCancel != nil {
				p.watchCancel()
				p.watchCancel = nil
			}
		}
	}
	if a.Win == nil || !a.cfg.View.AutoRefresh {
		return
	}
	t := a.activeTab()
	if t == nil {
		return
	}
	for _, p := range t.Panes {
		if p.Virtual == virtualNone && p.Path != "" && !p.Loading {
			a.startWatcher(p, p.Path, p.Gen)
		}
	}
}

// dirFingerprint hashes names, sizes and times of a directory listing.
func dirFingerprint(entries []fsx.Entry) uint64 {
	h := fnv.New64a()
	var buf [8]byte
	for _, e := range entries {
		h.Write([]byte(e.Name))
		binary.LittleEndian.PutUint64(buf[:], uint64(e.Size))
		h.Write(buf[:])
		binary.LittleEndian.PutUint64(buf[:], uint64(e.ModTime.UnixNano()))
		h.Write(buf[:])
		if e.IsDir {
			h.Write([]byte{1})
		}
	}
	return h.Sum64()
}

// applyPane sorts and groups the pane's entries into rows.
func (a *App) applyPane(p *Pane) {
	p.Rows = a.sortedRows(p)
	if p.Selected >= len(p.Rows) {
		p.Selected = -1
	}
	if p.Selected < 0 {
		for _, e := range p.Entries {
			if e.Path == p.ScrollToPath {
				for i, r := range p.Rows {
					if !r.IsHeader && r.E.Path == e.Path {
						p.Selected = i
						p.List.ScrollIntoView(i)
						break
					}
				}
				break
			}
		}
	}
	if p.ScrollToPath != "" {
		for i, r := range p.Rows {
			if !r.IsHeader && r.E.Path == p.ScrollToPath {
				p.Selected = i
				p.List.ScrollIntoView(i)
				break
			}
		}
		p.ScrollToPath = ""
	}
	// Drop chosen paths that are no longer here.
	alive := map[string]bool{}
	for _, e := range p.Entries {
		alive[e.Path] = true
	}
	var stale []string
	for path := range p.Selection.All() {
		if !alive[path] {
			stale = append(stale, path)
		}
	}
	for _, path := range stale {
		p.Selection.Remove(path)
	}
	if len(p.keepSelection) > 0 {
		p.Selected = -1
		for _, path := range p.keepSelection {
			if alive[path] {
				p.Selection.Add(path)
			}
		}
		p.keepSelection = nil
	}
}

// computeSizes sums the folders shown in a pane, in the background, with a
// cache per folder.
func (a *App) computeSizes(p *Pane) {
	if a.Win == nil || !a.cfg.View.ComputeFolderSize || p.Virtual != virtualNone {
		return
	}
	p.SizeGen++
	gen := p.SizeGen
	var dirs []fsx.Entry
	for _, e := range p.Entries {
		if e.IsDir && !e.Link && len(dirs) < 300 {
			dirs = append(dirs, e)
		}
	}
	if len(dirs) == 0 {
		return
	}
	p.SizeBusy = true
	cached := map[string]sizeCache{}
	for _, d := range dirs {
		if c, ok := a.sizes[store.FolderKey(d.Path)]; ok && time.Since(c.At) < 10*time.Minute {
			cached[d.Path] = c
		}
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		for _, d := range dirs {
			if ctx.Err() != nil {
				break
			}
			key := store.FolderKey(d.Path)
			if c, ok := cached[d.Path]; ok {
				a.Win.Update(func() {
					if p.SizeGen != gen {
						return
					}
					if p.Sizes == nil {
						p.Sizes = map[string]int64{}
					}
					p.Sizes[d.Path] = c.Size
				})
				continue
			}
			size, files, err := fsx.DirSize(ctx, d.Path, 0)
			if err != nil || ctx.Err() != nil {
				continue
			}
			a.Win.Update(func() {
				if p.SizeGen != gen {
					return
				}
				if p.Sizes == nil {
					p.Sizes = map[string]int64{}
				}
				p.Sizes[d.Path] = size
				a.sizes[key] = sizeCache{Size: size, Files: files, At: time.Now()}
			})
		}
		a.Win.Update(func() {
			if p.SizeGen == gen {
				p.SizeBusy = false
			}
		})
	}()
}

// navigate moves a pane to a folder, recording history.
func (a *App) navigate(p *Pane, path string) {
	if p == nil {
		return
	}
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		path = filepath.Join(p.Path, path)
	}
	if p.Virtual == virtualNone && path == p.Path {
		a.refreshPane(p)
		return
	}
	p.Virtual = virtualNone
	p.TagID = ""
	p.SearchRoot = ""
	p.Path = path
	if p.HistAt >= 0 && p.HistAt < len(p.History) && p.History[p.HistAt] == path {
		// Keep the history as it is.
	} else {
		cut := p.HistAt + 1
		if cut < 0 {
			cut = 0
		}
		if cut > len(p.History) {
			cut = len(p.History)
		}
		p.History = append(p.History[:cut], path)
		p.HistAt = len(p.History) - 1
	}
	p.Selected = -1
	p.Selection.Clear()
	p.keepSelection = nil
	a.updateTabTitle(p)
	a.saveFolderPref(p)
	a.loadPane(p)
	a.saveSessionSoon()
}

// openVirtual shows a folder Pulse makes up in a pane.
func (a *App) openVirtual(p *Pane, kind virtualKind, tagID string) {
	if p == nil {
		return
	}
	p.Virtual = kind
	p.TagID = tagID
	p.SearchRoot = ""
	switch kind {
	case virtualComputer:
		p.Path = ""
	case virtualTrash:
		p.Path = ""
	case virtualTag:
		p.Path = ""
	default:
		p.Path = ""
	}
	p.History = nil
	p.HistAt = 0
	p.Selected = -1
	p.Selection.Clear()
	a.updateTabTitle(p)
	a.loadVirtual(p)
	a.saveSessionSoon()
}

// goBack moves a pane back through its history.
func (a *App) goBack(p *Pane) {
	if p == nil || p.Virtual != virtualNone || p.HistAt <= 0 || len(p.History) == 0 {
		return
	}
	p.HistAt--
	p.Path = p.History[p.HistAt]
	p.Selected = -1
	p.Selection.Clear()
	p.keepSelection = nil
	a.updateTabTitle(p)
	a.loadPane(p)
	a.saveSessionSoon()
}

// goForward moves a pane forward through its history.
func (a *App) goForward(p *Pane) {
	if p == nil || p.Virtual != virtualNone || p.HistAt >= len(p.History)-1 {
		return
	}
	p.HistAt++
	p.Path = p.History[p.HistAt]
	p.Selected = -1
	p.Selection.Clear()
	p.keepSelection = nil
	a.updateTabTitle(p)
	a.loadPane(p)
	a.saveSessionSoon()
}

// goUp moves a pane to its parent folder, or to This PC from a drive root.
func (a *App) goUp(p *Pane) {
	if p == nil {
		return
	}
	if p.Virtual != virtualNone {
		a.openVirtual(p, virtualComputer, "")
		return
	}
	parent := filepath.Dir(filepath.Clean(p.Path))
	if parent == p.Path {
		a.openVirtual(p, virtualComputer, "")
		return
	}
	a.navigate(p, parent)
}

// refreshPane reloads a pane, keeping the selection where possible.
func (a *App) refreshPane(p *Pane) {
	if p == nil {
		return
	}
	if p.Virtual != virtualNone {
		a.loadVirtual(p)
		return
	}
	sel := ""
	if e := a.selectedEntry(p); e != nil {
		sel = e.Path
	}
	var kept []string
	for path := range p.Selection.All() {
		kept = append(kept, path)
	}
	p.keepSelection = kept
	p.ScrollToPath = sel
	a.invalidatePaneCaches(p)
	a.loadPane(p)
}

// invalidatePaneCaches drops cached folder sizes and thumbnails of a
// pane's entries before it reloads.
func (a *App) invalidatePaneCaches(p *Pane) {
	for _, e := range p.Entries {
		if e.IsDir {
			delete(a.sizes, store.FolderKey(e.Path))
			continue
		}
		delete(a.thumb.m, e.Path)
		delete(a.thumb.failed, e.Path)
		delete(a.thumb.loading, e.Path)
		delete(a.imgDims, e.Path)
	}
}

// selectedEntry returns the entry chosen in a pane, or nil.
func (a *App) selectedEntry(p *Pane) *fsx.Entry {
	if p == nil || p.Selected < 0 || p.Selected >= len(p.Rows) {
		return nil
	}
	r := p.Rows[p.Selected]
	if r.IsHeader {
		return nil
	}
	return &r.E
}

// selectedEntries returns every chosen entry of a pane.
func (a *App) selectedEntries(p *Pane) []fsx.Entry {
	if p == nil {
		return nil
	}
	var out []fsx.Entry
	for _, e := range p.Entries {
		if p.Selection.Has(e.Path) {
			out = append(out, e)
		}
	}
	if len(out) == 0 {
		if e := a.selectedEntry(p); e != nil {
			out = append(out, *e)
		}
	}
	return out
}

// activateRow opens a row or folder, or restores a recycle bin item.
func (a *App) activateRow(p *Pane, i int) {
	if p == nil || i < 0 || i >= len(p.Rows) {
		return
	}
	r := p.Rows[i]
	if r.IsHeader {
		return
	}
	if r.TrashID != "" {
		a.restoreTrashItem(r.TrashID, p)
		return
	}
	if r.E.IsDir {
		a.navigate(p, r.E.Path)
		return
	}
	if err := platform.Open(r.E.Path); err != nil {
		a.toast = a.t("openFailed")
	}
}

// openSelected opens the chosen entries of the active pane.
func (a *App) openSelected() {
	p := a.activePane()
	if p == nil {
		return
	}
	entries := a.selectedEntries(p)
	if len(entries) == 0 {
		return
	}
	if len(entries) == 1 {
		a.activateEntry(p, entries[0])
		return
	}
	for _, e := range entries {
		if err := platform.Open(e.Path); err != nil {
			a.toast = a.t("openFailed")
			return
		}
	}
}

// activateEntry opens one entry from a pane.
func (a *App) activateEntry(p *Pane, e fsx.Entry) {
	if e.IsDir {
		a.navigate(p, e.Path)
		return
	}
	if err := platform.Open(e.Path); err != nil {
		a.toast = a.t("openFailed")
	}
}

// selectPath chooses a path in a pane and shows it.
func (a *App) selectPath(p *Pane, path string) {
	if p == nil {
		return
	}
	for i, r := range p.Rows {
		if !r.IsHeader && r.E.Path == path {
			p.Selected = i
			p.List.ScrollIntoView(i)
			return
		}
	}
}

// revealEntry shows a path in a pane, navigating to its folder.
func (a *App) revealEntry(p *Pane, e fsx.Entry) {
	if p == nil {
		return
	}
	dir := e.Path
	if !e.IsDir {
		dir = filepath.Dir(e.Path)
	}
	if p.Virtual == virtualNone && filepath.Clean(p.Path) == filepath.Clean(dir) {
		a.selectPath(p, e.Path)
		return
	}
	p.ScrollToPath = e.Path
	a.navigate(p, dir)
}

// updateTabTitle makes the tab's title follow its active pane.
func (a *App) updateTabTitle(p *Pane) {
	for _, t := range a.tabs {
		for _, pane := range t.Panes {
			if pane == p {
				t.Title = a.tabTitle(p)
				return
			}
		}
	}
}

// syncTabTitle refreshes the active tab's title.
func (a *App) syncTabTitle() {
	t := a.activeTab()
	if t == nil {
		return
	}
	t.Title = a.tabTitle(t.Panes[t.Active])
}

// setPaneView changes a pane's view and remembers it.
func (a *App) setPaneView(p *Pane, v viewMode) {
	if p == nil || p.View == v {
		return
	}
	p.View = v
	a.saveFolderPref(p)
	a.applyPane(p)
}

// setPaneSort changes a pane's sort and remembers it.
func (a *App) setPaneSort(p *Pane, by string, desc bool) {
	if p == nil {
		return
	}
	p.SortBy, p.SortDesc = by, desc
	p.SortOrders = nil
	a.saveFolderPref(p)
	a.applyPane(p)
}

// setPaneGroup changes a pane's grouping.
func (a *App) setPaneGroup(p *Pane, group string) {
	if p == nil {
		return
	}
	p.Group = group
	a.saveFolderPref(p)
	a.applyPane(p)
}

// setPaneFilter changes a pane's kind filter.
func (a *App) setPaneFilter(p *Pane, f string) {
	if p == nil {
		return
	}
	p.KindFilter = f
	a.applyPane(p)
}

// folderPrefFromPane captures a pane's view, sort, group and columns.
func (a *App) folderPrefFromPane(p *Pane) store.FolderPref {
	pref := store.FolderPref{
		View:     viewKey(p.View),
		SortBy:   p.SortBy,
		SortDesc: p.SortDesc,
		Group:    p.Group,
	}
	if len(p.List.Columns.Order) > 0 {
		pref.ColumnOrder = append([]string(nil), p.List.Columns.Order...)
	}
	for _, so := range p.SortOrders {
		pref.SortOrders = append(pref.SortOrders, store.SortPref{Column: so.Column, Descending: so.Descending})
	}
	if len(p.List.Columns.Widths) > 0 {
		pref.ColumnWidths = map[string]float32{}
		for id, w := range p.List.Columns.Widths {
			pref.ColumnWidths[id] = w
		}
	}
	if len(p.HiddenCols) > 0 {
		for id := range p.HiddenCols {
			pref.HiddenColumns = append(pref.HiddenColumns, id)
		}
	}
	return pref
}

// saveFolderPref stores a pane's view, sort and group for its folder.
func (a *App) saveFolderPref(p *Pane) {
	if p == nil || p.Virtual != virtualNone {
		return
	}
	if a.cfg.FolderPrefs == nil {
		a.cfg.FolderPrefs = map[string]store.FolderPref{}
	}
	key := store.FolderKey(p.Path)
	a.cfg.FolderPrefs[key] = a.folderPrefFromPane(p)
	if len(a.cfg.FolderPrefs) > 2000 {
		for k := range a.cfg.FolderPrefs {
			delete(a.cfg.FolderPrefs, k)
			if len(a.cfg.FolderPrefs) <= 1500 {
				break
			}
		}
	}
	a.saveSessionSoon()
}

// smartDate writes a modification time as Pulse does.
func (a *App) smartDate(t time.Time) string {
	if !a.cfg.View.SmartDates {
		return a.fullDate(t)
	}
	now := a.now
	if now.IsZero() {
		now = time.Now()
	}
	y, m, d := t.Date()
	ny, nm, nd := now.Date()
	if y == ny && m == nm && d == nd {
		return a.t("today") + " " + t.Format("15:04")
	}
	if y == ny && m == nm && d == nd-1 {
		return a.t("yesterday") + " " + t.Format("15:04")
	}
	diff := now.Sub(t)
	if diff >= 0 && diff < 7*24*time.Hour && y == ny {
		return a.weekday(t) + " " + t.Format("15:04")
	}
	return a.fullDate(t)
}

func (a *App) fullDate(t time.Time) string {
	switch a.lang() {
	case langEN:
		return t.Format("Jan 2, 2006 15:04")
	default:
		return t.Format("2006-01-02 15:04")
	}
}

func (a *App) weekday(t time.Time) string {
	names := [][7]string{
		{"周日", "周一", "周二", "周三", "周四", "周五", "周六"},
		{"週日", "週一", "週二", "週三", "週四", "週五", "週六"},
		{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"},
	}
	return names[a.lang()][int(t.Weekday())]
}

// driveDisplay names a drive as the sidebar and This PC do.
func (a *App) driveDisplay(d platform.Drive) string {
	name := d.Name
	if name == "" {
		switch d.Kind {
		case "fixed":
			name = a.t("driveFixed")
		case "removable":
			name = a.t("driveRemovable")
		case "network":
			name = a.t("driveNetwork")
		case "optical":
			name = a.t("driveOptical")
		default:
			name = a.t("driveOther")
		}
	}
	if d.Letter != "" {
		return name + " (" + d.Letter + ")"
	}
	return name
}

// driveTypeName names a drive's kind.
func (a *App) driveTypeName(d platform.Drive) string {
	switch d.Kind {
	case "fixed":
		return a.t("driveFixed")
	case "removable":
		return a.t("driveRemovable")
	case "network":
		return a.t("driveNetwork")
	case "optical":
		return a.t("driveOptical")
	}
	return a.t("driveOther")
}

// findDrive returns the drive info of a This PC row.
func (a *App) findDrive(p *Pane, path string) (platform.Drive, bool) {
	for _, d := range p.Drives {
		if d.Path == path {
			return d, true
		}
	}
	return platform.Drive{}, false
}

// capacityBar draws a small used/free bar.
func (a *App) capacityBar(c *ui.Context, p Palette, used, total int64, width float32) {
	ui.Box(c).Width(width).Height(3).Radius(2).Background(p.Surface).Children(func() {
		frac := float32(0)
		if total > 0 {
			frac = float32(used) / float32(total)
		}
		if frac < 0 {
			frac = 0
		}
		if frac > 1 {
			frac = 1
		}
		ui.Box(c).Width(width * frac).Height(3).Radius(2).Background(p.Accent)
	})
}

// sizeSum returns the total size of the files directly in a pane.
func (a *App) sizeSum(p *Pane) int64 {
	var n int64
	for _, e := range p.Entries {
		if !e.IsDir {
			n += e.Size
		}
	}
	return n
}

// kindCounts counts folders and files in a pane.
func (a *App) kindCounts(p *Pane) (folders, files int) {
	for _, e := range p.Entries {
		if e.IsDir {
			folders++
		} else {
			files++
		}
	}
	return
}

// syncTableSort reorders a pane's rows by the table's sort.
func (a *App) syncTableSort(p *Pane) bool {
	if len(p.SortOrders) > 0 {
		first := p.SortOrders[0]
		if first.Column != p.SortBy || first.Descending != p.SortDesc {
			p.SortBy, p.SortDesc = first.Column, first.Descending
			a.saveFolderPref(p)
			a.applyPane(p)
			return true
		}
		return false
	}
	if p.sortOrder.Column == "" {
		return false
	}
	by := p.sortOrder.Column
	desc := p.sortOrder.Descending
	if by != p.SortBy || desc != p.SortDesc {
		p.SortBy, p.SortDesc = by, desc
		p.SortOrders = nil
		a.saveFolderPref(p)
		a.applyPane(p)
		return true
	}
	return false
}

// sortOrderFor fills a pane's table sort from its settings.
func (a *App) sortOrderFor(p *Pane) {
	p.sortOrder = ui.SortOrder{Column: p.SortBy, Descending: p.SortDesc}
}

// ensureValidPath falls back to the default location when a path is gone.
func (a *App) ensureValidPath(p *Pane) {
	if p.Virtual != virtualNone {
		return
	}
	if !platform.Exists(p.Path) {
		p.Path = a.defaultPath()
		p.History = []string{p.Path}
		p.HistAt = 0
	}
}

// openInNewTab opens a folder in a new tab.
func (a *App) openInNewTab(path string) {
	a.newTab(path)
}

// openSearchTab shows search results in a new tab.
func (a *App) openSearchTab(query, root string, results []fsx.Entry) {
	p := a.newPane(root)
	p.Virtual = virtualSearch
	p.SearchRoot = root
	p.Entries = results
	p.Query = query
	p.Path = root
	p.History = nil
	p.HistAt = 0
	t := &Tab{Layout: layoutSingle, Active: 0, Panes: []*Pane{p}, SplitA: 50, SplitB: 50, SplitV: 50}
	t.Title = a.t("tabSearch")
	a.tabs = append(a.tabs, t)
	a.tab = len(a.tabs) - 1
	a.applyPane(p)
	a.saveSessionSoon()
}

// pathExists reports whether a path can be opened.
func pathExists(path string) bool { return platform.Exists(path) }

// fileInfoOf returns metadata, or an error message.
func fileInfoOf(path string) (fsx.Entry, error) {
	e, err := fsx.Stat(path)
	if err != nil {
		return e, fmt.Errorf("%w", err)
	}
	return e, nil
}

// osOpen opens a path with its default app.
func osOpen(path string) error { return platform.Open(path) }

// osReveal reveals a path in the system file manager.
func osReveal(path string) error { return platform.Reveal(path) }

// sortRowsByDate re-sorts recycle bin items newest first.
func sortTrashRows(rows []row) {
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].E.ModTime.After(rows[j].E.ModTime) })
}

// ensureNoTrailingSlash cleans a path for display.
func cleanPath(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}

// homeTitle names the home folder.
func (a *App) homeTitle() string {
	if a.lang() == langEN {
		return "Home"
	}
	return "主目录"
}

// unusedOsImport keeps os imported for helpers above in all builds.
var _ = os.ErrNotExist

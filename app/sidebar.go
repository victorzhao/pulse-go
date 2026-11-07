package app

import (
	"path/filepath"
	"time"

	"pulse/internal/fsx"
	"pulse/internal/platform"
	"pulse/internal/store"

	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
)

// drivesCached returns the mounted volumes, refreshed every few seconds.
func (a *App) drivesCached() []platform.Drive {
	if a.drives == nil || time.Since(a.drivesAt) > 5*time.Second {
		a.drives = platform.Drives()
		a.drivesAt = time.Now()
	}
	return a.drives
}

// trashSummary returns the recycle bin's count and size, refreshed
// occasionally.
func (a *App) trashSummary() (int, int64) {
	if time.Since(a.trashAt) > 3*time.Second {
		a.trashCount, a.trashSize = a.trash.Size()
		a.trashAt = time.Now()
	}
	return a.trashCount, a.trashSize
}

// sidebar builds the source list and the staging tray.
func (a *App) sidebar(c *ui.Context, p Palette) {
	col := ui.Column(c).Fill().Clip().Background(p.Panel).BorderWidth(0, 1, 0, 0).BorderColor(p.BorderSoft)
	col.Children(func() {
		ui.Scroll(c).Grow(1).Padding(6).Gap(1).Children(func() {
			for _, id := range a.side.order {
				switch id {
				case "starred":
					a.sideStarred(c, p)
				case "quickAccess":
					a.sideQuick(c, p)
				case "recent":
					a.sideRecent(c, p)
				case "tags":
					a.sideTags(c, p)
				case "computer":
					a.sideComputer(c, p)
				case "network":
					a.sideNetwork(c, p)
				}
			}
		})
		a.stashCard(c, p)
	})
}

// sideStarred builds the starred section.
func (a *App) sideStarred(c *ui.Context, p Palette) {
	a.sideSectionHeader(c, p, "starred", "starred", &a.side.openStarred, true, func() {
		a.addCurrentToFavorites()
	})
	if !a.side.openStarred {
		return
	}
	if len(a.cfg.Favorites) == 0 {
		a.sideEmpty(c, p, a.t("starredEmpty"))
	}
	for _, f := range a.cfg.Favorites {
		f := f
		row := a.sideItem(c, p, "star", f.Name, a.side.starredSel == f.Path)
		row.ContextMenu(func(m *ui.Menu) {
			if m.Item(a.t("removeFavorite")).Chosen() {
				a.removeFavorite(f.Path)
			}
			if m.Item(a.t("openNewTab")).Chosen() {
				a.openFavorite(f)
			}
		})
		if row.Clicked() {
			a.side.starredSel = f.Path
			a.openFavorite(f)
		}
	}
}

// sideQuick builds the quick access section.
func (a *App) sideQuick(c *ui.Context, p Palette) {
	a.sideSectionHeader(c, p, "quickAccess", "quickAccess", &a.side.openQuick, true, func() {
		a.addCurrentToQuick()
	})
	if !a.side.openQuick {
		return
	}
	a.quickItem(c, p, "home", a.homeTitle(), a.homeDir(), "")
	for _, d := range platform.UserDirs() {
		a.quickItem(c, p, quickIcon(d.Name), a.t(d.Name), d.Path, "")
	}
	for _, f := range a.cfg.Quick {
		f := f
		row := a.sideItem(c, p, "folder", f.Name, a.side.quickSel == f.Path)
		row.ContextMenu(func(m *ui.Menu) {
			if m.Item(a.t("removeQuick")).Chosen() {
				a.removeQuick(f.Path)
			}
		})
		if row.Clicked() {
			a.side.quickSel = f.Path
			a.navigateActive(f.Path)
		}
	}
	count, size := a.trashSummary()
	label := a.t("recycleBin")
	row := ui.Row(c).Height(32).Padding(4, 8).Gap(8).Radius(6)
	if a.side.quickSel == "__trash" {
		row.Background(fade(p.Accent, 0.18))
	}
	row.Children(func() {
		ui.Image(c, quickIcons["recycle"]).Size(15, 15)
		ui.Column(c).Grow(1).Gap(0).Justify(ui.Center).Children(func() {
			ui.Text(c, label).FontSize(12).SingleLine().TextColor(p.Text)
			ui.Text(c, a.tf("itemsCount", count)+" · "+fsx.FormatSize(size)).FontSize(9.5).SingleLine().TextColor(p.TextFaint)
		})
	})
	if row.Clicked() {
		a.side.quickSel = "__trash"
		if pane := a.activePane(); pane != nil {
			a.openVirtual(pane, virtualTrash, "")
		}
	}
	row.ContextMenu(func(m *ui.Menu) {
		if m.Item(a.t("emptyTrash")).Chosen() {
			a.emptyTrash()
		}
	})
}

// sideRecent builds the recent section.
func (a *App) sideRecent(c *ui.Context, p Palette) {
	a.sideSectionHeader(c, p, "recent", "recent", &a.side.openRecent, false, nil)
	if !a.side.openRecent {
		return
	}
	if len(a.recent) == 0 {
		a.sideEmpty(c, p, a.t("noRecent"))
	}
	for i, r := range a.recent {
		if i >= 12 {
			break
		}
		r := r
		row := ui.Row(c).Height(29).Padding(4, 8).Gap(8).Radius(6)
		if a.side.recentSel == r.Path {
			row.Background(fade(p.Accent, 0.18))
		}
		row.Children(func() {
			ui.Image(c, quickIcons["clock"]).Size(15, 15)
			ui.Text(c, r.Name).FontSize(12).SingleLine().Grow(1).TextColor(p.Text)
		})
		row.ContextMenu(func(m *ui.Menu) {
			if m.Item(a.t("removeRecent")).Chosen() {
				a.removeRecent(r.Path)
			}
		})
		if row.Clicked() {
			a.side.recentSel = r.Path
			a.navigateActive(r.Path)
		}
	}
}

// sideTags builds the tags section.
func (a *App) sideTags(c *ui.Context, p Palette) {
	a.sideSectionHeader(c, p, "tags", "tags", &a.side.openTags, true, func() {
		a.tags.Open = true
		a.tags.Editing = -1
		a.tags.Name = ""
		a.tags.Color = defaultTagColor(a.cfg.Tags)
	})
	if !a.side.openTags {
		return
	}
	counts := a.tagCounts()
	for _, tag := range a.cfg.Tags {
		tag := tag
		row := ui.Row(c).Height(28).Padding(4, 8).Gap(8).Radius(6)
		if a.side.tagSel == tag.ID {
			row.Background(fade(p.Accent, 0.18))
		}
		row.Children(func() {
			ui.Box(c).Size(8, 8).Radius(4).Background(ui.Hex(tag.Color))
			ui.Text(c, tag.Name).FontSize(12).SingleLine().Grow(1).TextColor(p.Text)
			ui.Text(c, itoa(counts[tag.ID])).FontSize(10.5).TextColor(p.TextFaint)
		})
		row.OnClick(func() {
			a.side.tagSel = tag.ID
			if pane := a.activePane(); pane != nil {
				a.openVirtual(pane, virtualTag, tag.ID)
			}
		})
		row.ContextMenu(func(m *ui.Menu) {
			if m.Item(a.t("rename")).Chosen() {
				a.tags.Open = true
				a.tags.Editing = a.tagIndex(tag.ID)
				a.tags.Name = tag.Name
				a.tags.Color = tag.Color
			}
			if m.Item(a.t("delete")).Chosen() {
				a.deleteTag(tag.ID)
			}
		})
	}
}

// sideComputer builds the This PC section.
func (a *App) sideComputer(c *ui.Context, p Palette) {
	a.sideSectionHeader(c, p, "computer", "computer", &a.side.openComputer, false, nil)
	if !a.side.openComputer {
		return
	}
	for _, d := range a.drivesCached() {
		d := d
		name := a.driveDisplay(d)
		row := ui.Column(c).Padding(4, 8).Gap(2).Radius(6)
		if a.side.driveSel == d.Path {
			row.Background(fade(p.Accent, 0.18))
		}
		row.Children(func() {
			ui.Row(c).Gap(8).Children(func() {
				ui.Image(c, kindIcon(fsx.KindDisk)).Size(15, 15)
				ui.Text(c, name).FontSize(12).SingleLine().Grow(1).TextColor(p.Text)
				if d.Total > 0 {
					ui.Text(c, fsx.FormatSize(d.Free)).FontSize(10).TextColor(p.TextFaint)
				}
			})
			if a.cfg.View.ShowDriveBars && d.Total > 0 {
				a.capacityBar(c, p, d.Total-d.Free, d.Total, 200)
			}
		})
		row.OnClick(func() {
			a.side.driveSel = d.Path
			a.navigateActive(d.Path)
		})
		row.ContextMenu(func(m *ui.Menu) {
			if m.Item(a.t("openNewTab")).Chosen() {
				a.newTab(d.Path)
			}
			if m.Item(a.t("properties")).Chosen() {
				if e, err := fsx.Stat(d.Path); err == nil {
					a.showProperties([]fsx.Entry{e})
				}
			}
		})
	}
}

// sideNetwork builds the network section.
func (a *App) sideNetwork(c *ui.Context, p Palette) {
	a.sideSectionHeader(c, p, "network", "network", &a.side.openNetwork, false, nil)
	if !a.side.openNetwork {
		return
	}
	var nets []platform.Drive
	for _, d := range a.drivesCached() {
		if d.Kind == "network" {
			nets = append(nets, d)
		}
	}
	if len(nets) == 0 {
		a.sideEmpty(c, p, a.t("networkEmpty"))
	}
	for _, d := range nets {
		d := d
		row := a.sideItem(c, p, "network", a.driveDisplay(d), false)
		if row.Clicked() {
			a.navigateActive(d.Path)
		}
		row.ContextMenu(func(m *ui.Menu) {
			if m.Item(a.t("openNewTab")).Chosen() {
				a.newTab(d.Path)
			}
		})
	}
}

// quickIcon returns the icon of a well-known folder.
func quickIcon(name string) string {
	switch name {
	case "desktop":
		return "desktop"
	case "documents":
		return "file"
	case "downloads":
		return "download"
	case "pictures":
		return "picture"
	case "music":
		return "music"
	case "videos":
		return "video"
	}
	return "folder"
}

// sideSectionHeader draws a collapsible section title, with an optional
// plus button.
func (a *App) sideSectionHeader(c *ui.Context, p Palette, id, titleKey string, open *bool, plus bool, add func()) {
	row := ui.Row(c).Height(28).Padding(2, 6).Gap(5).Radius(5)
	if row.Hovered() {
		row.Background(p.SurfaceHi)
	}
	row.Children(func() {
		ui.Text(c, a.t(titleKey)).FontSize(11).Bold().TextColor(p.TextMuted).SingleLine()
		ui.Spacer(c)
		if plus && add != nil {
			if flatButton(c, p, 18, 18).Children(func() {
				ui.Icon(c, icon("plus")).FontSize(12).TextColor(p.TextMuted)
			}).Clicked() {
				add()
			}
		}
		arrow := "chevron-down"
		if !*open {
			arrow = "chevron-right"
		}
		if flatButton(c, p, 18, 18).Children(func() {
			ui.Icon(c, icon(arrow)).FontSize(12).TextColor(p.TextFaint)
		}).Clicked() {
			*open = !*open
		}
	})
	row.OnClick(func() {
		*open = !*open
	})
	row.Drag(id)
	if from, ok := ui.Drop[string](row); ok && from != id {
		a.moveSideSection(from, id)
	}
}

// moveSideSection moves one sidebar section before another.
func (a *App) moveSideSection(from, before string) {
	fromAt, beforeAt := -1, -1
	for i, id := range a.side.order {
		if id == from {
			fromAt = i
		}
		if id == before {
			beforeAt = i
		}
	}
	if fromAt < 0 || beforeAt < 0 || fromAt == beforeAt {
		return
	}
	id := a.side.order[fromAt]
	order := append(a.side.order[:fromAt], a.side.order[fromAt+1:]...)
	beforeAt = -1
	for i, v := range order {
		if v == before {
			beforeAt = i
			break
		}
	}
	if beforeAt < 0 {
		beforeAt = len(order)
	}
	a.side.order = append(order[:beforeAt], append([]string{id}, order[beforeAt:]...)...)
	a.saveSessionSoon()
}

// sideItem draws one sidebar row with an icon and a label.
func (a *App) sideItem(c *ui.Context, p Palette, iconName, label string, selected bool) ui.Element {
	row := ui.Row(c).Height(29).Padding(4, 8).Gap(8).Radius(6)
	if selected {
		row.Background(fade(p.Accent, 0.18))
	}
	row.Children(func() {
		ui.Icon(c, icon(iconName)).FontSize(15).TextColor(p.TextMuted)
		ui.Text(c, label).FontSize(12).SingleLine().Grow(1).TextColor(p.Text)
	})
	return row
}

// sideItemWithMeta adds a muted second line to a sidebar row.
func (a *App) sideItemWithMeta(c *ui.Context, p Palette, iconName, label, meta string, selected bool) ui.Element {
	row := ui.Row(c).Height(32).Padding(4, 8).Gap(8).Radius(6)
	if selected {
		row.Background(fade(p.Accent, 0.18))
	}
	row.Children(func() {
		ui.Icon(c, icon(iconName)).FontSize(15).TextColor(p.TextMuted)
		ui.Column(c).Grow(1).Gap(0).Justify(ui.Center).Children(func() {
			ui.Text(c, label).FontSize(12).SingleLine().TextColor(p.Text)
			ui.Text(c, meta).FontSize(9.5).SingleLine().TextColor(p.TextFaint)
		})
	})
	return row
}

// quickItem draws a quick access folder.
func (a *App) quickItem(c *ui.Context, p Palette, iconName, label, path, meta string) {
	row := ui.Row(c).Height(29).Padding(4, 8).Gap(8).Radius(6)
	if a.side.quickSel == path {
		row.Background(fade(p.Accent, 0.18))
	}
	row.Children(func() {
		if svg, ok := quickIcons[iconName]; ok {
			ui.Image(c, svg).Size(15, 15)
		} else {
			ui.Image(c, kindIcon(fsx.KindFolder)).Size(15, 15)
		}
		ui.Text(c, label).FontSize(12).SingleLine().Grow(1).TextColor(p.Text)
		if meta != "" {
			ui.Text(c, meta).FontSize(9.5).TextColor(p.TextFaint)
		}
	})
	if row.Clicked() {
		a.side.quickSel = path
		a.navigateActive(path)
	}
}

// sideEmpty draws a muted line under an empty section.
func (a *App) sideEmpty(c *ui.Context, p Palette, text string) {
	ui.Text(c, text).FontSize(10.5).TextColor(p.TextFaint).Padding(2, 12)
}

// navigateActive moves the active pane to a path, revealing files.
func (a *App) navigateActive(path string) {
	pane := a.activePane()
	if pane == nil {
		return
	}
	if info, err := fsx.Stat(path); err == nil && !info.IsDir {
		a.revealEntry(pane, info)
		return
	}
	a.navigate(pane, path)
}

// openFavorite opens a starred folder or reveals a starred file.
func (a *App) openFavorite(f store.Favorite) {
	pane := a.activePane()
	if pane == nil {
		return
	}
	if info, err := fsx.Stat(f.Path); err == nil {
		if info.IsDir {
			a.navigate(pane, f.Path)
		} else {
			a.revealEntry(pane, info)
		}
		return
	}
	a.toast = a.t("notFound")
}

// addCurrentToFavorites stars the active pane's folder.
func (a *App) addCurrentToFavorites() {
	pane := a.activePane()
	if pane == nil || pane.Virtual != virtualNone {
		return
	}
	if a.isFavorite(pane.Path) {
		return
	}
	name := filepath.Base(pane.Path)
	if name == "." || name == "" {
		name = pane.Path
	}
	a.cfg.Favorites = append(a.cfg.Favorites, store.Favorite{Name: name, Path: pane.Path})
	a.saveSessionSoon()
}

// addCurrentToQuick adds the active folder to quick access.
func (a *App) addCurrentToQuick() {
	pane := a.activePane()
	if pane == nil || pane.Virtual != virtualNone {
		return
	}
	for _, q := range a.cfg.Quick {
		if q.Path == pane.Path {
			return
		}
	}
	name := filepath.Base(pane.Path)
	if name == "." || name == "" {
		name = pane.Path
	}
	a.cfg.Quick = append(a.cfg.Quick, store.Favorite{Name: name, Path: pane.Path})
	a.saveSessionSoon()
}

// removeFavorite unstars a path.
func (a *App) removeFavorite(path string) {
	for i, f := range a.cfg.Favorites {
		if f.Path == path {
			a.cfg.Favorites = append(a.cfg.Favorites[:i], a.cfg.Favorites[i+1:]...)
			a.saveSessionSoon()
			return
		}
	}
}

// removeQuick drops a quick access shortcut.
func (a *App) removeQuick(path string) {
	for i, f := range a.cfg.Quick {
		if f.Path == path {
			a.cfg.Quick = append(a.cfg.Quick[:i], a.cfg.Quick[i+1:]...)
			a.saveSessionSoon()
			return
		}
	}
}

// tagCounts counts the files assigned to each tag.
func (a *App) tagCounts() map[string]int {
	counts := map[string]int{}
	for _, id := range a.cfg.TagAssignments {
		counts[id]++
	}
	return counts
}

// stashCard draws the staging tray at the bottom of the sidebar.
func (a *App) stashCard(c *ui.Context, p Palette) {
	card := ui.Column(c).Padding(8).Gap(6).Radius(8).Background(p.Surface).Border(1, p.BorderSoft)
	_, overDrag := ui.DragOver[dragFiles](card)
	if overDrag {
		card.Border(1, p.Accent).Background(fade(p.Accent, 0.12))
	}
	card.Children(func() {
		ui.Row(c).Gap(6).Children(func() {
			ui.Icon(c, icon("download")).FontSize(14).TextColor(p.Accent)
			ui.Text(c, a.t("stash")).FontSize(11).Bold().TextColor(p.Text).SingleLine()
			ui.Spacer(c)
			if len(a.stash.Items) > 0 {
				if ui.Button(c, a.t("copyHere")).FontSize(10).Height(22).Clicked() {
					a.pasteStash()
				}
				if ui.Button(c, "").Width(22).Height(22).Radius(4).Children(func() {
					ui.Icon(c, icon("clear")).FontSize(12)
				}).Tooltip(a.t("clearStash")).Clicked() {
					a.stash.Items = nil
					a.toast = a.t("stashCleared")
				}
			}
		})
		if len(a.stash.Items) == 0 {
			ui.Box(c).FillWidth().Padding(8).Radius(6).Border(1, p.BorderSoft).BorderStyle(ui.BorderDashed).Children(func() {
				ui.Column(c).Gap(2).AlignItems(ui.Center).Children(func() {
					ui.Text(c, a.t("stashEmpty")).FontSize(10.5).TextColor(p.TextMuted)
					ui.Text(c, a.t("stashHint")).FontSize(9.5).TextColor(p.TextFaint).MaxLines(2)
				})
			})
		} else {
			for i, item := range a.stash.Items {
				if i >= 2 {
					break
				}
				item := item
				row := ui.Row(c).Gap(6).Padding(3, 2).Radius(4)
				row.Children(func() {
					ui.Image(c, kindIcon(kindOfPath(item.Path, item.IsDir))).Size(14, 14)
					ui.Column(c).Grow(1).Gap(0).Children(func() {
						ui.Text(c, item.Name).FontSize(10).SingleLine().TextColor(p.Text)
						ui.Text(c, fsx.FormatSize(item.Size)+" · "+a.stashOp(item)).FontSize(9).TextColor(p.TextFaint).SingleLine()
					})
					if ui.Button(c, "").Width(18).Height(18).Radius(3).Children(func() {
						ui.Icon(c, icon("x")).FontSize(10)
					}).Clicked() {
						a.stash.Items = append(a.stash.Items[:i], a.stash.Items[i+1:]...)
					}
				})
				row.Drag(dragFiles{Paths: []string{item.Path}, Move: item.Move, FromStash: true})
				row.DragDataFrom(func() transfer.Data {
					data, err := transfer.FileData(item.Path)
					if err != nil {
						return transfer.Data{}
					}
					return data
				})
			}
			if len(a.stash.Items) > 2 {
				ui.Text(c, a.tf("stashCount", len(a.stash.Items))).FontSize(9.5).TextColor(p.TextMuted)
			}
			ui.Row(c).Gap(6).Children(func() {
				if len(a.stash.Items) == 2 {
					if ui.Button(c, a.t("compare")).FontSize(10).Height(22).Clicked() {
						a.compareStash()
					}
				}
				ui.Spacer(c)
				ui.Text(c, a.tf("stashSize", fsx.FormatSize(a.stashTotalSize()))).FontSize(9.5).TextColor(p.TextFaint)
			})
		}
	})
	if df, ok := ui.Drop[dragFiles](card); ok {
		a.addStash(df.Paths, df.Move)
	}
	if data, ok := ui.DropData(card, transfer.DropOptions{Formats: []transfer.Format{transfer.FileList}, Operations: transfer.Copy | transfer.Move}); ok {
		if paths, err := data.Data.Files(); err == nil {
			a.addStash(paths, data.Operation == transfer.Move)
		}
	}
}

// stashOp names how a staged item will be pasted.
func (a *App) stashOp(item stashItem) string {
	if item.Move {
		return a.t("stashMove")
	}
	return a.t("stashCopy")
}

// stashTotalSize sums the staged items.
func (a *App) stashTotalSize() int64 {
	var n int64
	for _, item := range a.stash.Items {
		n += item.Size
	}
	return n
}

// kindOfPath guesses a kind for an item that is not on disk.
func kindOfPath(path string, isDir bool) fsx.Kind {
	if isDir {
		return fsx.KindFolder
	}
	ext := stringsTrimExt(path)
	return fsx.KindOf(ext)
}

func stringsTrimExt(path string) string {
	name := filepath.Base(path)
	if i := lastDot(name); i >= 0 {
		return lowerASCII(name[i+1:])
	}
	return ""
}

func lastDot(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '.' {
			return i
		}
	}
	return -1
}

func lowerASCII(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}

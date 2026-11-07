package app

import (
	"path/filepath"
	"strings"

	"pulse/internal/fsx"
	"pulse/internal/platform"
	"pulse/internal/store"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
)

// dragFiles is the value dragged within the window: files from a list, or
// items of the staging tray.
type dragFiles struct {
	Paths     []string
	Move      bool
	FromStash bool
}

// panes lays out the two sides, each with its own tabs.
func (a *App) panes(c *ui.Context, p Palette) ui.Element {
	a.ensureSideDefaults()
	return ui.SplitPercent(c, &a.splitPercent, func() {
		a.sidePaneView(c, p, 0)
	}, func() {
		a.sidePaneView(c, p, 1)
	})
}

// sidePaneView builds one side: its tab strip, header and file view.
func (a *App) sidePaneView(c *ui.Context, p Palette, idx int) ui.Element {
	side := a.sideAt(idx)
	pane := a.sidePane(idx)
	active := a.activeSide == idx
	col := ui.Column(c).Fill().Clip().Background(p.Panel)
	if active {
		col.BorderWidth(1, 1, 0, 1).BorderColor(p.Border)
	} else {
		col.BorderWidth(1, 0, 0, 0).BorderColor(p.BorderSoft)
	}
	col.Children(func() {
		a.sideTabStrip(c, p, idx)
		if pane == nil {
			return
		}
		pane.viewTab, pane.viewIdx = side, side.Active
		a.paneHeader(c, p, pane, active)
		a.paneBody(c, p, pane)
	})
	return col
}

// paneBody builds a pane's file view, drops and background menu.
func (a *App) paneBody(c *ui.Context, p Palette, pane *Pane) {
	body := ui.Column(c).Fill()
	body.ContextMenu(func(m *ui.Menu) {
		if pane.Virtual == virtualNone {
			if m.Item(a.t("newFolder")).Shortcut(ui.Cmd|ui.Shift, ui.KeyN).Chosen() {
				a.beginNewFolder(pane)
			}
			if m.Item(a.t("paste")).Shortcut(ui.Cmd, ui.KeyV).Disabled(len(a.stash.Items) == 0).Chosen() {
				a.pasteStash()
			}
			m.Separator()
		}
		if m.Item(a.t("selectAll")).Shortcut(ui.Cmd, ui.KeyA).Chosen() {
			a.selectAllRows(pane)
		}
		if m.Item(a.t("invertSelection")).Chosen() {
			a.invertSelection(pane)
		}
		if m.Item(a.t("refresh")).Shortcut(0, ui.KeyF5).Chosen() {
			a.refreshPane(pane)
		}
		m.Separator()
		m.Submenu(a.t("view"), func(m *ui.Menu) {
			if m.Item(a.t("viewDetails")).Radio(pane.View == viewDetails).Chosen() {
				a.setPaneView(pane, viewDetails)
			}
			if m.Item(a.t("viewTiles")).Radio(pane.View == viewTiles).Chosen() {
				a.setPaneView(pane, viewTiles)
			}
			if m.Item(a.t("viewIcons")).Radio(pane.View == viewIcons).Chosen() {
				a.setPaneView(pane, viewIcons)
			}
			if m.Item(a.t("viewList")).Radio(pane.View == viewList).Chosen() {
				a.setPaneView(pane, viewList)
			}
			if m.Item(a.t("viewContent")).Radio(pane.View == viewContent).Chosen() {
				a.setPaneView(pane, viewContent)
			}
		})
		if pane.Virtual == virtualTrash && len(pane.Entries) > 0 {
			m.Separator()
			if m.Item(a.t("emptyTrash")).Chosen() {
				a.emptyTrash()
			}
		}
	})
	// Files dropped from the system land here.
	opts := transfer.DropOptions{Formats: []transfer.Format{transfer.FileList}, Operations: transfer.Copy | transfer.Move}
	_, overData := ui.DataDragOver(body, opts)
	_, overDrag := ui.DragOver[dragFiles](body)
	if overData || overDrag {
		if overDrag {
			pane.DropHint = 1
		} else {
			pane.DropHint = 2
		}
	} else if pane.DropHint != 0 && !overData && !overDrag {
		pane.DropHint = 0
	}
	if drop, ok := ui.DropData(body, opts); ok {
		if paths, err := drop.Data.Files(); err == nil && len(paths) > 0 {
			move := drop.Operation == transfer.Move
			a.dropFilesInto(pane, paths, move)
		}
		pane.DropHint = 0
	}
	if df, ok := ui.Drop[dragFiles](body); ok {
		a.dropFilesInto(pane, df.Paths, df.Move)
		if df.FromStash {
			a.stash.Items = nil
		}
		pane.DropHint = 0
	}
	body.Children(func() {
		switch pane.View {
		case viewDetails:
			a.detailsView(c, p, pane)
		case viewTiles:
			a.tilesView(c, p, pane, true)
		case viewIcons:
			a.tilesView(c, p, pane, false)
		case viewList:
			a.listView(c, p, pane)
		case viewContent:
			a.contentView(c, p, pane)
		}
		if pane.DropHint != 0 {
			ui.Box(c).Fill().Background(ui.RGBA(0, 0, 0, 0.35)).Children(func() {
				ui.Box(c).Attach(ui.AnchorCenter, ui.AnchorCenter).Padding(18, 26).Radius(12).
					Background(p.PanelSolid).Border(1, p.Accent).Children(func() {
					ui.Column(c).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, icon("download")).FontSize(28).TextColor(p.Accent)
						msg := a.t("dropCopyHere")
						if pane.DropHint == 2 {
							msg = a.t("dropMoveHere")
						}
						ui.Text(c, msg).FontSize(13).TextColor(p.Text)
						ui.Text(c, pane.Path).FontSize(11).TextColor(p.TextMuted).SingleLine()
					})
				})
			})
		}
	})
}

// paneHeader draws a pane's name and view controls.
func (a *App) paneHeader(c *ui.Context, p Palette, pane *Pane, active bool) {
	title := a.tabTitle(pane)
	row := ui.Row(c).Height(34).Padding(0, 8).Gap(6)
	row.Background(p.Panel)
	row.BorderWidth(0, 0, 1, 0).BorderColor(p.BorderSoft)
	row.Children(func() {
		if flatButton(c, p, 24, 24).Children(func() {
			ui.Icon(c, icon("folder")).FontSize(15).TextColor(p.Accent)
		}).Clicked() {
			a.activatePane(pane)
		}
		name := ui.Text(c, title).FontSize(13).Bold().SingleLine()
		if active {
			name.TextColor(p.Text)
		} else {
			name.TextColor(p.TextMuted)
		}
		ui.Spacer(c)
		if pane.Loading {
			ui.Spinner(c).Size(14, 14)
		}
		a.viewButton(c, p, pane, viewList, "list", "viewList")
		a.viewButton(c, p, pane, viewTiles, "tiles", "viewTiles")
		a.viewButton(c, p, pane, viewIcons, "icons", "viewIcons")
		a.viewButton(c, p, pane, viewDetails, "details", "viewDetails")
		a.viewButton(c, p, pane, viewContent, "content", "viewContent")
		flatButton(c, p, 24, 24).Children(func() {
			ui.Icon(c, icon("more")).FontSize(15).TextColor(p.TextMuted)
		}).Tooltip(a.t("more")).Label(a.t("more")).PopoverMenu(func(m *ui.Menu) {
			if m.Item(a.t("viewDetails")).Radio(pane.View == viewDetails).Chosen() {
				a.setPaneView(pane, viewDetails)
			}
			if m.Item(a.t("viewTiles")).Radio(pane.View == viewTiles).Chosen() {
				a.setPaneView(pane, viewTiles)
			}
			if m.Item(a.t("viewIcons")).Radio(pane.View == viewIcons).Chosen() {
				a.setPaneView(pane, viewIcons)
			}
			if m.Item(a.t("viewList")).Radio(pane.View == viewList).Chosen() {
				a.setPaneView(pane, viewList)
			}
			if m.Item(a.t("viewContent")).Radio(pane.View == viewContent).Chosen() {
				a.setPaneView(pane, viewContent)
			}
			m.Separator()
			if m.Item(a.t("refresh")).Shortcut(0, ui.KeyF5).Chosen() {
				a.refreshPane(pane)
			}
			if m.Item(a.t("openFolderTab")).Chosen() {
				a.openInNewTab(pane.Path)
			}
		})
	})
	// Clicking the header makes its pane the active one.
	row.OnClick(func() {
		a.activatePane(pane)
	})
}

// viewButton draws one of the pane header's view switches.
func (a *App) viewButton(c *ui.Context, p Palette, pane *Pane, v viewMode, iconName, labelKey string) {
	btn := flatButton(c, p, 24, 24).Children(func() {
		ui.Icon(c, icon(iconName)).FontSize(14).TextColor(p.TextMuted)
	}).Tooltip(a.t(labelKey)).Label(a.t(labelKey))
	if pane.View == v {
		btn.Background(fade(p.Accent, 0.18)).TextColor(p.Accent)
	}
	if btn.Clicked() {
		a.setPaneView(pane, v)
	}
}

// dropFilesInto copies or moves dropped paths into a pane's folder.
func (a *App) dropFilesInto(pane *Pane, paths []string, move bool) {
	if pane.Virtual != virtualNone {
		a.toast = a.t("cannotOpen")
		return
	}
	a.startTransfer(paths, pane.Path, move)
}

// dragValueFor returns the value a row drags: the chosen paths when the row
// is one of them, else the row alone.
func (a *App) dragValueFor(p *Pane, e fsx.Entry, move bool) dragFiles {
	paths := []string{}
	if p.Selection.Has(e.Path) {
		for path := range p.Selection.All() {
			paths = append(paths, path)
		}
	} else {
		paths = []string{e.Path}
	}
	return dragFiles{Paths: paths, Move: move}
}

// dragDataFor returns the system drag data of a row's files.
func (a *App) dragDataFor(p *Pane, e fsx.Entry) transfer.Data {
	paths := []string{e.Path}
	if p.Selection.Has(e.Path) {
		paths = paths[:0]
		for path := range p.Selection.All() {
			paths = append(paths, path)
		}
	}
	abs := make([]string, 0, len(paths))
	for _, path := range paths {
		if filepath.IsAbs(path) {
			abs = append(abs, path)
		}
	}
	data, err := transfer.FileData(abs...)
	if err != nil {
		return transfer.Data{}
	}
	return data
}

// rowKey is the identity of a row for the list and grid states.
func rowKey(r row) any {
	if r.IsHeader {
		return "header:" + r.Header
	}
	return r.E.Path
}

// rowLabel is the accessible name of a row.
func rowLabel(r row) string {
	if r.IsHeader {
		return r.Header
	}
	return r.E.Name
}

// contextMenu builds the menu of a row in a pane.
func (a *App) contextMenu(p *Pane, i int) func(m *ui.Menu) {
	return func(m *ui.Menu) {
		if i < 0 || i >= len(p.Rows) {
			return
		}
		r := p.Rows[i]
		if r.IsHeader {
			return
		}
		e := r.E
		if r.TrashID != "" {
			if m.Item(a.t("restoreItem")).Chosen() {
				a.restoreTrashItem(r.TrashID, p)
			}
			m.Separator()
			if m.Item(a.t("deleteItem")).Chosen() {
				a.confirm = confirmState{
					Open: true, Title: a.t("trashConfirm"), Body: a.t("trashConfirmBody"),
					Confirm: a.t("deleteItem"), Danger: true,
					Action: func() {
						if err := a.trash.Remove(r.TrashID); err != nil {
							a.toast = a.t("deleteFailed")
						} else {
							if orig := p.TrashOriginal[r.E.Path]; orig != "" {
								a.forgetPath(orig)
							}
							a.refreshPane(p)
						}
					},
				}
			}
			return
		}
		if m.Item(a.t("open")).Shortcut(0, ui.KeyEnter).Chosen() {
			a.activateEntry(p, e)
		}
		if !e.IsDir {
			if m.Item(a.t("openWith")).Chosen() {
				_ = platform.OpenWith(e.Path)
			}
			if m.Item(a.t("openLocation")).Chosen() {
				_ = platform.Reveal(e.Path)
			}
		}
		m.Separator()
		if m.Item(a.t("cut")).Shortcut(ui.Cmd, ui.KeyX).Chosen() {
			a.stagePaths([]string{e.Path}, true)
		}
		if m.Item(a.t("copy")).Shortcut(ui.Cmd, ui.KeyC).Chosen() {
			a.stagePaths([]string{e.Path}, false)
		}
		if m.Item(a.t("copyPath")).Chosen() {
			mygo.Clipboard.WriteText(e.Path)
		}
		if m.Item(a.t("copyName")).Chosen() {
			mygo.Clipboard.WriteText(e.Name)
		}
		m.Separator()
		if m.Item(a.t("rename")).Shortcut(0, ui.KeyF2).Chosen() {
			a.beginRename(p, e)
		}
		if len(a.selectedEntries(p)) > 1 {
			if m.Item(a.t("batchRename")).Chosen() {
				a.beginBatchRename(p)
			}
		}
		if m.Item(a.t("delete")).Shortcut(0, ui.KeyDelete).Chosen() {
			a.deleteEntries(p, []fsx.Entry{e}, false)
		}
		if m.Item(a.t("duplicate")).Chosen() {
			a.duplicateEntry(p, e)
		}
		m.Separator()
		m.Submenu(a.t("addTag"), func(m *ui.Menu) {
			for _, tag := range a.cfg.Tags {
				if m.Item(tag.Name).Radio(a.tagOf(e.Path) == tag.ID).Chosen() {
					a.setTag([]string{e.Path}, tag.ID)
				}
			}
		})
		if a.tagOf(e.Path) != "" {
			if m.Item(a.t("removeTag")).Chosen() {
				a.setTag([]string{e.Path}, "")
			}
		}
		m.Separator()
		if m.Item(a.t("properties")).Chosen() {
			a.showProperties([]fsx.Entry{e})
		}
	}
}

// tagOf returns the tag assigned to a path.
func (a *App) tagOf(path string) string {
	return a.cfg.TagAssignments[path]
}

// isFavorite reports whether a path is starred.
func (a *App) isFavorite(path string) bool {
	for _, f := range a.cfg.Favorites {
		if f.Path == path {
			return true
		}
	}
	return false
}

// toggleFavorite stars or unstars a path.
func (a *App) toggleFavorite(e fsx.Entry) {
	for i, f := range a.cfg.Favorites {
		if f.Path == e.Path {
			a.cfg.Favorites = append(a.cfg.Favorites[:i], a.cfg.Favorites[i+1:]...)
			a.saveSessionSoon()
			return
		}
	}
	a.cfg.Favorites = append(a.cfg.Favorites, storeFavorite(e.Name, e.Path))
	a.saveSessionSoon()
}

func storeFavorite(name, path string) store.Favorite {
	return store.Favorite{Name: name, Path: path}
}

// emptyMessage returns the text a pane shows when it has no rows.
func (a *App) emptyMessage(p *Pane) string {
	switch p.Virtual {
	case virtualComputer:
		return a.t("noResults")
	case virtualTrash:
		return a.t("trashEmpty")
	case virtualTag:
		return a.t("noTagged")
	case virtualStarred:
		return a.t("starredEmpty")
	case virtualRecent:
		return a.t("noRecent")
	case virtualNetwork:
		return a.t("networkEmpty")
	case virtualSearch:
		return a.t("noResults")
	}
	return a.t("emptyFolder")
}

// emptyState draws the empty folder message.
func (a *App) emptyState(c *ui.Context, p Palette, pane *Pane) {
	ui.Column(c).Fill().Center().Gap(10).Children(func() {
		ui.Image(c, kindIcon(fsx.KindFolder)).Size(52, 52).Opacity(0.5)
		ui.Text(c, a.emptyMessage(pane)).FontSize(13).TextColor(p.TextMuted)
		if pane.Err != "" {
			ui.Text(c, pane.Err).FontSize(11).TextColor(p.Danger).MaxLines(3)
		}
	})
}

// paneSubtitle returns what a pane's header shows under the name.
func paneSubtitle(p *Pane) string {
	if p.Virtual != virtualNone {
		return ""
	}
	return p.Path
}

// activatePane makes a pane the one commands act on.
func (a *App) activatePane(pane *Pane) {
	if pane == nil || pane.viewTab == nil {
		return
	}
	if pane.viewTab.Active != pane.viewIdx {
		pane.viewTab.Active = pane.viewIdx
	}
	if pane.viewTab == a.right {
		a.activeSide = 1
	} else {
		a.activeSide = 0
	}
	a.refreshWatchers()
}

// cleanName removes path separators from a typed name.
func cleanName(name string) string {
	return strings.TrimSpace(name)
}

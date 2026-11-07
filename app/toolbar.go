package app

import (
	"path/filepath"
	"strings"

	"pulse/internal/fsx"
	"pulse/internal/platform"

	"github.com/egoist/mygo/ui"
)

// toolbar draws the address bar and the file commands.
func (a *App) toolbar(c *ui.Context, p Palette) {
	pane := a.activePane()
	ui.Column(c).Background(p.Panel).BorderWidth(0, 0, 1, 0).BorderColor(p.BorderSoft).Children(func() {
		ui.Row(c).Height(42).Padding(8, 10).Gap(6).Children(func() {
			if pane == nil {
				return
			}
			back := toolbarButton(c, p, "back", a.t("back"), pane.Virtual == virtualNone && pane.HistAt <= 0)
			if back.Clicked() {
				a.goBack(pane)
			}
			fwd := toolbarButton(c, p, "forward", a.t("forward"), pane.Virtual == virtualNone || pane.HistAt >= len(pane.History)-1)
			if fwd.Clicked() {
				a.goForward(pane)
			}
			up := toolbarButton(c, p, "up", a.t("up"), false)
			if up.Clicked() {
				a.goUp(pane)
			}
			// The address bar: breadcrumbs, or a field while editing.
			addr := ui.Box(c).Grow(1).Height(30).Radius(6).Background(p.Surface).Border(1, p.BorderSoft)
			addr.Children(func() {
				if a.addressEdit {
					ti := ui.Autocomplete(c, &a.addressDraft, a.addressSuggestions()).AutoFocus().Bind(&a.addressHandle).FontSize(12)
					if ti.Submitted() {
						a.navigateAddress(pane, a.addressDraft)
						a.addressEdit = false
					}
				} else {
					items, paths := a.breadcrumb(pane)
					chosen := len(items) - 1
					bc := ui.Breadcrumbs(c, items, &chosen).Label(a.t("locationLabel"))
					if bc.Changed() && pane.Virtual == virtualNone && chosen >= 0 && chosen < len(paths) {
						if paths[chosen] == "" {
							a.openVirtual(pane, virtualComputer, "")
						} else {
							a.navigate(pane, paths[chosen])
						}
					}
				}
			})
			addr.OnClick(func() {
				if !a.addressEdit {
					a.addressEdit = true
					a.addressDraft = pane.Path
					a.addressHandle.Focus()
				}
			})
			refresh := toolbarButton(c, p, "refresh", a.t("refresh"), false)
			if refresh.Clicked() {
				a.refreshPane(pane)
			}
			term := toolbarButton(c, p, "terminal", a.t("terminal"), pane.Virtual != virtualNone)
			if term.Clicked() {
				if err := platform.OpenTerminal(pane.Path); err != nil {
					a.toast = a.t("terminalFailed")
				}
			}
			sf := ui.SearchField(c, &a.searchQuery).Width(220).Placeholder(a.t("searchInFolder")).Bind(&a.searchHandle)
			if sf.Changed() {
				pane.Query = a.searchQuery
				a.applyPane(pane)
			}
			if sf.Submitted() {
				a.openSearchOverlay()
				a.startSearch()
			}
			ui.Text(c, "Ctrl+K").FontSize(9.5).TextColor(p.TextFaint)
		})
		ui.Row(c).Height(38).Padding(4, 10).Gap(5).Children(func() {
			if pane == nil {
				return
			}
			newBtn := ui.PrimaryButton(c, "").Height(30).Radius(6).Padding(8, 12).Children(func() {
				ui.Row(c).Gap(5).Children(func() {
					ui.Icon(c, icon("plus")).FontSize(14)
					ui.Text(c, a.t("new")).FontSize(11.5).SingleLine()
					ui.Icon(c, icon("chevron-down")).FontSize(11)
				})
			}).PopoverMenu(func(m *ui.Menu) {
				if m.Item(a.t("newFolder")).Shortcut(ui.Cmd|ui.Shift, ui.KeyN).Chosen() {
					a.beginNewFolder(pane)
				}
				if m.Item(a.t("newFile")).Chosen() {
					a.beginNewFile(pane)
				}
				if m.Item(a.t("newTab")).Shortcut(ui.Cmd, ui.KeyT).Chosen() {
					a.newTab(pane.Path)
				}
			})
			_ = newBtn
			sel := len(a.selectedEntries(pane))
			cut := toolbarButton(c, p, "cut", a.t("cut"), sel == 0)
			if cut.Clicked() {
				a.stageEntries(pane, true)
			}
			cp := toolbarButton(c, p, "copy", a.t("copy"), sel == 0)
			if cp.Clicked() {
				a.stageEntries(pane, false)
			}
			paste := toolbarButton(c, p, "paste", a.t("paste"), len(a.stash.Items) == 0)
			if paste.Clicked() {
				a.pasteStash()
			}
			ren := toolbarButton(c, p, "rename", a.t("rename"), sel == 0)
			if ren.Clicked() {
				if sel > 1 {
					a.beginBatchRename(pane)
				} else if e := a.selectedEntry(pane); e != nil {
					a.beginRename(pane, *e)
				}
			}
			del := toolbarButton(c, p, "delete", a.t("delete"), sel == 0)
			if del.Clicked() {
				a.deleteEntries(pane, a.selectedEntries(pane), false)
			}
			ui.Box(c).Width(1).Height(22).Background(p.BorderSoft)
			// Sort.
			sortLabel := a.t("sortName")
			switch pane.SortBy {
			case "date":
				sortLabel = a.t("sortDate")
			case "type":
				sortLabel = a.t("sortType")
			case "size":
				sortLabel = a.t("sortSize")
			}
			_ = sortLabel
			menuButton(c, p, "sort", a.t("sort"), func(m *ui.Menu) {
				if m.Item(a.t("sortName")).Radio(pane.SortBy == "name").Chosen() {
					a.setPaneSort(pane, "name", pane.SortDesc)
				}
				if m.Item(a.t("sortDate")).Radio(pane.SortBy == "date").Chosen() {
					a.setPaneSort(pane, "date", pane.SortDesc)
				}
				if m.Item(a.t("sortType")).Radio(pane.SortBy == "type").Chosen() {
					a.setPaneSort(pane, "type", pane.SortDesc)
				}
				if m.Item(a.t("sortSize")).Radio(pane.SortBy == "size").Chosen() {
					a.setPaneSort(pane, "size", pane.SortDesc)
				}
				m.Separator()
				if m.Item(a.t("ascending")).Radio(!pane.SortDesc).Chosen() {
					a.setPaneSort(pane, pane.SortBy, false)
				}
				if m.Item(a.t("descending")).Radio(pane.SortDesc).Chosen() {
					a.setPaneSort(pane, pane.SortBy, true)
				}
				m.Separator()
				ff := a.cfg.View.FoldersFirst
				if m.Item(a.t("foldersFirst")).Checked(ff).Chosen() {
					a.cfg.View.FoldersFirst = !ff
					a.applyPane(pane)
					a.saveSessionSoon()
				}
			})
			// Filter.
			filterLabel := a.t("filter")
			if pane.KindFilter != "" && pane.KindFilter != "all" {
				filterLabel = a.filterName(pane.KindFilter)
			}
			_ = filterLabel
			menuButton(c, p, "filter", a.t("filter"), func(m *ui.Menu) {
				for _, f := range []struct{ id, key string }{
					{"all", "filterAll"}, {"folders", "filterFolders"}, {"documents", "filterDocuments"},
					{"images", "filterImages"}, {"videos", "filterVideos"}, {"audio", "filterAudio"},
					{"archives", "filterArchives"}, {"code", "filterCode"}, {"other", "filterOther"},
				} {
					if m.Item(a.t(f.key)).Radio(pane.KindFilter == f.id || pane.KindFilter == "" && f.id == "all").Chosen() {
						if f.id == "all" {
							a.setPaneFilter(pane, "")
						} else {
							a.setPaneFilter(pane, f.id)
						}
					}
				}
			})
			// Group.
			groupLabel := a.t("groupNone")
			switch pane.Group {
			case "name":
				groupLabel = a.t("groupName")
			case "date":
				groupLabel = a.t("groupDate")
			case "type":
				groupLabel = a.t("groupType")
			case "size":
				groupLabel = a.t("groupSize")
			}
			_ = groupLabel
			menuButton(c, p, "group", a.t("group"), func(m *ui.Menu) {
				for _, g := range []struct{ id, key string }{
					{"none", "groupNone"}, {"name", "groupName"}, {"date", "groupDate"},
					{"type", "groupType"}, {"size", "groupSize"},
				} {
					if m.Item(a.t(g.key)).Radio(pane.Group == g.id || pane.Group == "" && g.id == "none").Chosen() {
						a.setPaneGroup(pane, g.id)
					}
				}
			})
			ui.Spacer(c)
			if flatTextButton(c, p, a.t("compare"), 64, 26).Clicked() {
				a.startFolderCompare()
			}
			dt := toolbarButton(c, p, "details-pane", a.t("showDetails"), false)
			if a.detailsOn {
				dt.TextColor(p.Accent)
			}
			if dt.Clicked() {
				a.detailsOn = !a.detailsOn
				a.cfg.View.ShowDetails = a.detailsOn
				a.saveSessionSoon()
			}
		})
	})
}

// toolbarButton draws a flat icon button.
func toolbarButton(c *ui.Context, p Palette, iconName, tip string, disabled bool) ui.Element {
	b := flatButton(c, p, 28, 28)
	b.Disabled(disabled)
	return b.Children(func() {
		ui.Icon(c, icon(iconName)).FontSize(15).TextColor(p.TextMuted)
	}).Tooltip(tip).Label(tip)
}

// menuButton draws a toolbar button with an icon, a label and a menu.
func menuButton(c *ui.Context, p Palette, iconName, label string, build func(m *ui.Menu)) ui.Element {
	return flatMenuButton(c, p, iconName, label, build)
}

// filterName names a kind filter.
func (a *App) filterName(f string) string {
	switch f {
	case "folders":
		return a.t("filterFolders")
	case "documents":
		return a.t("filterDocuments")
	case "images":
		return a.t("filterImages")
	case "videos":
		return a.t("filterVideos")
	case "audio":
		return a.t("filterAudio")
	case "archives":
		return a.t("filterArchives")
	case "code":
		return a.t("filterCode")
	case "other":
		return a.t("filterOther")
	}
	return a.t("filterAll")
}

// breadcrumb returns the labels and paths of a pane's location.
func (a *App) breadcrumb(pane *Pane) ([]string, []string) {
	if pane == nil {
		return []string{a.t("computer")}, []string{""}
	}
	switch pane.Virtual {
	case virtualComputer:
		return []string{a.t("computer")}, []string{""}
	case virtualTrash:
		return []string{a.t("recycleBin")}, []string{""}
	case virtualTag:
		for _, tag := range a.cfg.Tags {
			if tag.ID == pane.TagID {
				return []string{a.tf("tabTag", tag.Name)}, []string{""}
			}
		}
		return []string{a.t("tags")}, []string{""}
	case virtualStarred:
		return []string{a.t("starred")}, []string{""}
	case virtualRecent:
		return []string{a.t("recent")}, []string{""}
	case virtualNetwork:
		return []string{a.t("network")}, []string{""}
	case virtualSearch:
		return []string{a.t("tabSearch")}, []string{""}
	}
	path := filepath.Clean(pane.Path)
	vol := filepath.VolumeName(path)
	var items []string
	var paths []string
	items = append(items, a.t("computer"))
	paths = append(paths, "")
	rest := strings.TrimPrefix(path, vol)
	rest = strings.TrimLeft(rest, string(filepath.Separator))
	parts := []string{}
	if rest != "" {
		parts = strings.Split(rest, string(filepath.Separator))
	}
	if vol != "" {
		items = append(items, vol)
		paths = append(paths, vol+string(filepath.Separator))
	} else {
		items = append(items, "/")
		paths = append(paths, "/")
	}
	at := paths[len(paths)-1]
	for _, part := range parts {
		if part == "" {
			continue
		}
		at = filepath.Join(at, part)
		items = append(items, part)
		paths = append(paths, at)
	}
	if len(items) == 1 {
		items = append(items, path)
		paths = append(paths, path)
	}
	return items, paths
}

// addressSuggestions lists paths the address bar suggests.
func (a *App) addressSuggestions() []string {
	seen := map[string]bool{}
	var out []string
	add := func(path string) {
		if path == "" || seen[path] {
			return
		}
		seen[path] = true
		out = append(out, path)
	}
	add(platform.Home())
	for _, d := range platform.UserDirs() {
		add(d.Path)
	}
	for _, d := range a.drivesCached() {
		add(d.Path)
	}
	for _, f := range a.cfg.Favorites {
		add(f.Path)
	}
	for _, f := range a.cfg.Quick {
		add(f.Path)
	}
	for _, r := range a.recent {
		add(r.Path)
	}
	if len(out) > 60 {
		out = out[:60]
	}
	return out
}

// navigateAddress goes to a typed path, or runs a terminal command.
func (a *App) navigateAddress(pane *Pane, text string) {
	t := strings.TrimSpace(text)
	if t == "" {
		return
	}
	switch strings.ToLower(t) {
	case "cmd", "powershell", "pwsh", "wt", "terminal":
		if err := platform.OpenTerminal(pane.Path); err != nil {
			a.toast = a.t("terminalFailed")
		}
		return
	}
	if strings.HasPrefix(t, "~") {
		t = filepath.Join(platform.Home(), strings.TrimPrefix(t, "~"))
	}
	if !filepath.IsAbs(t) {
		t = filepath.Join(pane.Path, t)
	}
	t = filepath.Clean(t)
	if !platform.Exists(t) {
		a.toast = a.t("notFound")
		return
	}
	if info, err := fsx.Stat(t); err == nil && !info.IsDir {
		a.revealEntry(pane, info)
		return
	}
	if pane.Virtual != virtualNone {
		pane.Virtual = virtualNone
	}
	a.navigate(pane, t)
}

// focusAddress asks for the address bar's focus.
func (a *App) focusAddress() {
	pane := a.activePane()
	if pane == nil {
		return
	}
	a.addressEdit = true
	a.addressDraft = pane.Path
	a.addressHandle.Focus()
}

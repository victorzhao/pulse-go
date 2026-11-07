package app

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"pulse/internal/fsx"
	"pulse/internal/platform"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// openSearchOverlay shows the global search dialog.
func (a *App) openSearchOverlay() {
	pane := a.activePane()
	if pane != nil {
		a.search.Root = pane.Path
	}
	a.search.Open = true
	if a.search.Query == "" {
		a.search.Query = a.searchQuery
	}
}

// startSearch runs the search dialog's search.
func (a *App) startSearch() {
	if a.Win == nil {
		return
	}
	if a.search.Cancel != nil {
		a.search.Cancel()
	}
	q := strings.TrimSpace(a.search.Query)
	if q != "" {
		a.addSearchHistory(q)
	}
	if q == "" && a.search.Kind == "" && a.search.Date == "" && a.search.Size == "" && a.search.Tag == "" {
		a.search.Err = a.t("searchNoQuery")
		return
	}
	var roots []string
	switch a.search.Scope {
	case 1:
		for _, d := range a.drivesCached() {
			roots = append(roots, d.Path)
		}
	case 2:
		if a.search.Root != "" {
			roots = []string{a.search.Root}
		}
	default:
		if pane := a.activePane(); pane != nil && pane.Virtual == virtualNone {
			roots = []string{pane.Path}
		} else {
			roots = []string{platform.Home()}
		}
	}
	if len(roots) == 0 {
		a.search.Err = a.t("folderNotFound")
		return
	}
	tagAssign := map[string]string{}
	for k, v := range a.cfg.TagAssignments {
		tagAssign[k] = v
	}
	tagFilter := a.search.Tag
	depth := 20
	if !a.search.Subfolders {
		depth = 1
	}
	filter := fsx.SearchFilter{
		Query:         q,
		MaxDepth:      depth,
		Limit:         1500,
		IncludeHidden: a.search.Hidden,
	}
	switch a.search.Kind {
	case "folders":
		filter.FoldersOnly = true
	case "files":
		filter.FilesOnly = true
	case "documents":
		filter.Kinds = []fsx.Kind{fsx.KindDocument, fsx.KindPDF, fsx.KindSheet, fsx.KindSlides, fsx.KindText}
	case "images":
		filter.Kinds = []fsx.Kind{fsx.KindImage}
	case "videos":
		filter.Kinds = []fsx.Kind{fsx.KindVideo}
	case "audio":
		filter.Kinds = []fsx.Kind{fsx.KindAudio}
	case "archives":
		filter.Kinds = []fsx.Kind{fsx.KindArchive}
	case "code":
		filter.Kinds = []fsx.Kind{fsx.KindCode}
	}
	now := time.Now()
	switch a.search.Date {
	case "today":
		y, m, d := now.Date()
		filter.After = time.Date(y, m, d, 0, 0, 0, 0, time.Local).Unix()
	case "last7":
		filter.After = now.AddDate(0, 0, -7).Unix()
	case "last30":
		filter.After = now.AddDate(0, 0, -30).Unix()
	case "thisYear":
		filter.After = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.Local).Unix()
	}
	switch a.search.Size {
	case "small":
		filter.MaxSize = 100 * 1024
	case "medium":
		filter.MinSize = 100 * 1024
		filter.MaxSize = 10 * 1024 * 1024
	case "large":
		filter.MinSize = 10 * 1024 * 1024
	}
	a.search.Gen++
	gen := a.search.Gen
	a.search.Results = nil
	a.search.Done = 0
	a.search.Err = ""
	a.search.Searching = true
	a.search.Selected = -1
	ctx, cancel := context.WithCancel(context.Background())
	a.search.Cancel = cancel
	go func() {
		seen := map[string]bool{}
		var batch []fsx.Entry
		last := time.Now()
		flush := func() {
			if len(batch) == 0 {
				return
			}
			items := batch
			batch = nil
			a.Win.Update(func() {
				if a.search.Gen != gen {
					return
				}
				for _, e := range items {
					if seen[e.Path] {
						continue
					}
					seen[e.Path] = true
					a.search.Results = append(a.search.Results, e)
				}
				a.search.Done = len(a.search.Results)
			})
			last = time.Now()
		}
		for _, root := range roots {
			if ctx.Err() != nil {
				break
			}
			f := filter
			f.Root = root
			err := fsx.Search(ctx, f, func(e fsx.Entry) bool {
				if ctx.Err() != nil {
					return false
				}
				if tagFilter != "" {
					if tagAssign[e.Path] != tagFilter {
						return true
					}
				}
				if seen[e.Path] {
					return true
				}
				batch = append(batch, e)
				if len(batch) >= 40 || time.Since(last) > 120*time.Millisecond {
					flush()
				}
				return true
			})
			if err != nil {
				a.Win.Update(func() {
					if a.search.Gen == gen {
						a.search.Err = err.Error()
					}
				})
			}
			flush()
			if ctx.Err() != nil {
				break
			}
		}
		flush()
		cancelled := ctx.Err() != nil
		a.Win.Update(func() {
			if a.search.Gen != gen {
				return
			}
			a.search.Searching = false
			if cancelled {
				a.search.Done = len(a.search.Results)
			}
		})
	}()
}

// addSearchHistory records a query.
func (a *App) addSearchHistory(q string) {
	if q == "" {
		return
	}
	for i, old := range a.searchHistory {
		if old == q {
			a.searchHistory = append(a.searchHistory[:i], a.searchHistory[i+1:]...)
			break
		}
	}
	a.searchHistory = append([]string{q}, a.searchHistory...)
	if len(a.searchHistory) > 20 {
		a.searchHistory = a.searchHistory[:20]
	}
	a.saveSessionSoon()
}

// stopSearch cancels a running search.
func (a *App) stopSearch() {
	if a.search.Cancel != nil {
		a.search.Cancel()
		a.search.Cancel = nil
	}
	a.search.Searching = false
}

// searchModal shows the global search dialog.
func (a *App) searchModal(c *ui.Context, p Palette) {
	if !a.search.Open {
		return
	}
	m := ui.Modal(c, &a.search.Open, func() {
		ui.Column(c).Width(760).Height(540).Gap(8).Children(func() {
			ui.Row(c).Gap(8).Children(func() {
				ui.Icon(c, icon("search")).FontSize(17).TextColor(p.Accent)
				ui.Text(c, a.t("globalSearch")).FontSize(15).Bold()
				ui.Spacer(c)
				if ui.Button(c, a.t("close")).Clicked() {
					a.stopSearch()
					a.search.Open = false
				}
			})
			ui.Row(c).Gap(8).Children(func() {
				ti := ui.Autocomplete(c, &a.search.Query, a.searchHistory).Grow(1).Placeholder(a.t("searchPlaceholderLong")).AutoFocus()
				if ti.Submitted() {
					a.startSearch()
				}
				if a.search.Searching {
					if ui.Button(c, a.t("stop")).Clicked() {
						a.stopSearch()
					}
				} else if ui.PrimaryButton(c, a.t("startSearch")).Clicked() {
					a.startSearch()
				}
			})
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				scope := a.search.Scope
				if ui.Segmented(c, &scope, a.t("currentFolder"), a.t("thisPC"), a.t("pickedFolder")).Changed() {
					a.search.Scope = scope
				}
				ui.Text(c, a.scopeLabel()).FontSize(10).TextColor(p.TextMuted).SingleLine().Grow(1)
				if a.search.Scope == 2 {
					if ui.Button(c, a.t("browse")).FontSize(10).Height(22).Clicked() {
						paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.Win, Title: a.t("searchFolderPicker"), Directory: true})
						if err == nil && len(paths) > 0 {
							a.search.Root = paths[0]
						}
					}
				}
				hidden := a.search.Hidden
				if ui.Checkbox(c, &hidden, a.t("includeHidden")).Changed() {
					a.search.Hidden = hidden
				}
				sub := a.search.Subfolders
				if ui.Checkbox(c, &sub, a.t("includeSubfolders")).Changed() {
					a.search.Subfolders = sub
				}
				if ui.Button(c, a.t("advancedSearch")).Clicked() {
					a.advanced.Open = true
					a.advanced.Query = a.search.Query
					a.advanced.Scope = a.search.Scope
					a.advanced.Root = a.search.Root
				}
			})
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, a.t("searchTagsLabel")).FontSize(10).TextColor(p.TextMuted)
				tagLabel := a.t("anyTag")
				for _, tag := range a.cfg.Tags {
					if tag.ID == a.search.Tag {
						tagLabel = tag.Name
					}
				}
				tagOptions := []string{a.t("anyTag")}
				for _, tag := range a.cfg.Tags {
					tagOptions = append(tagOptions, tag.Name)
				}
				ui.Select(c, &tagLabel, tagOptions).Width(140)
				if tagLabel != a.t("anyTag") {
					a.search.Tag = a.tagIDByName(tagLabel)
				} else {
					a.search.Tag = ""
				}
				ui.Spacer(c)
			})
			if a.search.Searching {
				ui.Row(c).Gap(8).Children(func() {
					ui.Spinner(c).Size(14, 14)
					ui.Text(c, a.tf("searchingProgress", a.search.Done)).FontSize(10.5).TextColor(p.Accent)
				})
			} else if a.search.Err != "" {
				ui.Text(c, a.search.Err).FontSize(10.5).TextColor(p.Danger)
			} else {
				ui.Text(c, a.tf("searchDone", a.search.Done)).FontSize(10.5).TextColor(p.TextMuted)
			}
			ui.Divider(c)
			if len(a.search.Results) == 0 {
				ui.Column(c).Fill().Center().Children(func() {
					ui.Text(c, a.t("noResults")).FontSize(12).TextColor(p.TextMuted)
				})
			} else {
				a.search.List.Key = func(i int) any { return a.search.Results[i].Path }
				a.search.List.Label = func(i int) string { return a.search.Results[i].Name }
				a.search.List.Selected = &a.search.Selected
				list := ui.List(c, &a.search.List, len(a.search.Results), func(i int) {
					if i < 0 || i >= len(a.search.Results) {
						return
					}
					e := a.search.Results[i]
					selected := a.search.Selected == i
					nameColor, pathColor, sizeColor := p.Text, p.TextFaint, p.TextMuted
					if selected {
						nameColor, pathColor, sizeColor = p.AccentText, p.AccentText, p.AccentText
					}
					row := ui.Row(c).Height(30).Gap(8).Padding(2, 6)
					row.ContextMenu(func(m *ui.Menu) {
						if m.Item(a.t("open")).Chosen() {
							a.openSearchResult(e)
						}
						if m.Item(a.t("revealResult")).Chosen() {
							a.revealSearchResult(e)
						}
						if m.Item(a.t("copyPath")).Chosen() {
							mygo.Clipboard.WriteText(e.Path)
						}
					})
					row.Children(func() {
						ui.Image(c, kindIcon(e.Kind)).Size(16, 16)
						ui.Column(c).Grow(1).Gap(0).Justify(ui.Center).Children(func() {
							if a.search.Query != "" && !selected {
								a.highlightedName(c, p, e.Name, a.search.Query)
							} else {
								ui.Text(c, e.Name).FontSize(11.5).SingleLine().TextColor(nameColor)
							}
							ui.Text(c, filepath.Dir(e.Path)).FontSize(9.5).SingleLine().TextColor(pathColor)
						})
						if !e.IsDir {
							ui.Text(c, fsx.FormatSize(e.Size)).FontSize(10).TextColor(sizeColor)
						}
					})
				}).Grow(1)
				if list.Submitted() && a.search.Selected >= 0 && a.search.Selected < len(a.search.Results) {
					a.openSearchResult(a.search.Results[a.search.Selected])
				}
			}
			ui.Row(c).Gap(8).Children(func() {
				if ui.Button(c, a.t("showResultsInPane")).Disabled(len(a.search.Results) == 0).Clicked() {
					root := ""
					if len(a.search.Results) > 0 {
						root = filepath.Dir(a.search.Results[0].Path)
					}
					a.openSearchTab(a.search.Query, root, a.search.Results)
					a.search.Open = false
				}
				ui.Spacer(c)
				if ui.Button(c, a.t("open")).Disabled(a.search.Selected < 0).Clicked() {
					if a.search.Selected >= 0 && a.search.Selected < len(a.search.Results) {
						a.openSearchResult(a.search.Results[a.search.Selected])
					}
				}
				if ui.Button(c, a.t("revealResult")).Disabled(a.search.Selected < 0).Clicked() {
					if a.search.Selected >= 0 && a.search.Selected < len(a.search.Results) {
						a.revealSearchResult(a.search.Results[a.search.Selected])
					}
				}
			})
		})
	})
	if m.Dismissed() {
		a.stopSearch()
	}
}

// tagIDByName returns a tag's ID from its name.
func (a *App) tagIDByName(name string) string {
	for _, tag := range a.cfg.Tags {
		if tag.Name == name {
			return tag.ID
		}
	}
	return ""
}

// scopeLabel names the search scope.
func (a *App) scopeLabel() string {
	switch a.search.Scope {
	case 1:
		return a.t("searchThisPC")
	case 2:
		if a.search.Root != "" {
			return a.search.Root
		}
		return a.t("searchScopeFolder")
	}
	if pane := a.activePane(); pane != nil && pane.Virtual == virtualNone {
		return pane.Path
	}
	return a.t("searchCurrent")
}

// openSearchResult opens a result in the active pane.
func (a *App) openSearchResult(e fsx.Entry) {
	pane := a.activePane()
	if pane == nil {
		return
	}
	a.search.Open = false
	if e.IsDir {
		a.navigate(pane, e.Path)
		return
	}
	a.revealEntry(pane, e)
	if err := platform.Open(e.Path); err != nil {
		a.toast = a.t("openFailed")
	}
}

// revealSearchResult shows a result in the active pane.
func (a *App) revealSearchResult(e fsx.Entry) {
	pane := a.activePane()
	if pane == nil {
		return
	}
	a.search.Open = false
	a.revealEntry(pane, e)
}

// advancedModal shows the advanced search dialog.
func (a *App) advancedModal(c *ui.Context, p Palette) {
	if !a.advanced.Open {
		return
	}
	ui.Modal(c, &a.advanced.Open, func() {
		ui.Column(c).Width(560).Gap(10).Children(func() {
			ui.Row(c).Gap(8).Children(func() {
				ui.Icon(c, icon("filter")).FontSize(17).TextColor(p.Accent)
				ui.Text(c, a.t("advancedTitle")).FontSize(15).Bold()
				ui.Spacer(c)
				if ui.Button(c, a.t("close")).Clicked() {
					a.advanced.Open = false
				}
			})
			ui.Text(c, a.t("advancedSearchHint")).FontSize(10.5).TextColor(p.TextMuted)
			ui.TextInput(c, &a.advanced.Query).Placeholder(a.t("searchNamePlaceholder"))
			ui.Row(c).Gap(8).Children(func() {
				ui.Text(c, a.t("kind")).FontSize(11).Width(80).TextColor(p.TextMuted)
				kind := a.advanced.Kind
				ui.Select(c, &kind, []string{a.t("anyKind"), a.t("filterFolders"), a.t("filterDocuments"), a.t("filterImages"), a.t("filterVideos"), a.t("filterAudio"), a.t("filterArchives"), a.t("filterCode"), a.t("filterOther")})
				if kind != a.advanced.Kind {
					a.advanced.Kind = kind
				}
			})
			ui.Row(c).Gap(8).Children(func() {
				ui.Text(c, a.t("date")).FontSize(11).Width(80).TextColor(p.TextMuted)
				date := a.advanced.Date
				ui.Select(c, &date, []string{a.t("anyDate"), a.t("today"), a.t("last7"), a.t("last30"), a.t("thisYear")})
				if date != a.advanced.Date {
					a.advanced.Date = date
				}
			})
			ui.Row(c).Gap(8).Children(func() {
				ui.Text(c, a.t("sizeRange")).FontSize(11).Width(80).TextColor(p.TextMuted)
				size := a.advanced.Size
				ui.Select(c, &size, []string{a.t("anySize"), a.t("small"), a.t("medium"), a.t("large")})
				if size != a.advanced.Size {
					a.advanced.Size = size
				}
			})
			ui.Row(c).Gap(8).Children(func() {
				ui.Text(c, a.t("searchTagsLabel")).FontSize(11).Width(80).TextColor(p.TextMuted)
				tag := a.advanced.Tag
				opts := []string{a.t("anyTag")}
				for _, t := range a.cfg.Tags {
					opts = append(opts, t.Name)
				}
				ui.Select(c, &tag, opts)
				if tag != a.advanced.Tag {
					a.advanced.Tag = tag
				}
			})
			ui.Checkbox(c, &a.advanced.Hidden, a.t("includeHidden"))
			ui.Checkbox(c, &a.advanced.Subfolders, a.t("includeSubfolders"))
			ui.Row(c).Gap(8).Children(func() {
				ui.Text(c, a.t("searchScope")).FontSize(11).Width(80).TextColor(p.TextMuted)
				scope := a.advanced.Scope
				if ui.Segmented(c, &scope, a.t("currentFolder"), a.t("thisPC"), a.t("pickedFolder")).Changed() {
					a.advanced.Scope = scope
				}
			})
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, a.t("cancel")).Clicked() {
					a.advanced.Open = false
				}
				if ui.PrimaryButton(c, a.t("startSearch")).Clicked() {
					a.applyAdvancedSearch()
				}
			})
		})
	})
}

// applyAdvancedSearch turns the advanced filters into a search.
func (a *App) applyAdvancedSearch() {
	a.search.Query = a.advanced.Query
	a.search.Scope = a.advanced.Scope
	a.search.Root = a.advanced.Root
	a.search.Kind = mapFilterKind(a, a.advanced.Kind)
	a.search.Date = mapFilterDate(a, a.advanced.Date)
	a.search.Size = mapFilterSize(a, a.advanced.Size)
	a.search.Tag = ""
	for _, tag := range a.cfg.Tags {
		if tag.Name == a.advanced.Tag {
			a.search.Tag = tag.ID
			break
		}
	}
	a.search.Hidden = a.advanced.Hidden
	a.search.Subfolders = a.advanced.Subfolders
	a.advanced.Open = false
	a.search.Open = true
	a.startSearch()
}

func mapFilterKind(a *App, label string) string {
	switch label {
	case a.t("anyKind"):
		return ""
	case a.t("filterFolders"):
		return "folders"
	case a.t("filterDocuments"):
		return "documents"
	case a.t("filterImages"):
		return "images"
	case a.t("filterVideos"):
		return "videos"
	case a.t("filterAudio"):
		return "audio"
	case a.t("filterArchives"):
		return "archives"
	case a.t("filterCode"):
		return "code"
	case a.t("filterOther"):
		return "other"
	}
	return ""
}

func mapFilterDate(a *App, label string) string {
	switch label {
	case a.t("anyDate"):
		return ""
	case a.t("today"):
		return "today"
	case a.t("last7"):
		return "last7"
	case a.t("last30"):
		return "last30"
	case a.t("thisYear"):
		return "thisYear"
	}
	return ""
}

func mapFilterSize(a *App, label string) string {
	switch label {
	case a.t("anySize"):
		return ""
	case a.t("small"):
		return "small"
	case a.t("medium"):
		return "medium"
	case a.t("large"):
		return "large"
	}
	return ""
}

package app

import (
	"os"
	"strings"
	"time"

	"pulse/internal/fsx"
	"pulse/internal/platform"

	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
)

// detailsView shows the pane as a table of columns.
func (a *App) detailsView(c *ui.Context, p Palette, pane *Pane) {
	a.sortOrderFor(pane)
	pane.List.Key = func(i int) any { return rowKey(pane.Rows[i]) }
	pane.List.Label = func(i int) string { return rowLabel(pane.Rows[i]) }
	pane.List.Header = func(i int) bool { return pane.Rows[i].IsHeader }
	pane.List.Selection = &pane.Selection
	pane.List.Selected = &pane.Selected
	pane.List.Sort = &pane.sortOrder
	pane.List.SortBy = &pane.SortOrders
	pane.List.DragSelect = true
	narrow := a.paneNarrow(c)
	pane.List.HeaderMenu = func(m *ui.Menu, col int) {
		cols := a.tableColumns(pane, narrow)
		if col >= 0 && col < len(cols) {
			if m.Item(a.t("hideColumn")).Chosen() {
				if pane.HiddenCols == nil {
					pane.HiddenCols = map[string]bool{}
				}
				pane.HiddenCols[cols[col].ID] = true
				a.saveFolderPref(pane)
				c.Invalidate()
			}
		}
		if m.Item(a.t("showAllColumns")).Chosen() {
			pane.HiddenCols = map[string]bool{}
			a.saveFolderPref(pane)
			c.Invalidate()
		}
	}
	pane.List.CellMenu = func(m *ui.Menu, row, col int) {
		a.contextMenu(pane, row)(m)
	}
	if pane.Loading && len(pane.Rows) == 0 {
		a.loadingState(c, p, pane)
		return
	}
	if len(pane.Rows) == 0 {
		a.emptyState(c, p, pane)
		return
	}
	cols := a.tableColumns(pane, narrow)
	keys := make([]string, len(cols))
	for i := range cols {
		keys[i] = cols[i].ID
	}
	maxSize := a.maxEntrySize(pane)
	table := ui.Table(c, &pane.List, cols, len(pane.Rows), func(row, col int) {
		if row < 0 || row >= len(pane.Rows) {
			return
		}
		if pane.Rows[row].IsHeader {
			if col == 0 {
				a.groupHeader(c, p, pane.Rows[row].Header)
			}
			return
		}
		switch keys[col] {
		case "name":
			a.nameCell(c, p, pane, row)
		case "date":
			a.dateCell(c, p, pane, row)
		case "type":
			a.typeCell(c, p, pane, row)
		case "size":
			a.sizeCell(c, p, pane, row, maxSize)
		}
	}).Grow(1)
	if table.Changed() {
		a.activatePane(pane)
		if a.cfg.General.SingleClickOpen {
			a.activateRow(pane, pane.Selected)
		}
	}
	if table.Submitted() {
		a.activateRow(pane, pane.Selected)
	}
	if a.syncTableSort(pane) {
		c.Invalidate()
	}
}

// loadingState shows a spinner while a folder loads.
func (a *App) loadingState(c *ui.Context, p Palette, pane *Pane) {
	ui.Column(c).Fill().Center().Gap(10).Children(func() {
		ui.Spinner(c).Size(24, 24)
		ui.Text(c, a.t("loading")).FontSize(12).TextColor(p.TextMuted)
	})
}

// tableColumns returns the visible columns of the details view. A narrow
// pane shows the name and the size, as Pulse's split panes do.
func (a *App) tableColumns(pane *Pane, narrow bool) []ui.TableColumn {
	var all []ui.TableColumn
	if narrow {
		all = []ui.TableColumn{
			{Title: a.t("columnName"), ID: "name", Sortable: true, MinWidth: 120},
			{Title: a.t("columnSize"), ID: "size", Width: 92, MinWidth: 70, Align: ui.End, Sortable: true},
		}
	} else {
		all = []ui.TableColumn{
			{Title: a.t("columnName"), ID: "name", Sortable: true, MinWidth: 180},
			{Title: a.t("columnDate"), ID: "date", Width: 165, MinWidth: 120, Sortable: true},
			{Title: a.t("columnType"), ID: "type", Width: 155, MinWidth: 100, Sortable: true},
			{Title: a.t("columnSize"), ID: "size", Width: 110, MinWidth: 80, Align: ui.End, Sortable: true},
		}
	}
	out := make([]ui.TableColumn, 0, len(all))
	for _, col := range all {
		if pane.HiddenCols[col.ID] {
			continue
		}
		out = append(out, col)
	}
	return out
}

// paneNarrow estimates whether a side has room for every column.
func (a *App) paneNarrow(c *ui.Context) bool {
	w, _ := c.Size()
	if a.sidebarOn {
		w -= a.sidebarWidth
	}
	if a.detailsOn {
		w -= w * (100 - a.detailsPercent) / 100
	}
	return w/2 < 520
}

// maxEntrySize returns the largest size shown, for size bars.
func (a *App) maxEntrySize(pane *Pane) int64 {
	var max int64
	for _, r := range pane.Rows {
		if r.IsHeader {
			continue
		}
		s := entrySize(r.E, pane.Sizes)
		if s > max {
			max = s
		}
	}
	return max
}

// nameCell builds the name column of a row.
func (a *App) nameCell(c *ui.Context, p Palette, pane *Pane, row int) {
	r := pane.Rows[row]
	e := r.E
	zebra := a.cfg.View.Zebra && row%2 == 1
	cell := ui.Row(c).FillWidth().Height(p.RowH).Gap(6).Padding(3, 8).Clip()
	if zebra {
		cell.Background(p.Surface)
	}
	hovered := cell.Hovered()
	cell.Children(func() {
		ui.Image(c, kindIcon(e.Kind)).Size(16, 16)
		if pane.Query != "" {
			ui.Box(c).Grow(1).Children(func() {
				a.highlightedName(c, p, e.Name, pane.Query)
			})
		} else {
			ui.Box(c).Grow(1).Children(func() {
				name := e.Name
				if ui.EditableText(c, &name).Changed() {
					a.renameEntry(pane, e.Path, name)
				}
			})
		}
		if tag := a.tagOf(e.Path); tag != "" {
			ui.Box(c).Size(6, 6).Radius(3).Background(a.tagColor(tag))
		}
		if pane.CompareOn {
			a.compareBadge(c, p, pane, e.Path)
		}
		if a.isFavorite(e.Path) || hovered {
			fav := a.isFavorite(e.Path)
			iconName := "star"
			if fav {
				iconName = "star-filled"
			}
			if ui.Button(c, "").Width(20).Height(20).Radius(4).Children(func() {
				ui.Icon(c, icon(iconName)).FontSize(13)
			}).Tooltip(a.t("starred")).Clicked() {
				a.toggleFavorite(e)
				c.Invalidate()
			}
		}
	})
}

// highlightedName draws a name with the matching query marked.
func (a *App) highlightedName(c *ui.Context, p Palette, name, query string) {
	ranges, ok := fsx.MatchRanges(name, query)
	if !ok || len(ranges) == 0 {
		ui.Text(c, name).SingleLine().Grow(1)
		return
	}
	spans := make([]ui.Span, 0, len(ranges)*2+1)
	at := 0
	for _, rg := range ranges {
		start, end := rg[0], rg[1]
		if start < at {
			start = at
		}
		if end > len(name) {
			end = len(name)
		}
		if start > at {
			spans = append(spans, ui.Span{Text: name[at:start]})
		}
		spans = append(spans, ui.Span{Text: name[start:end], Background: fade(p.Accent, 0.38), Radius: 2})
		at = end
	}
	if at < len(name) {
		spans = append(spans, ui.Span{Text: name[at:]})
	}
	ui.RichText(c, spans...).SingleLine().Grow(1)
}

// dateCell builds the modified date column.
func (a *App) dateCell(c *ui.Context, p Palette, pane *Pane, row int) {
	r := pane.Rows[row]
	cell := ui.Row(c).FillWidth().Height(p.RowH).Padding(3, 8).Clip()
	if a.cfg.View.Zebra && row%2 == 1 {
		cell.Background(p.Surface)
	}
	cell.Children(func() {
		ui.Text(c, a.smartDate(r.E.ModTime)).FontSize(11.5).TextColor(p.TextMuted).SingleLine()
	})
}

// typeCell builds the type column with a colored extension label.
func (a *App) typeCell(c *ui.Context, p Palette, pane *Pane, row int) {
	r := pane.Rows[row]
	e := r.E
	cell := ui.Row(c).FillWidth().Height(p.RowH).Gap(6).Padding(3, 8).Clip()
	if a.cfg.View.Zebra && row%2 == 1 {
		cell.Background(p.Surface)
	}
	cell.Children(func() {
		if e.IsDir {
			ui.Text(c, a.t("typeFolder")).FontSize(11.5).TextColor(p.TextMuted).SingleLine()
			return
		}
		if a.cfg.View.ExtensionColors && e.Ext != "" {
			col := a.extColor(e.Kind, p)
			ui.Text(c, strings.ToUpper(e.Ext)).FontSize(9.5).Bold().
				TextColor(col).Background(fade(col, 0.16)).Padding(1, 4).Radius(3)
		}
		ui.Text(c, a.typeName(e.Kind)).FontSize(11.5).TextColor(p.TextMuted).SingleLine()
	})
}

// extColor picks the label color of a file kind.
func (a *App) extColor(k fsx.Kind, p Palette) ui.Color {
	switch k {
	case fsx.KindImage:
		return p.TagGreen
	case fsx.KindVideo:
		return p.TagPurple
	case fsx.KindAudio:
		return ui.Hex("#c2477c")
	case fsx.KindArchive:
		return p.TagOrange
	case fsx.KindPDF:
		return p.Danger
	case fsx.KindSheet:
		return p.TagGreen
	case fsx.KindSlides:
		return p.TagOrange
	case fsx.KindCode:
		return p.TagBlue
	case fsx.KindExecutable:
		return p.TagBlue
	case fsx.KindFont:
		return p.TagPurple
	case fsx.KindDocument:
		return p.TagBlue
	}
	return p.TextMuted
}

// sizeCell builds the size column with a proportion bar.
func (a *App) sizeCell(c *ui.Context, p Palette, pane *Pane, row int, maxSize int64) {
	r := pane.Rows[row]
	e := r.E
	cell := ui.Column(c).FillWidth().Height(p.RowH).Padding(3, 8).Justify(ui.Center).Gap(2).Clip()
	if a.cfg.View.Zebra && row%2 == 1 {
		cell.Background(p.Surface)
	}
	cell.Children(func() {
		if e.Kind == fsx.KindDisk {
			if d, ok := a.findDrive(pane, e.Path); ok {
				used := d.Total - d.Free
				ui.Text(c, fsx.FormatSize(d.Free)+" / "+fsx.FormatSize(d.Total)).FontSize(10.5).
					TextColor(p.TextMuted).TextAlign(ui.End)
				if a.cfg.View.SizeBars && d.Total > 0 {
					a.sizeBar(c, p, float32(used)/float32(d.Total), p.Accent)
				}
				return
			}
		}
		var size int64
		known := true
		if e.IsDir {
			size, known = pane.Sizes[e.Path]
			if !known {
				ui.Text(c, "—").FontSize(11).TextColor(p.TextFaint).TextAlign(ui.End)
				return
			}
		} else {
			size = e.Size
		}
		ui.Text(c, fsx.FormatSize(size)).FontSize(11).TextColor(p.TextMuted).TextAlign(ui.End)
		if a.cfg.View.SizeBars && maxSize > 0 {
			a.sizeBar(c, p, float32(size)/float32(maxSize), p.Accent)
		}
	})
}

// sizeBar draws a proportion bar under a size.
func (a *App) sizeBar(c *ui.Context, p Palette, frac float32, col ui.Color) {
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	ui.Box(c).FillWidth().Height(2).Radius(1).Background(p.Surface).Children(func() {
		ui.Box(c).WidthPercent(frac * 100).Height(2).Radius(1).Background(col)
	})
}

// tilesView shows the pane as a grid of tiles or icons.
func (a *App) tilesView(c *ui.Context, p Palette, pane *Pane, tiles bool) {
	pane.Grid.Key = func(i int) any { return rowKey(pane.Rows[i]) }
	pane.Grid.Label = func(i int) string { return rowLabel(pane.Rows[i]) }
	pane.Grid.Selected = &pane.Selected
	pane.Grid.Selection = &pane.Selection
	if pane.Loading && len(pane.Rows) == 0 {
		a.loadingState(c, p, pane)
		return
	}
	if len(pane.Rows) == 0 {
		a.emptyState(c, p, pane)
		return
	}
	minW, h := float32(108), float32(112)
	thumbH := float32(64)
	if tiles {
		minW, h, thumbH = 136, 154, 100
	}
	grid := ui.GridView(c, &pane.Grid, len(pane.Rows), minW, h, func(i int) {
		if i < 0 || i >= len(pane.Rows) {
			return
		}
		r := pane.Rows[i]
		e := r.E
		item := ui.Column(c).Padding(6).Gap(3).Radius(8).Clip()
		if pane.Selected == i {
			item.Background(fade(p.Accent, 0.16))
		}
		item.Children(func() {
			box := ui.Box(c).FillWidth().Height(thumbH).Radius(6).Background(p.Surface)
			box.Children(func() {
				if tiles && e.Kind == fsx.KindImage {
					if bmp := a.thumbFor(e); bmp != nil {
						ui.Image(c, bmp).Fill().Fit(ui.Contain).Radius(5)
						return
					}
				}
				iconSize := float32(32)
				if tiles {
					iconSize = 44
				}
				ui.Image(c, kindIcon(e.Kind)).Size(iconSize, iconSize)
			})
			if pane.Query != "" {
				a.highlightedName(c, p, e.Name, pane.Query)
			} else {
				ui.Text(c, e.Name).FontSize(11.5).SingleLine().TextColor(p.Text)
			}
			if tiles {
				meta := a.typeName(e.Kind)
				if !e.IsDir {
					meta = fsx.FormatSize(e.Size)
				} else if s, ok := pane.Sizes[e.Path]; ok {
					meta = fsx.FormatSize(s)
				}
				ui.Text(c, meta).FontSize(10).TextColor(p.TextFaint).SingleLine()
			}
			if tag := a.tagOf(e.Path); tag != "" {
				ui.Box(c).Size(5, 5).Radius(2.5).Background(a.tagColor(tag))
			}
			if pane.CompareOn {
				a.compareBadge(c, p, pane, e.Path)
			}
		})
		item.ContextMenu(a.contextMenu(pane, i))
		item.Drag(a.dragValueFor(pane, e, c.Modifiers()&ui.Shift != 0))
		item.DragDataFrom(func() transfer.Data { return a.dragDataFor(pane, e) })
	}).Grow(1)
	if grid.Changed() {
		a.activatePane(pane)
		if a.cfg.General.SingleClickOpen {
			a.activateRow(pane, pane.Selected)
		}
	}
	if grid.Submitted() {
		a.activateRow(pane, pane.Selected)
	}
}

// listView shows the pane as compact rows.
func (a *App) listView(c *ui.Context, p Palette, pane *Pane) {
	pane.List.Key = func(i int) any { return rowKey(pane.Rows[i]) }
	pane.List.Label = func(i int) string { return rowLabel(pane.Rows[i]) }
	pane.List.Header = func(i int) bool { return pane.Rows[i].IsHeader }
	pane.List.Selection = &pane.Selection
	pane.List.Selected = &pane.Selected
	pane.List.DragSelect = true
	pane.List.CellMenu = nil
	if pane.Loading && len(pane.Rows) == 0 {
		a.loadingState(c, p, pane)
		return
	}
	if len(pane.Rows) == 0 {
		a.emptyState(c, p, pane)
		return
	}
	list := ui.List(c, &pane.List, len(pane.Rows), func(i int) {
		if i < 0 || i >= len(pane.Rows) {
			return
		}
		r := pane.Rows[i]
		if r.IsHeader {
			a.groupHeader(c, p, r.Header)
			return
		}
		e := r.E
		zebra := a.cfg.View.Zebra && i%2 == 1
		rowEl := ui.Row(c).Height(p.RowH-4).Gap(6).Padding(2, 8).Clip()
		if zebra {
			rowEl.Background(p.Surface)
		}
		rowEl.Children(func() {
			ui.Image(c, kindIcon(e.Kind)).Size(16, 16)
			if pane.Query != "" {
				a.highlightedName(c, p, e.Name, pane.Query)
			} else {
				ui.Text(c, e.Name).FontSize(12).SingleLine().Grow(1).TextColor(p.Text)
			}
			if tag := a.tagOf(e.Path); tag != "" {
				ui.Box(c).Size(6, 6).Radius(3).Background(a.tagColor(tag))
			}
			if pane.CompareOn {
				a.compareBadge(c, p, pane, e.Path)
			}
			if !e.IsDir {
				ui.Text(c, fsx.FormatSize(e.Size)).FontSize(10.5).TextColor(p.TextFaint)
			}
			ui.Text(c, a.smartDate(e.ModTime)).FontSize(10.5).TextColor(p.TextFaint)
		})
		rowEl.ContextMenu(a.contextMenu(pane, i))
		rowEl.Drag(a.dragValueFor(pane, e, c.Modifiers()&ui.Shift != 0))
		rowEl.DragDataFrom(func() transfer.Data { return a.dragDataFor(pane, e) })
	}).Grow(1)
	if list.Changed() {
		a.activatePane(pane)
		if a.cfg.General.SingleClickOpen {
			a.activateRow(pane, pane.Selected)
		}
	}
	if list.Submitted() {
		a.activateRow(pane, pane.Selected)
	}
}

// contentView shows larger rows with metadata.
func (a *App) contentView(c *ui.Context, p Palette, pane *Pane) {
	pane.List.Key = func(i int) any { return rowKey(pane.Rows[i]) }
	pane.List.Label = func(i int) string { return rowLabel(pane.Rows[i]) }
	pane.List.Header = func(i int) bool { return pane.Rows[i].IsHeader }
	pane.List.Selection = &pane.Selection
	pane.List.Selected = &pane.Selected
	pane.List.DragSelect = true
	if pane.Loading && len(pane.Rows) == 0 {
		a.loadingState(c, p, pane)
		return
	}
	if len(pane.Rows) == 0 {
		a.emptyState(c, p, pane)
		return
	}
	list := ui.List(c, &pane.List, len(pane.Rows), func(i int) {
		if i < 0 || i >= len(pane.Rows) {
			return
		}
		r := pane.Rows[i]
		if r.IsHeader {
			a.groupHeader(c, p, r.Header)
			return
		}
		e := r.E
		rowEl := ui.Row(c).Height(44).Gap(10).Padding(4, 10).Clip()
		if a.cfg.View.Zebra && i%2 == 1 {
			rowEl.Background(p.Surface)
		}
		rowEl.Children(func() {
			ui.Image(c, kindIcon(e.Kind)).Size(28, 28)
			ui.Column(c).Grow(1).Gap(1).Justify(ui.Center).Children(func() {
				ui.Text(c, e.Name).FontSize(12).SingleLine().TextColor(p.Text)
				meta := a.typeName(e.Kind)
				if !e.IsDir {
					meta += " · " + fsx.FormatSize(e.Size)
				}
				meta += " · " + a.smartDate(e.ModTime)
				ui.Text(c, meta).FontSize(10.5).TextColor(p.TextMuted).SingleLine()
			})
			if tag := a.tagOf(e.Path); tag != "" {
				ui.Box(c).Size(6, 6).Radius(3).Background(a.tagColor(tag))
			}
		})
		rowEl.ContextMenu(a.contextMenu(pane, i))
		rowEl.Drag(a.dragValueFor(pane, e, c.Modifiers()&ui.Shift != 0))
		rowEl.DragDataFrom(func() transfer.Data { return a.dragDataFor(pane, e) })
	}).Grow(1)
	if list.Changed() {
		a.activatePane(pane)
		if a.cfg.General.SingleClickOpen {
			a.activateRow(pane, pane.Selected)
		}
	}
	if list.Submitted() {
		a.activateRow(pane, pane.Selected)
	}
}

// groupHeader draws a group's title row.
func (a *App) groupHeader(c *ui.Context, p Palette, title string) {
	ui.Row(c).Height(26).Padding(4, 8).Background(p.Panel).Children(func() {
		ui.Text(c, title).FontSize(11).Bold().TextColor(p.TextMuted)
	})
}

// thumbFor returns a decoded thumbnail, decoding it in the background first.
func (a *App) thumbFor(e fsx.Entry) *ui.Bitmap {
	if a.Win == nil || !isImageExt(e.Ext) || e.Size > 20*1024*1024 {
		return nil
	}
	if bmp, ok := a.thumb.m[e.Path]; ok {
		return bmp
	}
	if a.thumb.failed[e.Path] || a.thumb.loading[e.Path] {
		return nil
	}
	if len(a.thumb.m) >= 400 {
		a.thumb.m = map[string]*ui.Bitmap{}
	}
	a.thumb.loading[e.Path] = true
	path := e.Path
	go func() {
		data, err := os.ReadFile(path)
		if err != nil {
			a.Win.Update(func() {
				delete(a.thumb.loading, path)
				a.thumb.failed[path] = true
			})
			return
		}
		a.Win.Update(func() {
			delete(a.thumb.loading, path)
			bmp, err := ui.DecodeBitmap(data)
			if err != nil {
				a.thumb.failed[path] = true
				return
			}
			a.thumb.m[path] = bmp
		})
	}()
	return nil
}

func isImageExt(ext string) bool {
	switch strings.ToLower(ext) {
	case "png", "jpg", "jpeg", "jpe", "gif", "bmp", "webp":
		return true
	}
	return false
}

// imageSize returns a cached image size, decoding the header in the
// background when it is not known yet.
func (a *App) imageSize(e fsx.Entry) (w, h int, ok bool) {
	if a.Win == nil || e.IsDir {
		return 0, 0, false
	}
	if d, known := a.imgDims[e.Path]; known {
		if time.Since(d.at) < 10*time.Minute {
			return d.w, d.h, d.ok
		}
	}
	path := e.Path
	a.imgDims[path] = dimEntry{at: time.Now()}
	go func() {
		w, h, ok := fsx.ImageDimensions(path)
		a.Win.Update(func() {
			a.imgDims[path] = dimEntry{w: w, h: h, ok: ok, at: time.Now()}
		})
	}()
	return 0, 0, false
}

// openWithDefault opens a path.
func openWithDefault(path string) error { return platform.Open(path) }

// compareBadge shows how an item differs from the other pane.
func (a *App) compareBadge(c *ui.Context, p Palette, pane *Pane, path string) {
	mark, ok := pane.Compare[path]
	if !ok {
		return
	}
	col := a.compareColor(mark.Status, p)
	ui.Text(c, a.compareLabel(mark.Status)).FontSize(9).TextColor(col).
		Background(fade(col, 0.15)).Padding(1, 4).Radius(3).SingleLine()
}

package app

import (
	"path/filepath"

	"pulse/internal/fsx"
	"pulse/internal/platform"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// handleShortcuts handles the window's keyboard commands.
func (a *App) handleShortcuts(c *ui.Context) {
	pane := a.activePane()

	if a.addressEdit && c.Shortcut(0, ui.KeyEscape) {
		a.addressEdit = false
	}
	if c.Shortcut(ui.Cmd, ui.KeyT) {
		a.newTabHere()
	}
	if c.Shortcut(ui.Cmd, ui.KeyW) {
		a.closeTab(a.tab)
	}
	if c.Shortcut(ui.Cmd, ui.KeyL) {
		a.focusAddress()
	}
	if c.Shortcut(ui.Cmd, ui.KeyF) {
		a.searchHandle.Focus()
	}
	if c.Shortcut(ui.Cmd, ui.KeyK) {
		a.openSearchOverlay()
	}
	if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyF) {
		a.advanced.Open = true
	}
	if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyN) && pane != nil {
		a.beginNewFolder(pane)
	}
	if c.Shortcut(ui.Cmd, ui.KeyB) {
		a.sidebarOn = !a.sidebarOn
		a.cfg.View.SidebarOn = a.sidebarOn
		a.saveSessionSoon()
	}
	if c.Shortcut(ui.Cmd, ui.KeyD) {
		a.detailsOn = !a.detailsOn
		a.cfg.View.ShowDetails = a.detailsOn
		a.saveSessionSoon()
	}
	layoutKeys := []ui.Key{ui.Key1, ui.Key2, ui.Key3, ui.Key4, ui.Key5}
	for i, k := range layoutKeys {
		if c.Shortcut(ui.Cmd, k) {
			a.setLayout(layoutMode(i))
		}
	}
	viewKeys := []ui.Key{ui.Key1, ui.Key2, ui.Key3, ui.Key4, ui.Key5}
	views := []viewMode{viewDetails, viewTiles, viewIcons, viewList, viewContent}
	for i, k := range viewKeys {
		if c.Shortcut(ui.Alt, k) && pane != nil {
			a.setPaneView(pane, views[i])
		}
	}
	if c.Shortcut(ui.Alt, ui.KeyLeft) && pane != nil {
		a.goBack(pane)
	}
	if c.Shortcut(ui.Alt, ui.KeyRight) && pane != nil {
		a.goForward(pane)
	}
	if c.Shortcut(ui.Alt, ui.KeyUp) && pane != nil {
		a.goUp(pane)
	}
	if c.Shortcut(ui.Cmd, ui.KeyPageDown) {
		a.selectTab((a.tab + 1) % len(a.tabs))
	}
	if c.Shortcut(ui.Cmd, ui.KeyPageUp) {
		a.selectTab((a.tab - 1 + len(a.tabs)) % len(a.tabs))
	}
	if pane != nil {
		if c.Shortcut(0, ui.KeyF2) && pane.View != viewDetails {
			if entries := a.selectedEntries(pane); len(entries) > 1 {
				a.beginBatchRename(pane)
			} else if len(entries) == 1 {
				a.beginRename(pane, entries[0])
			}
		}
		if c.Shortcut(0, ui.KeyF5) {
			a.refreshPane(pane)
		}
		if c.Shortcut(0, ui.KeyDelete) {
			a.deleteEntries(pane, a.selectedEntries(pane), false)
		}
		if c.Shortcut(ui.Shift, ui.KeyDelete) {
			a.deleteEntries(pane, a.selectedEntries(pane), true)
		}
		if c.Shortcut(ui.Cmd, ui.KeyA) {
			a.selectAllRows(pane)
		}
		if c.Shortcut(ui.Cmd, ui.KeyC) {
			a.stageEntries(pane, false)
		}
		if c.Shortcut(ui.Cmd, ui.KeyX) {
			a.stageEntries(pane, true)
		}
		if c.Shortcut(ui.Cmd, ui.KeyV) {
			a.pasteStash()
		}
		if c.Shortcut(ui.Cmd, ui.KeyZ) {
			if n := len(a.undo); n > 0 {
				u := a.undo[n-1]
				a.undo = a.undo[:n-1]
				a.runUndo(u)
			}
		}
		if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyC) {
			if e := a.selectedEntry(pane); e != nil {
				mygo.Clipboard.WriteText(e.Path)
			}
		}
		if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyV) {
			a.pasteFromSystemClipboard(pane)
		}
	}
	if c.Shortcut(0, ui.KeyEscape) {
		if a.search.Searching {
			a.stopSearch()
		}
	}
	if a.ops.Running && c.Shortcut(0, ui.KeyEscape) {
		if a.ops.Cancel != nil {
			a.ops.Cancel()
		}
	}
}

// invertSelection chooses every row that was not chosen.
func (a *App) invertSelection(pane *Pane) {
	for _, r := range pane.Rows {
		if r.IsHeader {
			continue
		}
		if pane.Selection.Has(r.E.Path) {
			pane.Selection.Remove(r.E.Path)
		} else {
			pane.Selection.Add(r.E.Path)
		}
	}
	pane.Selected = -1
}

// selectAllRows chooses every row of a pane.
func (a *App) selectAllRows(pane *Pane) {
	for _, r := range pane.Rows {
		if !r.IsHeader {
			pane.Selection.Add(r.E.Path)
		}
	}
	if pane.Selected < 0 {
		for i, r := range pane.Rows {
			if !r.IsHeader {
				pane.Selected = i
				break
			}
		}
	}
}

// pasteFromSystemClipboard copies paths found in the text clipboard into the
// pane.
func (a *App) pasteFromSystemClipboard(pane *Pane) {
	if pane == nil || pane.Virtual != virtualNone {
		return
	}
	text := mygo.Clipboard.ReadText()
	if text == "" {
		return
	}
	var paths []string
	for _, line := range splitTextLines(text) {
		line = trimQuotes(line)
		if line == "" {
			continue
		}
		if !filepath.IsAbs(line) {
			line = filepath.Join(pane.Path, line)
		}
		if platform.Exists(line) {
			paths = append(paths, line)
		}
	}
	if len(paths) == 0 {
		return
	}
	a.startTransfer(paths, pane.Path, false)
}

func trimQuotes(s string) string {
	s = trimSpace(s)
	if len(s) >= 2 && (s[0] == '"' && s[len(s)-1] == '"' || s[0] == '\'' && s[len(s)-1] == '\'') {
		return s[1 : len(s)-1]
	}
	return s
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\r' || s[start] == '\n') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r' || s[end-1] == '\n') {
		end--
	}
	return s[start:end]
}

// openSelectedEntry opens the chosen entry.
func (a *App) openSelectedEntry() {
	if e := a.selectedEntry(a.activePane()); e != nil {
		a.activateEntry(a.activePane(), *e)
	}
}

// renameSelection renames one entry, or several in the batch dialog.
func (a *App) renameSelection() {
	p := a.activePane()
	if p == nil {
		return
	}
	entries := a.selectedEntries(p)
	if len(entries) > 1 {
		a.beginBatchRename(p)
	} else if len(entries) == 1 {
		a.beginRename(p, entries[0])
	}
}

// selectedEntriesForAction returns the entries an action applies to.
func (a *App) selectedEntriesForAction() []fsx.Entry {
	return a.selectedEntries(a.activePane())
}

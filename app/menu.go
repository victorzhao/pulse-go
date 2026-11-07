package app

import (
	"path/filepath"
	"runtime"

	"pulse/internal/platform"

	"github.com/egoist/mygo"
)

// SetupMenu builds the window's menu bar in the current language.
func (a *App) SetupMenu() {
	if a.Win == nil {
		return
	}
	edit := &mygo.MenuItem{Role: mygo.RoleEditMenu}
	view := &mygo.MenuItem{Label: a.t("view"), Submenu: []*mygo.MenuItem{
		{Label: a.t("showSidebar"), Accelerator: "CmdOrCtrl+B", Type: mygo.MenuItemCheckbox, Checked: a.sidebarOn, Click: func(item *mygo.MenuItem, w *mygo.Window) {
			a.sidebarOn = item.IsChecked()
			a.cfg.View.SidebarOn = a.sidebarOn
			a.saveSessionSoon()
		}},
		{Label: a.t("showDetails"), Accelerator: "CmdOrCtrl+D", Type: mygo.MenuItemCheckbox, Checked: a.detailsOn, Click: func(item *mygo.MenuItem, w *mygo.Window) {
			a.detailsOn = item.IsChecked()
			a.cfg.View.ShowDetails = a.detailsOn
			a.saveSessionSoon()
		}},
		mygo.Separator(),
		{Label: a.t("focusLeftPane"), Accelerator: "CmdOrCtrl+Shift+Left", Click: func(*mygo.MenuItem, *mygo.Window) { a.focusSide(0) }},
		{Label: a.t("focusRightPane"), Accelerator: "CmdOrCtrl+Shift+Right", Click: func(*mygo.MenuItem, *mygo.Window) { a.focusSide(1) }},
		mygo.Separator(),
		{Label: a.t("view"), Submenu: []*mygo.MenuItem{
			{Label: a.t("viewDetails"), Accelerator: "Alt+1", Click: func(*mygo.MenuItem, *mygo.Window) { a.setView(viewDetails) }},
			{Label: a.t("viewTiles"), Accelerator: "Alt+2", Click: func(*mygo.MenuItem, *mygo.Window) { a.setView(viewTiles) }},
			{Label: a.t("viewIcons"), Accelerator: "Alt+3", Click: func(*mygo.MenuItem, *mygo.Window) { a.setView(viewIcons) }},
			{Label: a.t("viewList"), Accelerator: "Alt+4", Click: func(*mygo.MenuItem, *mygo.Window) { a.setView(viewList) }},
			{Label: a.t("viewContent"), Accelerator: "Alt+5", Click: func(*mygo.MenuItem, *mygo.Window) { a.setView(viewContent) }},
		}},
		mygo.Separator(),
		{Label: a.t("showHidden"), Type: mygo.MenuItemCheckbox, Checked: a.cfg.View.ShowHidden, Click: func(item *mygo.MenuItem, w *mygo.Window) {
			a.cfg.View.ShowHidden = item.IsChecked()
			if p := a.activePane(); p != nil {
				a.applyPane(p)
			}
			a.saveSessionSoon()
		}},
		{Label: a.t("refresh"), Accelerator: "F5", Click: func(*mygo.MenuItem, *mygo.Window) { a.refreshPane(a.activePane()) }},
	}}
	goMenu := &mygo.MenuItem{Label: a.t("go"), Submenu: []*mygo.MenuItem{
		{Label: a.t("back"), Accelerator: "Alt+Left", Click: func(*mygo.MenuItem, *mygo.Window) { a.goBack(a.activePane()) }},
		{Label: a.t("forward"), Accelerator: "Alt+Right", Click: func(*mygo.MenuItem, *mygo.Window) { a.goForward(a.activePane()) }},
		{Label: a.t("up"), Accelerator: "Alt+Up", Click: func(*mygo.MenuItem, *mygo.Window) { a.goUp(a.activePane()) }},
		mygo.Separator(),
		{Label: a.t("home"), Click: func(*mygo.MenuItem, *mygo.Window) { a.navigateActive(a.homeDir()) }},
		{Label: a.t("computer"), Click: func(*mygo.MenuItem, *mygo.Window) { a.openVirtualPane(virtualComputer, "") }},
		{Label: a.t("recycleBin"), Click: func(*mygo.MenuItem, *mygo.Window) { a.openVirtualPane(virtualTrash, "") }},
		{Label: a.t("tags"), Submenu: a.tagMenuItems()},
	}}
	tools := &mygo.MenuItem{Label: a.t("tools"), Submenu: []*mygo.MenuItem{
		{Label: a.t("settings"), Accelerator: "CmdOrCtrl+,", Click: func(*mygo.MenuItem, *mygo.Window) { a.openSettingsTab(0) }},
		{Label: a.t("globalSearch"), Accelerator: "CmdOrCtrl+K", Click: func(*mygo.MenuItem, *mygo.Window) { a.openSearchOverlay() }},
		{Label: a.t("advancedSearch"), Accelerator: "CmdOrCtrl+Shift+F", Click: func(*mygo.MenuItem, *mygo.Window) { a.advanced.Open = true }},
		mygo.Separator(),
		{Label: a.t("terminal"), Click: func(*mygo.MenuItem, *mygo.Window) {
			if p := a.activePane(); p != nil {
				if err := a.openTerminalAt(p.Path); err != nil {
					a.toast = a.t("terminalFailed")
				}
			}
		}},
		{Label: a.t("compareTitle"), Click: func(*mygo.MenuItem, *mygo.Window) { a.startFolderCompare() }},
	}}
	help := &mygo.MenuItem{Label: a.t("help"), Submenu: []*mygo.MenuItem{
		{Label: a.t("keyboard"), Click: func(*mygo.MenuItem, *mygo.Window) { a.openSettingsTab(4) }},
		{Label: a.t("about"), Click: func(*mygo.MenuItem, *mygo.Window) { a.openSettingsTab(4) }},
	}}
	file := &mygo.MenuItem{Label: a.t("file"), Submenu: []*mygo.MenuItem{
		{Label: a.t("newTab"), Accelerator: "CmdOrCtrl+T", Click: func(*mygo.MenuItem, *mygo.Window) { a.newTabHere() }},
		{Label: a.t("newFolder"), Accelerator: "CmdOrCtrl+Shift+N", Click: func(*mygo.MenuItem, *mygo.Window) { a.beginNewFolder(a.activePane()) }},
		{Label: a.t("newFile"), Click: func(*mygo.MenuItem, *mygo.Window) { a.beginNewFile(a.activePane()) }},
		{Label: a.t("closeTab"), Accelerator: "CmdOrCtrl+W", Click: func(*mygo.MenuItem, *mygo.Window) { a.closeActiveSideTab() }},
		mygo.Separator(),
		{Label: a.t("open"), Accelerator: "CmdOrCtrl+O", Click: func(*mygo.MenuItem, *mygo.Window) { a.openSelectedEntry() }},
		{Label: a.t("openNewTab"), Click: func(*mygo.MenuItem, *mygo.Window) { a.openSelectedInNewTab() }},
		mygo.Separator(),
		{Label: a.t("rename"), Click: func(*mygo.MenuItem, *mygo.Window) { a.renameSelection() }},
		{Label: a.t("delete"), Click: func(*mygo.MenuItem, *mygo.Window) {
			a.deleteEntries(a.activePane(), a.selectedEntriesForAction(), false)
		}},
		{Label: a.t("properties"), Click: func(*mygo.MenuItem, *mygo.Window) { a.propertiesForSelection() }},
		mygo.Separator(),
		{Role: mygo.RoleQuit},
	}}
	items := []*mygo.MenuItem{}
	if a.isMac() {
		items = append(items, &mygo.MenuItem{Role: mygo.RoleAppMenu})
	}
	items = append(items, file, edit, view, goMenu, tools, help, &mygo.MenuItem{Role: mygo.RoleWindowMenu})
	a.Win.SetMenu(mygo.NewMenu(items))
}

func (a *App) isMac() bool {
	return runtime.GOOS == "darwin"
}

// setView changes the active pane's view.
func (a *App) setView(v viewMode) {
	if p := a.activePane(); p != nil {
		a.setPaneView(p, v)
	}
}

// openVirtualPane shows a made-up folder in the active pane.
func (a *App) openVirtualPane(kind virtualKind, tag string) {
	if p := a.activePane(); p != nil {
		a.openVirtual(p, kind, tag)
	}
}

// openSettingsTab opens settings on a tab.
func (a *App) openSettingsTab(tab int) {
	a.settings.Open = true
	a.settings.Tab = tab
}

// tagMenuItems returns one item per tag.
func (a *App) tagMenuItems() []*mygo.MenuItem {
	items := make([]*mygo.MenuItem, 0, len(a.cfg.Tags))
	for _, tag := range a.cfg.Tags {
		tag := tag
		items = append(items, &mygo.MenuItem{Label: tag.Name, Click: func(*mygo.MenuItem, *mygo.Window) {
			if p := a.activePane(); p != nil {
				a.openVirtual(p, virtualTag, tag.ID)
			}
		}})
	}
	return items
}

// openTerminalAt opens a terminal in a folder.
func (a *App) openTerminalAt(path string) error {
	return platform.OpenTerminal(path)
}

// openSelectedInNewTab opens the selected folder in a new tab.
func (a *App) openSelectedInNewTab() {
	p := a.activePane()
	if p == nil {
		return
	}
	if e := a.selectedEntry(p); e != nil {
		if e.IsDir {
			a.newTab(e.Path)
		} else {
			a.newTab(filepath.Dir(e.Path))
		}
	}
}

// propertiesForSelection opens properties for the chosen entries.
func (a *App) propertiesForSelection() {
	if entries := a.selectedEntriesForAction(); len(entries) > 0 {
		a.showProperties(entries)
	}
}

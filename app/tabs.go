package app

import "github.com/egoist/mygo/ui"

// sideTabStrip draws one side's row of folder tabs.
func (a *App) sideTabStrip(c *ui.Context, p Palette, idx int) {
	side := a.sideAt(idx)
	ui.ScrollBoth(c).Height(32).Padding(2, 4).Background(p.Panel).BorderWidth(0, 0, 1, 0).BorderColor(p.BorderSoft).Children(func() {
		ui.Row(c).Gap(3).AlignItems(ui.Center).Children(func() {
			for i, pane := range side.Panes {
				i := i
				pane := pane
				chosen := side.Active == i
				active := chosen && a.activeSide == idx
				tab := ui.Row(c).Height(26).Padding(0, 6).Gap(5).Radius(5).Clip()
				if active {
					tab.Background(fade(p.Accent, 0.18))
					tab.BorderWidth(0, 0, 2, 0).BorderColor(p.Accent)
				} else if chosen {
					tab.Background(p.Surface)
				} else if tab.Hovered() {
					tab.Background(p.SurfaceHi)
				}
				tab.Children(func() {
					ui.Icon(c, icon("folder")).FontSize(12).TextColor(p.TextMuted)
					name := ui.Text(c, a.tabTitle(pane)).FontSize(11).SingleLine().MaxWidth(150)
					if chosen {
						name.TextColor(p.Text)
					} else {
						name.TextColor(p.TextMuted)
					}
					if flatButton(c, p, 18, 18).Children(func() {
						ui.Icon(c, icon("x")).FontSize(9).TextColor(p.TextFaint)
					}).Tooltip(a.t("closeTab")).Label(a.t("closeTab")).Clicked() {
						a.closeSideTab(idx, i)
					}
				})
				tab.OnClick(func() {
					a.selectSideTab(idx, i)
				})
				tab.ContextMenu(func(m *ui.Menu) {
					if m.Item(a.t("closeTab")).Shortcut(ui.Cmd, ui.KeyW).Chosen() {
						a.closeSideTab(idx, i)
					}
					if m.Item(a.t("closeOthers")).Chosen() {
						a.closeSideOtherTabs(idx, i)
					}
					if m.Item(a.t("closeRight")).Chosen() {
						a.closeSideTabsRight(idx, i)
					}
					if m.Item(a.t("closeLeft")).Chosen() {
						a.closeSideTabsLeft(idx, i)
					}
				})
				tab.HandleInput(func(ev ui.InputEvent) bool {
					if ev.Button == 2 {
						if ev.Kind == ui.InputPointerUp {
							a.closeSideTab(idx, i)
						}
						return true
					}
					return false
				})
				tab.Drag(i)
				if from, ok := ui.Drop[int](tab); ok && from != i {
					a.moveSideTab(idx, from, i)
				}
			}
			path := a.defaultPath()
			if p := a.sidePane(idx); p != nil && p.Virtual == virtualNone {
				path = p.Path
			}
			if flatButton(c, p, 24, 24).Children(func() {
				ui.Icon(c, icon("plus")).FontSize(13).TextColor(p.TextMuted)
			}).Tooltip(a.t("newTab")).Label(a.t("newTab")).Clicked() {
				a.newSideTab(idx, path)
			}
		})
	})
}

// sidePane returns a side's active tab.
func (a *App) sidePane(idx int) *Pane {
	t := a.sideAt(idx)
	if t == nil || len(t.Panes) == 0 {
		return nil
	}
	if t.Active < 0 || t.Active >= len(t.Panes) {
		t.Active = 0
	}
	return t.Panes[t.Active]
}

// newSideTab adds a tab to a side and shows it.
func (a *App) newSideTab(idx int, path string) {
	side := a.sideAt(idx)
	p := a.newPane(path)
	side.Panes = append(side.Panes, p)
	side.Active = len(side.Panes) - 1
	a.activeSide = idx
	a.loadPane(p)
	a.refreshWatchers()
	a.saveSessionSoon()
}

// closeSideTab closes tab i of a side. The last tab resets to the default
// location instead of leaving the side empty.
func (a *App) closeSideTab(idx int, i int) {
	side := a.sideAt(idx)
	if i < 0 || i >= len(side.Panes) {
		return
	}
	if len(side.Panes) == 1 {
		a.stopPaneWatcher(side.Panes[0])
		p := a.newPane(a.defaultPath())
		side.Panes[0] = p
		side.Active = 0
		a.activeSide = idx
		a.loadPane(p)
		a.refreshWatchers()
		a.saveSessionSoon()
		return
	}
	a.stopPaneWatcher(side.Panes[i])
	side.Panes = append(side.Panes[:i], side.Panes[i+1:]...)
	if side.Active >= len(side.Panes) {
		side.Active = len(side.Panes) - 1
	}
	if side.Active < 0 {
		side.Active = 0
	}
	a.activeSide = idx
	a.refreshWatchers()
	a.saveSessionSoon()
}

// selectSideTab shows tab i of a side.
func (a *App) selectSideTab(idx int, i int) {
	side := a.sideAt(idx)
	if i < 0 || i >= len(side.Panes) {
		return
	}
	side.Active = i
	a.activeSide = idx
	a.refreshWatchers()
	a.saveSessionSoon()
}

// closeSideOtherTabs keeps only tab i of a side.
func (a *App) closeSideOtherTabs(idx int, i int) {
	side := a.sideAt(idx)
	if i < 0 || i >= len(side.Panes) {
		return
	}
	for j, p := range side.Panes {
		if j != i {
			a.stopPaneWatcher(p)
		}
	}
	side.Panes = []*Pane{side.Panes[i]}
	side.Active = 0
	a.activeSide = idx
	a.refreshWatchers()
	a.saveSessionSoon()
}

// closeSideTabsRight closes the tabs after i of a side.
func (a *App) closeSideTabsRight(idx int, i int) {
	side := a.sideAt(idx)
	if i < 0 || i >= len(side.Panes) || len(side.Panes) <= 1 {
		return
	}
	for _, p := range side.Panes[i+1:] {
		a.stopPaneWatcher(p)
	}
	side.Panes = side.Panes[:i+1]
	if side.Active > i {
		side.Active = i
	}
	a.activeSide = idx
	a.refreshWatchers()
	a.saveSessionSoon()
}

// closeSideTabsLeft closes the tabs before i of a side.
func (a *App) closeSideTabsLeft(idx int, i int) {
	side := a.sideAt(idx)
	if i <= 0 || i >= len(side.Panes) || len(side.Panes) <= 1 {
		return
	}
	for _, p := range side.Panes[:i] {
		a.stopPaneWatcher(p)
	}
	side.Panes = side.Panes[i:]
	side.Active -= i
	if side.Active < 0 {
		side.Active = 0
	}
	a.activeSide = idx
	a.refreshWatchers()
	a.saveSessionSoon()
}

// moveSideTab moves a tab before another within a side.
func (a *App) moveSideTab(idx int, from, to int) {
	side := a.sideAt(idx)
	if from < 0 || from >= len(side.Panes) || to < 0 || to >= len(side.Panes) || from == to {
		return
	}
	t := side.Panes[from]
	side.Panes = append(side.Panes[:from], side.Panes[from+1:]...)
	if to > len(side.Panes) {
		to = len(side.Panes)
	}
	side.Panes = append(side.Panes[:to], append([]*Pane{t}, side.Panes[to:]...)...)
	for i, p := range side.Panes {
		if p == t {
			side.Active = i
			break
		}
	}
	a.activeSide = idx
	a.refreshWatchers()
	a.saveSessionSoon()
}

// closeActiveSideTab closes the active tab of the active side.
func (a *App) closeActiveSideTab() {
	t := a.activeTab()
	if t == nil {
		return
	}
	a.closeSideTab(a.activeSide, t.Active)
}

// stopPaneWatcher stops one pane's folder watcher.
func (a *App) stopPaneWatcher(p *Pane) {
	if p != nil && p.watchCancel != nil {
		p.watchCancel()
		p.watchCancel = nil
	}
}

// stopPaneWatchers stops the watchers of every tab in a side.
func (a *App) stopPaneWatchers(t *Tab) {
	if t == nil {
		return
	}
	for _, p := range t.Panes {
		a.stopPaneWatcher(p)
	}
}

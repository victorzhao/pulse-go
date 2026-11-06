package app

import (
	"github.com/egoist/mygo/ui"
)

// tabStrip draws the row of tabs in the title bar.
func (a *App) tabStrip(c *ui.Context, p Palette) {
	ui.ScrollBoth(c).Grow(1).Height(38).Clip().Children(func() {
		ui.Row(c).Gap(3).Padding(2, 4).AlignItems(ui.Center).Children(func() {
			for i, t := range a.tabs {
				i := i
				t := t
				active := i == a.tab
				tab := ui.Row(c).Height(30).Padding(0, 8).Gap(6).Radius(6).Clip()
				if active {
					tab.Background(p.Surface)
					tab.BorderWidth(0, 0, 2, 0).BorderColor(p.Accent)
				} else if tab.Hovered() {
					tab.Background(fade(p.SurfaceHi, 0.7))
				}
				tab.Children(func() {
					ui.Icon(c, icon("folder")).FontSize(13).TextColor(p.TextMuted)
					name := ui.Text(c, t.Title).FontSize(11.5).SingleLine().MaxWidth(150)
					if active {
						name.TextColor(p.Text)
					} else {
						name.TextColor(p.TextMuted)
					}
					if flatButton(c, p, 18, 18).Children(func() {
						ui.Icon(c, icon("x")).FontSize(10).TextColor(p.TextFaint)
					}).Tooltip(a.t("closeTab")).Label(a.t("closeTab")).Clicked() {
						a.closeTab(i)
					}
				})
				tab.OnClick(func() {
					a.selectTab(i)
				})
				tab.ContextMenu(func(m *ui.Menu) {
					if m.Item(a.t("closeTab")).Shortcut(ui.Cmd, ui.KeyW).Chosen() {
						a.closeTab(i)
					}
					if m.Item(a.t("closeOthers")).Chosen() {
						a.closeOtherTabs(i)
					}
					if m.Item(a.t("closeRight")).Chosen() {
						a.closeTabsRight(i)
					}
					if m.Item(a.t("closeLeft")).Chosen() {
						a.closeTabsLeft(i)
					}
				})
				tab.HandleInput(func(ev ui.InputEvent) bool {
					if ev.Button == 2 {
						if ev.Kind == ui.InputPointerUp {
							a.closeTab(i)
						}
						return true
					}
					return false
				})
				tab.Drag(i)
				if from, ok := ui.Drop[int](tab); ok && from != i {
					a.moveTab(from, i)
				}
			}
		})
	})
}

// verticalTabList shows the tabs in the sidebar.
func (a *App) verticalTabList(c *ui.Context, p Palette) {
	h := float32(len(a.tabs)*32 + 40)
	if h > 260 {
		h = 260
	}
	ui.Scroll(c).Height(h).Padding(4).Children(func() {
		ui.Column(c).Gap(2).Children(func() {
			for i, t := range a.tabs {
				i := i
				t := t
				active := i == a.tab
				row := ui.Row(c).Height(30).Padding(3, 6).Gap(6).Radius(6)
				if active {
					row.Background(fade(p.Accent, 0.18))
					row.BorderWidth(0, 0, 0, 2).BorderColor(p.Accent)
				} else if row.Hovered() {
					row.Background(p.SurfaceHi)
				}
				row.Children(func() {
					ui.Icon(c, icon("folder")).FontSize(13).TextColor(p.TextMuted)
					ui.Text(c, t.Title).FontSize(11.5).SingleLine().Grow(1).TextColor(p.Text)
					if flatButton(c, p, 18, 18).Children(func() {
						ui.Icon(c, icon("x")).FontSize(10).TextColor(p.TextFaint)
					}).Tooltip(a.t("closeTab")).Label(a.t("closeTab")).Clicked() {
						a.closeTab(i)
					}
				})
				row.OnClick(func() {
					a.selectTab(i)
				})
				row.ContextMenu(func(m *ui.Menu) {
					if m.Item(a.t("closeTab")).Chosen() {
						a.closeTab(i)
					}
					if m.Item(a.t("closeOthers")).Chosen() {
						a.closeOtherTabs(i)
					}
				})
				row.HandleInput(func(ev ui.InputEvent) bool {
					if ev.Button == 2 {
						if ev.Kind == ui.InputPointerUp {
							a.closeTab(i)
						}
						return true
					}
					return false
				})
				row.Drag(i)
				if from, ok := ui.Drop[int](row); ok && from != i {
					a.moveTab(from, i)
				}
			}
			add := ui.Button(c, a.t("newTab")).Height(26).Radius(5).FontSize(10.5).
				Background(ui.Transparent).Border(0, ui.Transparent).TextColor(p.TextMuted)
			if add.Hovered() {
				add.Background(p.SurfaceHi)
			}
			if add.Clicked() {
				a.newTabHere()
			}
		})
	})
	ui.Divider(c).Padding(4, 0)
}

// closeOtherTabs keeps only tab i.
func (a *App) closeOtherTabs(i int) {
	if i < 0 || i >= len(a.tabs) {
		return
	}
	for _, tab := range a.tabs {
		if tab != a.tabs[i] {
			a.stopPaneWatchers(tab)
		}
	}
	t := a.tabs[i]
	a.tabs = []*Tab{t}
	a.tab = 0
	a.refreshWatchers()
	a.saveSessionSoon()
}

// closeTabsRight closes the tabs after i.
func (a *App) closeTabsRight(i int) {
	if i < 0 || i >= len(a.tabs) || len(a.tabs) <= 1 {
		return
	}
	for _, tab := range a.tabs[i+1:] {
		a.stopPaneWatchers(tab)
	}
	a.tabs = a.tabs[:i+1]
	if a.tab > i {
		a.tab = i
	}
	a.refreshWatchers()
	a.saveSessionSoon()
}

// closeTabsLeft closes the tabs before i.
func (a *App) closeTabsLeft(i int) {
	if i <= 0 || i >= len(a.tabs) || len(a.tabs) <= 1 {
		return
	}
	for _, tab := range a.tabs[:i] {
		a.stopPaneWatchers(tab)
	}
	a.tabs = a.tabs[i:]
	a.tab -= i
	if a.tab < 0 {
		a.tab = 0
	}
	a.saveSessionSoon()
}

// stopPaneWatchers stops the folder watchers of a tab.
func (a *App) stopPaneWatchers(t *Tab) {
	if t == nil {
		return
	}
	for _, p := range t.Panes {
		if p.watchCancel != nil {
			p.watchCancel()
			p.watchCancel = nil
		}
	}
}

// moveTab moves tab from before tab to.
func (a *App) moveTab(from, to int) {
	if from < 0 || from >= len(a.tabs) || to < 0 || to >= len(a.tabs) || from == to {
		return
	}
	t := a.tabs[from]
	a.tabs = append(a.tabs[:from], a.tabs[from+1:]...)
	if to > len(a.tabs) {
		to = len(a.tabs)
	}
	a.tabs = append(a.tabs[:to], append([]*Tab{t}, a.tabs[to:]...)...)
	for i, tab := range a.tabs {
		if tab == t {
			a.tab = i
			break
		}
	}
	a.refreshWatchers()
	a.saveSessionSoon()
}

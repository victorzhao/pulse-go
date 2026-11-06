package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"pulse/internal/fsx"
	"pulse/internal/platform"
	"pulse/internal/store"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// viewMode is one of Pulse's file views.
type viewMode int

const (
	viewDetails viewMode = iota
	viewTiles
	viewIcons
	viewList
	viewContent
)

func parseView(s string) viewMode {
	switch s {
	case "tiles":
		return viewTiles
	case "icons":
		return viewIcons
	case "list":
		return viewList
	case "content":
		return viewContent
	default:
		return viewDetails
	}
}

func viewKey(v viewMode) string {
	switch v {
	case viewTiles:
		return "tiles"
	case viewIcons:
		return "icons"
	case viewList:
		return "list"
	case viewContent:
		return "content"
	}
	return "details"
}

// layoutMode is one of the pane layouts.
type layoutMode int

const (
	layoutSingle layoutMode = iota
	layoutSplitH
	layoutSplitV
	layoutTriple
	layoutQuad
)

// virtualKind marks a pane showing a folder Pulse makes up rather than one
// on disk.
type virtualKind int

const (
	virtualNone virtualKind = iota
	virtualComputer
	virtualTrash
	virtualTag
	virtualStarred
	virtualRecent
	virtualNetwork
	virtualSearch
)

// row is one line of a pane: an entry, or a group header.
type row struct {
	E        fsx.Entry
	Header   string
	IsHeader bool
	// TrashID is the recycle bin item an entry came from.
	TrashID string
}

// Pane is one folder view inside a tab.
type Pane struct {
	Path       string
	History    []string
	HistAt     int
	View       viewMode
	SortBy     string
	SortDesc   bool
	Group      string
	KindFilter string

	Entries    []fsx.Entry
	Rows       []row
	Selected   int
	Selection  ui.Selection[string]
	List       ui.ListState
	Grid       ui.GridState
	sortOrder  ui.SortOrder
	SortOrders []ui.SortOrder

	Loading bool
	Err     string
	Gen     int

	// Query filters the current folder by name.
	Query string

	Virtual       virtualKind
	TagID         string
	TrashIDs      map[string]string
	TrashOriginal map[string]string
	Drives        []platform.Drive

	SearchRoot  string
	Searching   bool
	SearchDone  bool
	SearchGen   int
	SearchCount int

	Sizes    map[string]int64
	SizeGen  int
	SizeBusy bool

	Compare   map[string]compareMark
	CompareOn bool

	Renaming     string
	ScrollToPath string
	DropHint     int // 0 none, 1 copy, 2 move

	// Title is the tab title last derived from Path.
	Title string
	// viewTab and viewIdx are the pane's place while it builds.
	viewTab *Tab
	viewIdx int
	// watchCancel stops the folder watcher when the pane moves.
	watchCancel context.CancelFunc
	// keepSelection holds chosen paths across a refresh.
	keepSelection []string
	// HiddenCols names the detail columns hidden in this folder.
	HiddenCols map[string]bool
}

// compareMark says how an entry differs from its twin in the other pane.
type compareMark struct {
	Status int // 1 left only, 2 right only, 3 newer, 4 different, 5 same
}

// Tab is one window tab with its own layout and panes.
type Tab struct {
	Title  string
	Layout layoutMode
	Active int
	Panes  []*Pane
	SplitA float32
	SplitB float32
	SplitV float32
}

// App is the whole UI state.
type App struct {
	Win   *mygo.Window
	cfg   *store.Config
	trash *platform.Trash

	locale     string
	systemDark bool

	tabs []*Tab
	tab  int

	sidebarOn      bool
	sidebarWidth   float32
	detailsOn      bool
	detailsPercent float32

	side  sideState
	stash stashState
	wp    wallpaper
	thumb thumbCache

	addressEdit   bool
	addressDraft  string
	addressHandle ui.Handle
	searchHandle  ui.Handle

	searchQuery string

	settings  settingsState
	rename    renameState
	batch     batchRenameState
	newFolder newFolderState
	props     propsState
	confirm   confirmState
	conflict  conflictState
	diff      diffState
	compare   compareState
	search    searchState
	advanced  advancedState
	tags      tagsState

	ops         opsState
	pending     []transferJob
	undo        []undoAction
	toast       string
	toastAction *toastAction
	now         time.Time

	recent     []store.Favorite
	sizes      map[string]sizeCache
	imgDims    map[string]dimEntry
	drives     []platform.Drive
	drivesAt   time.Time
	trashCount int
	trashSize  int64
	trashAt    time.Time

	closing          bool
	lastActivePane   *Pane
	icon             []byte
	tray             *mygo.Tray
	searchHistory    []string
	detailsBasicOpen bool
	detailsTagsOpen  bool
	detailsAttrOpen  bool
}

// toastAction is a toast with an undo button.
type toastAction struct {
	Message string
	Label   string
	Run     func()
}

// undoAction reverses a completed file operation.
type undoAction struct {
	Label string
	Run   func() error
}

// dimEntry remembers an image's pixel size.
type dimEntry struct {
	w, h int
	ok   bool
	at   time.Time
}

// sizeCache remembers a computed folder size.
type sizeCache struct {
	Size  int64
	Files int
	At    time.Time
}

// New builds the app state and restores the last session.
func New(win *mygo.Window) *App {
	cfg := store.Load()
	a := &App{
		Win:              win,
		cfg:              cfg,
		trash:            platform.NewTrash(),
		locale:           mygo.App.Locale(),
		systemDark:       mygo.Theme.IsDark(),
		sidebarOn:        cfg.View.SidebarOn,
		sidebarWidth:     cfg.View.SidebarWidth,
		detailsOn:        cfg.View.ShowDetails,
		detailsPercent:   70,
		recent:           cfg.Recent,
		searchHistory:    append([]string(nil), cfg.SearchHistory...),
		sizes:            map[string]sizeCache{},
		imgDims:          map[string]dimEntry{},
		search:           searchState{Subfolders: true},
		advanced:         advancedState{Subfolders: true},
		detailsBasicOpen: true,
		detailsTagsOpen:  true,
		detailsAttrOpen:  false,
	}
	a.side = newSideState()
	if len(cfg.SidebarOrder) > 0 {
		a.side.order = normalizeSideOrder(cfg.SidebarOrder)
	}
	a.thumb.init()
	if win != nil {
		if w, _ := win.Size(); w > 0 {
			a.detailsPercent = 100 - (a.cfg.View.DetailsWidth/float32(w))*100
			if a.detailsPercent < 30 {
				a.detailsPercent = 70
			}
		}
	}
	mygo.App.SetLanguages("zh-Hans", "zh-Hant", "en")
	if a.cfg.Language != "" {
		mygo.App.SetLocale(a.cfg.Language)
	}
	if win != nil {
		mygo.Theme.OnUpdated(func() {
			win.Update(func() {
				a.systemDark = mygo.Theme.IsDark()
			})
		})
	}
	a.initSession()
	a.loadWallpaper()
	return a
}

// initSession restores the saved tabs, or starts with one tab.
func (a *App) initSession() {
	if a.cfg.General.RestoreTabs && len(a.cfg.Session.Tabs) > 0 {
		for _, ts := range a.cfg.Session.Tabs {
			t := &Tab{Title: ts.Title, Layout: layoutMode(ts.Layout), Active: ts.Active, SplitA: 50, SplitB: 50, SplitV: 50}
			if t.Active < 0 || t.Active >= len(ts.Panes) {
				t.Active = 0
			}
			for _, ps := range ts.Panes {
				p := &Pane{
					Path:     ps.Path,
					View:     parseView(ps.View),
					SortBy:   ps.SortBy,
					SortDesc: ps.SortDesc,
					Group:    ps.Group,
					Selected: -1,
					Sizes:    map[string]int64{},
					Compare:  map[string]compareMark{},
				}
				p.List.DragSelect = true
				t.Panes = append(t.Panes, p)
			}
			if len(t.Panes) == 0 {
				p := a.newPane(a.defaultPath())
				t.Panes = []*Pane{p}
			}
			if t.Title == "" {
				t.Title = a.tabTitle(t.Panes[t.Active])
			}
			a.tabs = append(a.tabs, t)
		}
		a.tab = a.cfg.Session.Active
		if a.tab < 0 || a.tab >= len(a.tabs) {
			a.tab = 0
		}
		for _, t := range a.tabs {
			for _, p := range t.Panes {
				if p.Path == "" {
					p.Path = a.defaultPath()
				}
				p.History = []string{p.Path}
				p.HistAt = 0
				a.loadPane(p)
			}
		}
		return
	}
	p := a.newPane(a.defaultPath())
	t := &Tab{Layout: layoutSingle, Active: 0, Panes: []*Pane{p}, SplitA: 50, SplitB: 50, SplitV: 50}
	t.Title = a.tabTitle(p)
	a.tabs = []*Tab{t}
	a.tab = 0
	a.loadPane(p)
}

// SetIcon keeps the app icon, for the window and the tray.
func (a *App) SetIcon(png []byte) {
	a.icon = png
	a.applyTray()
}

// StartHidden reports whether the app starts in the tray.
func (a *App) StartHidden() bool { return a.cfg.General.Tray }

// HasTray reports whether the tray icon is showing.
func (a *App) HasTray() bool { return a.tray != nil }

// applyTray creates or removes the tray icon as the setting says.
func (a *App) applyTray() {
	if a.Win == nil || len(a.icon) == 0 {
		return
	}
	if a.cfg.General.Tray {
		if a.tray != nil {
			return
		}
		tray, err := mygo.NewTray(mygo.TrayOptions{
			Icon:    a.icon,
			ToolTip: "Pulse",
			Menu: mygo.NewMenu([]*mygo.MenuItem{
				{Label: a.t("app.name"), Click: func(*mygo.MenuItem, *mygo.Window) { a.ShowWindow() }},
				{Label: a.t("open"), Click: func(*mygo.MenuItem, *mygo.Window) { a.ShowWindow() }},
				mygo.Separator(),
				{Role: mygo.RoleQuit},
			}),
		})
		if err == nil {
			a.tray = tray
			tray.OnClick(a.ShowWindow)
		}
		return
	}
	if a.tray != nil {
		a.tray.Destroy()
		a.tray = nil
	}
}

// verticalTabs reports whether tabs show in the sidebar.
func (a *App) verticalTabs() bool {
	return a.cfg.View.VerticalTabs && a.sidebarOn
}

// ShowSearch shows the window and opens the global search dialog, as the
// Alt+Space shortcut does.
func (a *App) ShowSearch() {
	if a.Win == nil {
		return
	}
	a.Win.Update(func() {
		a.Win.Show()
		a.Win.Focus()
		a.openSearchOverlay()
	})
}

// ShowWindow shows and focuses the main window.
func (a *App) ShowWindow() {
	if a.Win == nil {
		return
	}
	a.Win.Show()
	a.Win.Focus()
}

// HideWindow hides the main window to the tray.
func (a *App) HideWindow() {
	if a.Win != nil {
		a.Win.Hide()
	}
}

// defaultPath returns where a new tab starts.
func (a *App) defaultPath() string {
	if a.cfg.General.DefaultLocation != "" && platform.Exists(a.cfg.General.DefaultLocation) {
		return a.cfg.General.DefaultLocation
	}
	return platform.Home()
}

// newPane makes a pane with the configured defaults.
func (a *App) newPane(path string) *Pane {
	v := parseView(a.cfg.View.DefaultView)
	p := &Pane{
		Path:     path,
		View:     v,
		SortBy:   a.cfg.View.SortBy,
		SortDesc: a.cfg.View.SortDesc,
		Group:    a.cfg.View.GroupBy,
		Selected: -1,
		Sizes:    map[string]int64{},
		Compare:  map[string]compareMark{},
		History:  []string{path},
		HistAt:   0,
	}
	p.List.DragSelect = true
	if pref, ok := a.cfg.FolderPrefs[store.FolderKey(path)]; ok {
		p.View = parseView(pref.View)
		p.SortBy = pref.SortBy
		p.SortDesc = pref.SortDesc
		p.Group = pref.Group
		for _, so := range pref.SortOrders {
			p.SortOrders = append(p.SortOrders, ui.SortOrder{Column: so.Column, Descending: so.Descending})
		}
		if len(p.SortOrders) > 0 {
			p.SortBy, p.SortDesc = p.SortOrders[0].Column, p.SortOrders[0].Descending
		}
		p.List.Columns = ui.TableLayout{Order: append([]string(nil), pref.ColumnOrder...), Widths: map[string]float32{}}
		for id, w := range pref.ColumnWidths {
			p.List.Columns.Widths[id] = w
		}
		if len(pref.HiddenColumns) > 0 {
			p.HiddenCols = map[string]bool{}
			for _, id := range pref.HiddenColumns {
				p.HiddenCols[id] = true
			}
		}
	}
	return p
}

// activeTab returns the tab shown.
func (a *App) activeTab() *Tab {
	if a.tab < 0 || a.tab >= len(a.tabs) {
		if len(a.tabs) == 0 {
			return nil
		}
		a.tab = 0
	}
	return a.tabs[a.tab]
}

// activePane returns the pane that commands act on.
func (a *App) activePane() *Pane {
	t := a.activeTab()
	if t == nil || len(t.Panes) == 0 {
		return nil
	}
	if t.Active < 0 || t.Active >= len(t.Panes) {
		t.Active = 0
	}
	return t.Panes[t.Active]
}

// tabTitle derives a tab's title from its active pane.
func (a *App) tabTitle(p *Pane) string {
	if p == nil {
		return a.t("newTabTitle")
	}
	switch p.Virtual {
	case virtualComputer:
		return a.t("tabThisPC")
	case virtualTrash:
		return a.t("tabTrash")
	case virtualTag:
		for _, tag := range a.cfg.Tags {
			if tag.ID == p.TagID {
				return a.tf("tabTag", tag.Name)
			}
		}
		return a.t("tags")
	case virtualStarred:
		return a.t("tabStarred")
	case virtualRecent:
		return a.t("tabRecent")
	case virtualSearch:
		return a.t("tabSearch")
	case virtualNetwork:
		return a.t("network")
	}
	name := filepath.Base(filepath.Clean(p.Path))
	if name == "." || name == string(filepath.Separator) || name == "" {
		if vol := volumeName(p.Path); vol != "" {
			return vol
		}
		return a.t("computer")
	}
	return name
}

func volumeName(path string) string {
	v := filepath.VolumeName(path)
	if v != "" {
		return v
	}
	if path == "/" {
		return "/"
	}
	return ""
}

// View builds the window.
func (a *App) View(c *ui.Context) {
	a.locale = c.Locale()
	a.now = c.Now()
	p := a.palette()
	c.SetTheme(p.theme())

	a.handleShortcuts(c)
	a.loadWallpaper()

	root := ui.Box(c).Fill().Background(ui.Transparent).Draw(func(painter *ui.Painter, r ui.Rect) {
		if a.wp.bmp == nil {
			a.paintBackground(painter, r, p)
		}
	})
	root.Children(func() {
		if a.wp.bmp != nil {
			ui.Image(c, a.wp.bmp).Fill().Fit(ui.Cover).Blur(float32(a.cfg.Appearance.Blur) / 12).Cache()
			dim := ui.RGBA(0, 0, 0, 0.25)
			if !p.Dark {
				dim = ui.RGBA(255, 255, 255, 0.18)
			}
			ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Background(dim)
			ui.Column(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Children(func() {
				a.mainColumn(c, p)
			})
		} else {
			ui.Column(c).Fill().Children(func() {
				a.mainColumn(c, p)
			})
		}
		a.overlays(c, p)
	})
	a.flushToasts(c)
	if a.toastAction != nil {
		act := a.toastAction
		a.toastAction = nil
		a.toast = ""
		c.ToastAction(act.Message, act.Label, act.Run)
	} else if a.toast != "" {
		msg := a.toast
		a.toast = ""
		c.Toast(msg)
	}
}

// flushToasts shows any queued message.
func (a *App) flushToasts(c *ui.Context) {}

// mainColumn builds the title bar, body and status bar.
func (a *App) mainColumn(c *ui.Context, p Palette) {
	a.titleBar(c, p)
	if a.sidebarOn {
		ui.Split(c, &a.sidebarWidth, func() {
			a.sidebar(c, p)
		}, func() {
			a.mainArea(c, p)
		}).Grow(1)
	} else {
		a.mainArea(c, p).Grow(1)
	}
	a.statusBar(c, p)
}

// mainArea builds the toolbar, panes and details panel.
func (a *App) mainArea(c *ui.Context, p Palette) ui.Element {
	return ui.Column(c).Fill().Children(func() {
		a.toolbar(c, p)
		if a.detailsOn {
			ui.SplitPercent(c, &a.detailsPercent, func() {
				a.panes(c, p)
			}, func() {
				a.detailsPanel(c, p)
			}).Grow(1)
		} else {
			a.panes(c, p).Grow(1)
		}
	})
}

// titleBar draws the custom title bar: the app name, the tabs, the new tab
// button, the settings and info buttons, and the window controls.
func (a *App) titleBar(c *ui.Context, p Palette) {
	bar := ui.Row(c).Height(46).DragWindow().BorderWidth(0, 0, 1, 0).BorderColor(p.BorderSoft)
	bar.Background(p.Panel)
	bar.Children(func() {
		w := float32(160)
		if a.sidebarOn {
			w = a.sidebarWidth
		}
		ui.Row(c).Width(w).Padding(0, 12).Gap(8).Children(func() {
			ui.Icon(c, icon("logo")).Size(20, 20).TextColor(p.Accent)
			ui.Text(c, a.t("app.name")).FontSize(12.5).SingleLine().TextColor(p.TextMuted).Grow(1)
			if flatButton(c, p, 24, 24).Children(func() {
				ui.Icon(c, icon("sidebar")).FontSize(14).TextColor(p.TextMuted)
			}).Tooltip(a.t("showSidebar")).Label(a.t("showSidebar")).Clicked() {
				a.sidebarOn = !a.sidebarOn
				a.cfg.View.SidebarOn = a.sidebarOn
				a.saveSessionSoon()
			}
		})
		if !a.verticalTabs() {
			a.tabStrip(c, p)
		}
		if flatButton(c, p, 30, 26).Children(func() {
			ui.Icon(c, icon("plus")).FontSize(15).TextColor(p.TextMuted)
		}).Tooltip(a.t("newTab")).Label(a.t("newTab")).Clicked() {
			a.newTabHere()
		}
		ui.Spacer(c)
		if flatButton(c, p, 30, 26).Children(func() {
			ui.Icon(c, icon("settings")).FontSize(15).TextColor(p.TextMuted)
		}).Tooltip(a.t("settings")).Label(a.t("settings")).Clicked() {
			a.settings.Open = true
		}
		if flatButton(c, p, 30, 26).Children(func() {
			ui.Icon(c, icon("info")).FontSize(15).TextColor(p.TextMuted)
		}).Tooltip(a.t("about")).Label(a.t("about")).Clicked() {
			a.settings.Open = true
			a.settings.Tab = 4
		}
		if a.Win != nil {
			a.windowButtons(c, p)
		}
	})
}

// windowButtons draws minimize, maximize and close for the hidden title bar.
func (a *App) windowButtons(c *ui.Context, p Palette) {
	ui.Row(c).Gap(2).Padding(0, 6).Children(func() {
		if flatButton(c, p, 40, 28).Children(func() {
			ui.Icon(c, icon("minimize")).FontSize(14).TextColor(p.TextMuted)
		}).Tooltip(a.t("minimizeWindow")).Label(a.t("minimizeWindow")).Clicked() {
			if a.Win != nil {
				a.Win.Minimize()
			}
		}
		maxIcon := "maximize"
		if a.Win.IsMaximized() {
			maxIcon = "restore"
		}
		if flatButton(c, p, 40, 28).Children(func() {
			ui.Icon(c, icon(maxIcon)).FontSize(13).TextColor(p.TextMuted)
		}).MaximizeButton().Tooltip(a.t("maximizeWindow")).Label(a.t("maximizeWindow")).Clicked() {
			if a.Win == nil {
				return
			}
			if a.Win.IsMaximized() {
				a.Win.Unmaximize()
			} else {
				a.Win.Maximize()
			}
		}
		close := flatButton(c, p, 40, 28).Children(func() {
			ui.Icon(c, icon("close")).FontSize(14)
		}).TextColor(p.TextMuted).Tooltip(a.t("close")).Label(a.t("close"))
		if close.Hovered() {
			close.Background(ui.Hex("#c42b1c")).TextColor(ui.Hex("#ffffff"))
		}
		if close.Clicked() && a.Win != nil {
			a.Win.Close()
		}
	})
}

// statusBar shows counts, the selection and the view controls.
func (a *App) statusBar(c *ui.Context, p Palette) {
	ap := a.activePane()
	ui.Row(c).Height(28).Padding(0, 12).Gap(14).Background(p.Panel).BorderWidth(1, 0, 0, 0).BorderColor(p.BorderSoft).Children(func() {
		if ap == nil {
			return
		}
		folders, files := 0, 0
		for _, r := range ap.Rows {
			if r.IsHeader {
				continue
			}
			if r.E.IsDir {
				folders++
			} else {
				files++
			}
		}
		ui.Text(c, a.tf("statusItems", folders+files, folders, files)).FontSize(11.5).TextColor(p.TextMuted)
		sel := ap.Selection.Len()
		if sel == 1 {
			if e := a.selectedEntry(ap); e != nil {
				ui.Text(c, "·").TextColor(p.TextFaint)
				ui.Text(c, a.t("selected")+" "+e.Name).FontSize(11.5).TextColor(p.Text).SingleLine()
				if !e.IsDir {
					ui.Text(c, fsx.FormatSize(e.Size)).FontSize(11.5).TextColor(p.TextMuted)
				}
			}
		} else if sel > 1 {
			ui.Text(c, "·").TextColor(p.TextFaint)
			ui.Text(c, a.tf("statusSelected", sel)).FontSize(11.5).TextColor(p.Text)
		}
		if ap.Searching {
			ui.Text(c, a.tf("searchingProgress", ap.SearchCount)).FontSize(11.5).TextColor(p.Accent)
		}
		ui.Spacer(c)
		if ap.SizeBusy {
			ui.Text(c, a.t("computingSize")).FontSize(11.5).TextColor(p.TextMuted)
		}
		if ap.Virtual == virtualNone {
			if size := a.sizeSum(ap); size > 0 {
				ui.Text(c, a.tf("statusSize", fsx.FormatSize(size))).FontSize(11.5).TextColor(p.TextMuted)
			}
		}
		if n := len(a.stash.Items); n > 0 {
			ui.Text(c, a.tf("stashCount", n)).FontSize(11.5).TextColor(p.Accent)
		}
	})
}

// overlays builds the dialogs, modals and menus shown over the window.
func (a *App) overlays(c *ui.Context, p Palette) {
	a.settingsModal(c, p)
	a.renameModal(c, p)
	a.batchRenameModal(c, p)
	a.newFolderModal(c, p)
	a.propertiesModal(c, p)
	a.confirmModal(c, p)
	a.conflictModal(c, p)
	a.diffModal(c, p)
	a.compareModal(c, p)
	a.searchModal(c, p)
	a.advancedModal(c, p)
	a.tagsModal(c, p)
	a.opsOverlay(c, p)
}

// opsOverlay shows a running file operation above the status bar.
func (a *App) opsOverlay(c *ui.Context, p Palette) {
	if !a.ops.Running {
		return
	}
	ui.Box(c).Absolute().Left(0).Right(0).Bottom(40).Children(func() {
		ui.Box(c).Attach(ui.AnchorBottom, ui.AnchorBottom).Width(420).Padding(12, 14).Radius(10).
			Background(p.PanelSolid).Border(1, p.Border).Shadow(0, 8, 24, 0, ui.RGBA(0, 0, 0, 0.35)).Children(func() {
			ui.Column(c).Gap(6).Children(func() {
				ui.Row(c).Gap(8).Children(func() {
					ui.Icon(c, icon("copy")).FontSize(15).TextColor(p.Accent)
					ui.Text(c, a.ops.Title).FontSize(12).Bold().Grow(1)
					ui.Text(c, a.tf("operationProgress", a.ops.Done, a.ops.Total)).FontSize(10.5).TextColor(p.TextMuted)
					if ui.Button(c, a.t("cancel")).FontSize(10).Height(22).Clicked() {
						if a.ops.Cancel != nil {
							a.ops.Cancel()
						}
					}
				})
				frac := float64(0)
				if a.ops.Total > 0 {
					frac = float64(a.ops.Done) / float64(a.ops.Total)
				}
				ui.Progress(c, frac).Height(4).Radius(2)
				if a.ops.Current != "" {
					ui.Text(c, a.ops.Current).FontSize(10).TextColor(p.TextFaint).SingleLine()
				}
			})
		})
	})
}

// newTabHere opens a tab at the active pane's folder.
func (a *App) newTabHere() {
	path := a.defaultPath()
	if p := a.activePane(); p != nil && p.Virtual == virtualNone {
		path = p.Path
	}
	a.newTab(path)
}

// newTab opens a tab at path.
func (a *App) newTab(path string) {
	p := a.newPane(path)
	t := &Tab{Layout: layoutSingle, Active: 0, Panes: []*Pane{p}, SplitA: 50, SplitB: 50, SplitV: 50}
	t.Title = a.tabTitle(p)
	a.tabs = append(a.tabs, t)
	a.tab = len(a.tabs) - 1
	a.loadPane(p)
	a.refreshWatchers()
	a.saveSessionSoon()
}

// closeTab closes the tab at index i, keeping at least one.
func (a *App) closeTab(i int) {
	if len(a.tabs) <= 1 {
		return
	}
	if i < 0 || i >= len(a.tabs) {
		return
	}
	a.tabs = append(a.tabs[:i], a.tabs[i+1:]...)
	if a.tab >= len(a.tabs) {
		a.tab = len(a.tabs) - 1
	}
	if a.tab < 0 {
		a.tab = 0
	}
	a.saveSessionSoon()
}

// selectTab shows tab i.
func (a *App) selectTab(i int) {
	if i < 0 || i >= len(a.tabs) {
		return
	}
	a.tab = i
	a.refreshWatchers()
	a.saveSessionSoon()
}

// setLayout changes the active tab's layout, creating panes as needed.
func (a *App) setLayout(m layoutMode) {
	t := a.activeTab()
	if t == nil {
		return
	}
	t.Layout = m
	need := 1
	switch m {
	case layoutSplitH, layoutSplitV:
		need = 2
	case layoutTriple:
		need = 3
	case layoutQuad:
		need = 4
	}
	for len(t.Panes) < need {
		path := t.Panes[len(t.Panes)-1].Path
		if path == "" {
			path = a.defaultPath()
		}
		p := a.newPane(path)
		t.Panes = append(t.Panes, p)
		a.loadPane(p)
	}
	if t.Active >= need {
		t.Active = 0
	}
	if t.SplitA == 0 {
		t.SplitA = 50
	}
	a.saveSessionSoon()
}

// ensurePanes makes a tab's panes match its layout.
func (a *App) ensurePanes(t *Tab) {
	if t == nil {
		return
	}
	need := 1
	switch t.Layout {
	case layoutSplitH, layoutSplitV:
		need = 2
	case layoutTriple:
		need = 3
	case layoutQuad:
		need = 4
	}
	for len(t.Panes) < need {
		path := a.defaultPath()
		if len(t.Panes) > 0 && t.Panes[len(t.Panes)-1].Path != "" {
			path = t.Panes[len(t.Panes)-1].Path
		}
		p := a.newPane(path)
		t.Panes = append(t.Panes, p)
		a.loadPane(p)
	}
}

// saveSessionSoon saves the session and settings after a change.
func (a *App) saveSessionSoon() {
	if a.cfg == nil || a.Win == nil {
		return
	}
	a.SaveSession()
}

// SaveSession writes the tabs and view state to the configuration.
func (a *App) SaveSession() {
	if a.cfg == nil {
		return
	}
	if a.cfg.FolderPrefs == nil {
		a.cfg.FolderPrefs = map[string]store.FolderPref{}
	}
	s := store.Session{Active: a.tab}
	for _, t := range a.tabs {
		ts := store.TabState{Title: t.Title, Layout: int(t.Layout), Active: t.Active}
		for _, p := range t.Panes {
			if p.Virtual == virtualNone && p.Path != "" {
				a.cfg.FolderPrefs[store.FolderKey(p.Path)] = a.folderPrefFromPane(p)
			}
			ts.Panes = append(ts.Panes, store.PaneState{
				Path:     p.Path,
				View:     viewKey(p.View),
				SortBy:   p.SortBy,
				SortDesc: p.SortDesc,
				Group:    p.Group,
			})
		}
		s.Tabs = append(s.Tabs, ts)
	}
	a.cfg.Session = s
	a.cfg.View.SidebarOn = a.sidebarOn
	a.cfg.View.SidebarWidth = a.sidebarWidth
	a.cfg.View.ShowDetails = a.detailsOn
	a.cfg.View.DetailsWidth = a.detailsWidth()
	a.cfg.Recent = a.recent
	a.cfg.SearchHistory = append([]string(nil), a.searchHistory...)
	a.cfg.SidebarOrder = append([]string(nil), a.side.order...)
	_ = store.Save(a.cfg)
}

// detailsWidth converts the split percentage to a width.
func (a *App) detailsWidth() float32 {
	if a.Win == nil {
		return 320
	}
	w, _ := a.Win.Size()
	return float32(w) * (100 - a.detailsPercent) / 100
}

// setLocale changes the display language and saves it.
func (a *App) setLocale(tag string) {
	a.cfg.Language = tag
	if tag == "" {
		mygo.App.SetLocale("")
	} else {
		mygo.App.SetLocale(tag)
	}
	a.SetupMenu()
	a.saveSessionSoon()
}

// addRecent records a visited folder.
func (a *App) addRecent(path string, isDir bool) {
	if path == "" {
		return
	}
	for i, r := range a.recent {
		if r.Path == path {
			a.recent = append(a.recent[:i], a.recent[i+1:]...)
			break
		}
	}
	name := filepath.Base(path)
	if name == "." || name == "" {
		name = path
	}
	a.recent = append([]store.Favorite{{Name: name, Path: path}}, a.recent...)
	if len(a.recent) > 30 {
		a.recent = a.recent[:30]
	}
}

// sortedRows sorts and groups a pane's entries for display.
func (a *App) sortedRows(p *Pane) []row {
	entries := make([]fsx.Entry, 0, len(p.Entries))
	for _, e := range p.Entries {
		if !a.cfg.View.ShowHidden && e.Hidden {
			continue
		}
		if p.Query != "" && !fsx.MatchName(e.Name, p.Query) {
			continue
		}
		if p.KindFilter != "" && p.KindFilter != "all" {
			if !kindMatchesFilter(e, p.KindFilter) {
				continue
			}
		}
		entries = append(entries, e)
	}
	sortEntries(entries, p.SortOrders, p.SortBy, p.SortDesc, a.cfg.View.FoldersFirst, p.Sizes)
	group := p.Group
	if p.View == viewTiles || p.View == viewIcons {
		group = "none"
	}
	if group == "none" || group == "" {
		out := make([]row, len(entries))
		for i, e := range entries {
			out[i] = row{E: e}
		}
		return out
	}
	out := make([]row, 0, len(entries)+8)
	last := ""
	for _, e := range entries {
		key := a.groupKey(e, group)
		if key != last {
			out = append(out, row{Header: key, IsHeader: true})
			last = key
		}
		out = append(out, row{E: e})
	}
	return out
}

func kindMatchesFilter(e fsx.Entry, f string) bool {
	switch f {
	case "folders":
		return e.IsDir
	case "documents":
		return e.Kind == fsx.KindDocument || e.Kind == fsx.KindPDF || e.Kind == fsx.KindSheet || e.Kind == fsx.KindSlides || e.Kind == fsx.KindText
	case "images":
		return e.Kind == fsx.KindImage
	case "videos":
		return e.Kind == fsx.KindVideo
	case "audio":
		return e.Kind == fsx.KindAudio
	case "archives":
		return e.Kind == fsx.KindArchive
	case "code":
		return e.Kind == fsx.KindCode
	case "other":
		return !e.IsDir && e.Kind == fsx.KindOther
	}
	return true
}

// groupKey returns the header of an entry's group.
func (a *App) groupKey(e fsx.Entry, group string) string {
	switch group {
	case "name":
		name := strings.ToUpper(e.Name)
		if name == "" {
			return "#"
		}
		r := []rune(name)[0]
		if r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return string(r)
		}
		return "#"
	case "date":
		return a.groupDate(e.ModTime, a.now)
	case "type":
		if e.IsDir {
			return a.t("folder")
		}
		return a.typeName(e.Kind)
	case "size":
		if e.IsDir {
			return a.t("folder")
		}
		s := e.Size
		switch {
		case s < 16*1024:
			return "0 – 16 KB"
		case s < 1024*1024:
			return "16 KB – 1 MB"
		case s < 16*1024*1024:
			return "1 – 16 MB"
		case s < 256*1024*1024:
			return "16 – 256 MB"
		default:
			return "> 256 MB"
		}
	}
	return ""
}

func (a *App) groupDate(t, now time.Time) string {
	y, m, d := t.Date()
	ny, nm, nd := now.Date()
	if y == ny && m == nm && d == nd {
		return a.t("today")
	}
	if y == ny && m == nm && d == nd-1 {
		return a.t("yesterday")
	}
	if y == ny && m == nm && nd-d <= 7 && nd > d {
		return a.t("thisWeek")
	}
	if y == ny && m == nm {
		return a.t("thisMonth")
	}
	if y == ny {
		return a.t("thisYearGroup")
	}
	if a.lang() == langEN {
		return fmt.Sprintf("%d", y)
	}
	return fmt.Sprintf("%d 年", y)
}

// sortEntries orders entries by one or more columns, folders first.
func sortEntries(entries []fsx.Entry, orders []ui.SortOrder, by string, desc bool, foldersFirst bool, sizes map[string]int64) {
	if len(orders) == 0 {
		orders = []ui.SortOrder{{Column: by, Descending: desc}}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if foldersFirst && a.IsDir != b.IsDir {
			return a.IsDir
		}
		for _, o := range orders {
			cmp := compareEntries(a, b, o.Column, sizes)
			if cmp == 0 {
				continue
			}
			if o.Descending {
				return cmp > 0
			}
			return cmp < 0
		}
		return false
	})
}

// compareEntries compares two entries by a column.
func compareEntries(a, b fsx.Entry, by string, sizes map[string]int64) int {
	switch by {
	case "date":
		if a.ModTime.Equal(b.ModTime) {
			return 0
		}
		if a.ModTime.Before(b.ModTime) {
			return -1
		}
		return 1
	case "type":
		ta, tb := typeSortKey(a), typeSortKey(b)
		return strings.Compare(ta, tb)
	case "size":
		sa, sb := entrySize(a, sizes), entrySize(b, sizes)
		if sa == sb {
			return 0
		}
		if sa < sb {
			return -1
		}
		return 1
	default:
		return naturalCompare(a.Name, b.Name)
	}
}

// naturalCompare compares names, putting digit runs in numeric order.
func naturalCompare(a, b string) int {
	la, lb := strings.ToLower(a), strings.ToLower(b)
	i, j := 0, 0
	for i < len(la) && j < len(lb) {
		ca, cb := la[i], lb[j]
		if ca >= '0' && ca <= '9' && cb >= '0' && cb <= '9' {
			si, sj := i, j
			for si < len(la) && la[si] >= '0' && la[si] <= '9' {
				si++
			}
			for sj < len(lb) && lb[sj] >= '0' && lb[sj] <= '9' {
				sj++
			}
			na := strings.TrimLeft(la[i:si], "0")
			nb := strings.TrimLeft(lb[j:sj], "0")
			if len(na) != len(nb) {
				if len(na) < len(nb) {
					return -1
				}
				return 1
			}
			if na != nb {
				if na < nb {
					return -1
				}
				return 1
			}
			i, j = si, sj
			continue
		}
		if ca != cb {
			if ca < cb {
				return -1
			}
			return 1
		}
		i++
		j++
	}
	if len(la) == len(lb) {
		return 0
	}
	if len(la) < len(lb) {
		return -1
	}
	return 1
}

// naturalLess reports whether a sorts before b.
func naturalLess(a, b string) bool { return naturalCompare(a, b) < 0 }

// typeSortKey orders entries by their extension.
func typeSortKey(e fsx.Entry) string {
	if e.IsDir {
		return "0"
	}
	return e.Ext + "|" + e.Name
}

// entrySize returns the size of an entry, using computed folder sizes.
func entrySize(e fsx.Entry, sizes map[string]int64) int64 {
	if e.IsDir {
		if sizes != nil {
			if s, ok := sizes[e.Path]; ok {
				return s
			}
		}
		return 0
	}
	return e.Size
}

// statFile fills an entry for a path that a virtual folder lists.
func statFile(path string) (fsx.Entry, error) {
	e, err := fsx.Stat(path)
	if err != nil {
		return fsx.Entry{Path: path, Name: filepath.Base(path)}, err
	}
	return e, nil
}

// osPathExists reports whether a path exists.
func osPathExists(path string) bool { return platform.Exists(path) }

// removeRecent drops a path from the recent list.
func (a *App) removeRecent(path string) {
	for i, r := range a.recent {
		if r.Path == path {
			a.recent = append(a.recent[:i], a.recent[i+1:]...)
			return
		}
	}
}

// homeDir returns the user's home.
func (a *App) homeDir() string { return platform.Home() }

// ensureDir checks a folder exists and is a folder.
func ensureDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("not a folder")
	}
	return nil
}

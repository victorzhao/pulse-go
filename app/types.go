package app

import (
	"time"

	"pulse/internal/fsx"
	"pulse/internal/store"

	"github.com/egoist/mygo/ui"
)

// sideState remembers the sidebar's collapsed sections and rows.
type sideState struct {
	openStarred  bool
	openQuick    bool
	openRecent   bool
	openTags     bool
	openComputer bool
	openNetwork  bool
	order        []string
	starredSel   string
	quickSel     string
	recentSel    string
	tagSel       string
	driveSel     string
	// addingTag marks the inline tag creation row.
	addingTag   bool
	newTagName  string
	newTagColor int
}

func newSideState() sideState {
	return sideState{
		openStarred: true, openQuick: true, openRecent: true,
		openTags: true, openComputer: true, openNetwork: true,
		order: []string{"starred", "quickAccess", "recent", "tags", "computer", "network"},
	}
}

// normalizeSideOrder keeps the known sections, adding any that are missing.
func normalizeSideOrder(order []string) []string {
	known := []string{"starred", "quickAccess", "recent", "tags", "computer", "network"}
	seen := map[string]bool{}
	out := make([]string, 0, len(known))
	for _, id := range order {
		for _, k := range known {
			if id == k && !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
	}
	for _, k := range known {
		if !seen[k] {
			out = append(out, k)
		}
	}
	return out
}

// stashItem is a file collected in the staging tray.
type stashItem struct {
	Path  string
	Name  string
	IsDir bool
	Size  int64
	Move  bool
	At    time.Time
}

// stashState is the staging tray.
type stashState struct {
	Items []stashItem
	Open  bool
	// Over marks that the pointer drags files over the tray.
	Over bool
}

// thumbCache keeps decoded image thumbnails.
type thumbCache struct {
	m map[string]*ui.Bitmap
	// failed remembers paths that could not be decoded.
	failed map[string]bool
	// loading marks decodes in flight.
	loading map[string]bool
	gen     int
}

func (t *thumbCache) init() {
	t.m = map[string]*ui.Bitmap{}
	t.failed = map[string]bool{}
	t.loading = map[string]bool{}
}

// settingsState is the settings dialog.
type settingsState struct {
	Open bool
	Tab  int
	// wallpaperDraft is the path being edited.
	wallpaperDraft string
}

// renameState is the rename dialog.
type renameState struct {
	Open bool
	Path string
	Name string
	Pane *Pane
}

// newFolderState is the new folder dialog.
type newFolderState struct {
	Open bool
	Name string
	Pane *Pane
	File bool
}

// batchRenameState is the batch rename dialog.
type batchRenameState struct {
	Open       bool
	Pane       *Pane
	Entries    []fsx.Entry
	Find       string
	Replace    string
	Prefix     string
	Suffix     string
	Numbering  bool
	Start      int
	Digits     int
	IncludeExt bool
	Preview    []string
	Error      string
}

// propsState is the properties dialog.
type propsState struct {
	Open        bool
	Entries     []fsx.Entry
	FolderSize  int64
	FolderFiles int
	FolderDirs  int
	Busy        bool
	Gen         int
}

// confirmState is a yes/no dialog.
type confirmState struct {
	Open    bool
	Title   string
	Body    string
	Confirm string
	Danger  bool
	Action  func()
}

// conflictState asks what to do about an existing destination.
type conflictState struct {
	Open     bool
	Path     string
	Dir      bool
	Src      string
	Merge    bool
	Choice   int // 0 replace, 1 skip, 2 keep both, 3 merge, -1 cancel
	ApplyAll bool
	Resume   func(choice int, applyAll bool)
	// Pending, Job, Choices and Index resolve several conflicts in turn.
	Pending []conflictItem
	Job     transferJob
	Choices map[string]int
	Index   int
}

// diffState is the text comparison dialog.
type diffState struct {
	Open       bool
	Title      string
	A, B       string
	Rows       []fsx.DiffRow
	Mode       int // 0 side by side, 1 unified
	OnlyDiff   bool
	IgnoreWS   bool
	IgnoreCase bool
	At         int
	Error      string
	List       ui.ListState
}

// compareState is the folder comparison dialog.
type compareState struct {
	Open      bool
	Left      *Pane
	Right     *Pane
	Same      int
	LeftOnly  int
	RightOnly int
	Newer     int
	Different int
}

// searchState is the global search dialog.
type searchState struct {
	Open      bool
	Query     string
	Scope     int // 0 current folder, 1 this PC, 2 picked folder
	Root      string
	Results   []fsx.Entry
	Selected  int
	List      ui.ListState
	Searching bool
	Done      int
	Gen       int
	Cancel    func()
	Err       string
	// showAll marks that results go to a tab instead of the dialog.
	showAll bool
	// Filters narrow a search.
	Kind       string
	Date       string
	Size       string
	Tag        string
	Hidden     bool
	Subfolders bool
}

// advancedState is the advanced search dialog.
type advancedState struct {
	Open       bool
	Query      string
	Kind       string
	Date       string
	Size       string
	Tag        string
	Hidden     bool
	Subfolders bool
	Scope      int
	Root       string
}

// tagsState is the tag manager dialog.
type tagsState struct {
	Open bool
	// editing is the tag index being renamed, -1 for none.
	Editing int
	Name    string
	Color   string
}

// opsState tracks a running file operation.
type opsState struct {
	Running bool
	Title   string
	Done    int
	Total   int
	Current string
	Err     string
	Cancel  func()
	// Gen invalidates late updates.
	Gen int
}

// folderPrefFor returns the stored preference for a folder.
func (a *App) folderPrefFor(path string) (store.FolderPref, bool) {
	p, ok := a.cfg.FolderPrefs[store.FolderKey(path)]
	return p, ok
}

// Package store keeps Pulse's preferences, tags, favorites, per-folder view
// settings and the last session on disk as JSON.
package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"pulse/internal/platform"
)

// Config is everything Pulse remembers between runs.
type Config struct {
	Version        int                   `json:"version"`
	Language       string                `json:"language"` // "", "zh-Hans", "zh-Hant", "en"
	Appearance     Appearance            `json:"appearance"`
	View           ViewPrefs             `json:"view"`
	General        GeneralPrefs          `json:"general"`
	Favorites      []Favorite            `json:"favorites"`
	Quick          []Favorite            `json:"quick"`
	Recent         []Favorite            `json:"recent"`
	Tags           []Tag                 `json:"tags"`
	TagAssignments map[string]string     `json:"tag_assignments"`
	FolderPrefs    map[string]FolderPref `json:"folder_prefs"`
	Session        Session               `json:"session"`
	SidebarOrder   []string              `json:"sidebar_order,omitempty"`
	SearchHistory  []string              `json:"search_history,omitempty"`
}

// Appearance holds how the window looks.
type Appearance struct {
	Theme        string `json:"theme"` // "system", "light", "dark"
	Accent       string `json:"accent"`
	Wallpaper    string `json:"wallpaper"`
	Transparency int    `json:"transparency"` // 0-100
	Blur         int    `json:"blur"`         // 0-100
	RowHeight    string `json:"row_height"`   // "compact", "standard", "roomy"
	TextRender   string `json:"text_render"`  // "auto", "clear", "smooth"
}

// ViewPrefs holds the file views' defaults.
type ViewPrefs struct {
	Zebra             bool    `json:"zebra"`
	ExtensionColors   bool    `json:"extension_colors"`
	SizeBars          bool    `json:"size_bars"`
	SmartDates        bool    `json:"smart_dates"`
	ShowHidden        bool    `json:"show_hidden"`
	FoldersFirst      bool    `json:"folders_first"`
	ShowDetails       bool    `json:"show_details"`
	DetailsWidth      float32 `json:"details_width"`
	SidebarWidth      float32 `json:"sidebar_width"`
	SidebarOn         bool    `json:"sidebar_on"`
	DefaultView       string  `json:"default_view"`
	GroupBy           string  `json:"group_by"`
	SortBy            string  `json:"sort_by"`
	SortDesc          bool    `json:"sort_desc"`
	ConfirmDelete     bool    `json:"confirm_delete"`
	ComputeFolderSize bool    `json:"compute_folder_size"`
	AutoRefresh       bool    `json:"auto_refresh"`
	ShowDriveBars     bool    `json:"show_drive_bars"`
	VerticalTabs      bool    `json:"vertical_tabs"`
}

// GeneralPrefs holds startup and integration settings.
type GeneralPrefs struct {
	RestoreTabs     bool   `json:"restore_tabs"`
	DefaultLocation string `json:"default_location"`
	SingleClickOpen bool   `json:"single_click_open"`
	Tray            bool   `json:"tray"`
	Startup         bool   `json:"startup"`
}

// Favorite is a starred folder or file.
type Favorite struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Tag is one color label.
type Tag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// FolderPref is the view state remembered for one folder.
// SortPref is one column of a folder's multi-column sort.
type SortPref struct {
	Column     string `json:"column"`
	Descending bool   `json:"descending"`
}

type FolderPref struct {
	View          string             `json:"view"`
	SortBy        string             `json:"sort_by"`
	SortDesc      bool               `json:"sort_desc"`
	SortOrders    []SortPref         `json:"sort_orders,omitempty"`
	Group         string             `json:"group"`
	ColumnOrder   []string           `json:"column_order,omitempty"`
	ColumnWidths  map[string]float32 `json:"column_widths,omitempty"`
	HiddenColumns []string           `json:"hidden_columns,omitempty"`
}

// Session is the last set of tabs.
type Session struct {
	Tabs   []TabState `json:"tabs"`
	Active int        `json:"active"`
}

// TabState is one tab's layout and panes.
type TabState struct {
	Title  string      `json:"title"`
	Layout int         `json:"layout"`
	Active int         `json:"active"`
	Panes  []PaneState `json:"panes"`
}

// PaneState is one pane's folder and view.
type PaneState struct {
	Path     string `json:"path"`
	View     string `json:"view"`
	SortBy   string `json:"sort_by"`
	SortDesc bool   `json:"sort_desc"`
	Group    string `json:"group"`
}

// DefaultTags are Pulse's seven color labels.
func DefaultTags() []Tag {
	return []Tag{
		{"urgent", "Urgent", "#e81123"},
		{"design", "Design review", "#f7630c"},
		{"progress", "In progress", "#107c10"},
		{"pending", "Pending", "#ffb900"},
		{"inspiration", "Inspiration", "#8764b8"},
		{"reference", "Reference", "#0078d4"},
		{"archive", "Archive", "#767676"},
	}
}

// Default returns a fresh configuration.
func Default() *Config {
	return &Config{
		Version:  1,
		Language: "",
		Appearance: Appearance{
			Theme:        "system",
			Accent:       "#0078d4",
			Transparency: 35,
			Blur:         60,
			RowHeight:    "standard",
			TextRender:   "auto",
		},
		View: ViewPrefs{
			Zebra:             false,
			ExtensionColors:   true,
			SizeBars:          true,
			SmartDates:        true,
			ShowHidden:        false,
			FoldersFirst:      true,
			ShowDetails:       true,
			DetailsWidth:      320,
			SidebarWidth:      250,
			SidebarOn:         true,
			DefaultView:       "details",
			GroupBy:           "none",
			SortBy:            "name",
			SortDesc:          false,
			ConfirmDelete:     true,
			ComputeFolderSize: true,
			AutoRefresh:       true,
			ShowDriveBars:     true,
		},
		General: GeneralPrefs{
			RestoreTabs:     true,
			DefaultLocation: platform.Home(),
			SingleClickOpen: false,
			Tray:            false,
			Startup:         false,
		},
		Favorites:      nil,
		Tags:           DefaultTags(),
		TagAssignments: map[string]string{},
		FolderPrefs:    map[string]FolderPref{},
		Session:        Session{},
	}
}

// Path returns the configuration file's path.
func Path() string { return filepath.Join(platform.ConfigDir(), "config.json") }

// Load reads the configuration, filling in defaults for missing fields.
func Load() *Config {
	cfg := Default()
	data, err := os.ReadFile(Path())
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return cfg
		}
		return cfg
	}
	_ = json.Unmarshal(data, cfg)
	normalize(cfg)
	return cfg
}

func normalize(cfg *Config) {
	if cfg.Version < 1 {
		// Configurations from before auto-refresh existed turn it on.
		cfg.View.AutoRefresh = true
		cfg.Version = 1
	}
	if cfg.Appearance.Theme == "" {
		cfg.Appearance.Theme = "system"
	}
	if cfg.Appearance.Accent == "" {
		cfg.Appearance.Accent = "#0078d4"
	}
	if cfg.Appearance.RowHeight == "" {
		cfg.Appearance.RowHeight = "standard"
	}
	if cfg.View.DefaultView == "" {
		cfg.View.DefaultView = "details"
	}
	if cfg.View.DetailsWidth <= 0 {
		cfg.View.DetailsWidth = 320
	}
	if cfg.View.SidebarWidth <= 0 {
		cfg.View.SidebarWidth = 250
	}
	if cfg.View.GroupBy == "" {
		cfg.View.GroupBy = "none"
	}
	if cfg.View.SortBy == "" {
		cfg.View.SortBy = "name"
	}
	if len(cfg.Tags) == 0 {
		cfg.Tags = DefaultTags()
	}
	if cfg.TagAssignments == nil {
		cfg.TagAssignments = map[string]string{}
	}
	if cfg.FolderPrefs == nil {
		cfg.FolderPrefs = map[string]FolderPref{}
	}
	if cfg.General.DefaultLocation == "" {
		cfg.General.DefaultLocation = platform.Home()
	}
}

// Save writes the configuration atomically.
func Save(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	p := Path()
	tmp := p + ".tmp"
	_ = os.Remove(tmp)
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, p); err != nil {
		// Windows may refuse to replace a file that is briefly open; write
		// the configuration directly rather than lose it.
		if err2 := os.WriteFile(p, data, 0o644); err2 != nil {
			return err
		}
	}
	_ = os.Remove(tmp)
	return nil
}

// FolderKey normalizes a folder path for the per-folder settings map.
func FolderKey(path string) string {
	p := filepath.Clean(path)
	if strings.HasSuffix(p, string(filepath.Separator)) && len(p) > 1 {
		p = strings.TrimSuffix(p, string(filepath.Separator))
	}
	return p
}

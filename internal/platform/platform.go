// Package platform isolates the operating system details Pulse needs:
// drives, the default handlers, the terminal and the recycle bin.
package platform

import (
	"os"
	"path/filepath"
)

// Drive is a mounted volume.
type Drive struct {
	// Name is the volume label, or "" when it has none.
	Name string
	// Path is the mount point or root, such as C:\ or /Volumes/Data.
	Path string
	// Letter is the drive letter on Windows, "" elsewhere.
	Letter string
	// Kind is a hint: "fixed", "removable", "network", "optical", "other".
	Kind  string
	Total int64
	Free  int64
}

// Dir is a well-known folder.
type Dir struct {
	Name string
	Path string
}

// Home returns the user's home directory.
func Home() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return string(filepath.Separator)
}

// UserDirs returns the well-known folders that exist, in display order.
func UserDirs() []Dir {
	home := Home()
	candidates := []Dir{
		{"desktop", filepath.Join(home, "Desktop")},
		{"documents", filepath.Join(home, "Documents")},
		{"downloads", filepath.Join(home, "Downloads")},
		{"pictures", filepath.Join(home, "Pictures")},
		{"music", filepath.Join(home, "Music")},
		{"videos", filepath.Join(home, "Videos")},
	}
	out := make([]Dir, 0, len(candidates))
	for _, d := range candidates {
		if info, err := os.Stat(d.Path); err == nil && info.IsDir() {
			out = append(out, d)
		}
	}
	return out
}

// ConfigDir returns Pulse's directory for preferences and the trash.
func ConfigDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = filepath.Join(Home(), ".config")
	}
	dir = filepath.Join(dir, "PulseGo")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// Exists reports whether path exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

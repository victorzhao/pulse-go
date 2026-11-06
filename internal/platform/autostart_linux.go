//go:build linux

package platform

import (
	"os"
	"path/filepath"
)

// SetAutostart adds or removes Pulse from the desktop's startup entries.
func SetAutostart(enable bool) error {
	base, err := os.UserConfigDir()
	if err != nil {
		base = filepath.Join(Home(), ".config")
	}
	dir := filepath.Join(base, "autostart")
	path := filepath.Join(dir, "pulse.desktop")
	if !enable {
		err := os.Remove(path)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	body := "[Desktop Entry]\nType=Application\nName=Pulse\nExec=\"" + exe + "\"\nX-GNOME-Autostart-enabled=true\n"
	return os.WriteFile(path, []byte(body), 0o644)
}

//go:build darwin

package platform

import (
	"os"
	"path/filepath"
)

const launchAgent = "com.easytech.pulse.plist"

// SetAutostart adds or removes Pulse's launch agent.
func SetAutostart(enable bool) error {
	dir := filepath.Join(Home(), "Library", "LaunchAgents")
	path := filepath.Join(dir, launchAgent)
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
	body := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>com.easytech.pulse</string>
<key>ProgramArguments</key><array><string>` + exe + `</string></array>
<key>RunAtLoad</key><true/>
</dict></plist>
`
	return os.WriteFile(path, []byte(body), 0o644)
}

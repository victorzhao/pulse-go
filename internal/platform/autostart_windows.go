//go:build windows

package platform

import (
	"os"
	"os/exec"
)

const runKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`

// SetAutostart adds or removes Pulse from the user's startup entries.
func SetAutostart(enable bool) error {
	if enable {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		return exec.Command("reg.exe", "add", runKey, "/v", "PulseGo", "/t", "REG_SZ", "/d", exe, "/f").Run()
	}
	_ = exec.Command("reg.exe", "delete", runKey, "/v", "PulseGo", "/f").Run()
	return nil
}

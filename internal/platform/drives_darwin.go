//go:build darwin

package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func diskUsage(path string) (total, free int64) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0
	}
	return int64(st.Blocks) * int64(st.Bsize), int64(st.Bavail) * int64(st.Bsize)
}

// fsType returns a volume's filesystem name.
func fsType(path string) string {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return ""
	}
	b := st.Fstypename[:]
	n := 0
	for n < len(b) && b[n] != 0 {
		n++
	}
	raw := make([]byte, 0, n)
	for i := 0; i < n; i++ {
		raw = append(raw, byte(b[i]))
	}
	return string(raw)
}

// Drives lists mounted volumes on macOS.
func Drives() []Drive {
	out := []Drive{}
	total, free := diskUsage("/")
	out = append(out, Drive{Name: "Macintosh HD", Path: "/", Kind: "fixed", Total: total, Free: free})
	if entries, err := os.ReadDir("/Volumes"); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			p := filepath.Join("/Volumes", e.Name())
			total, free := diskUsage(p)
			kind := "removable"
			switch fsType(p) {
			case "smbfs", "nfs", "afpfs", "webdav", "davfs":
				kind = "network"
			}
			out = append(out, Drive{Name: e.Name(), Path: p, Kind: kind, Total: total, Free: free})
		}
	}
	return out
}

// Open opens a path with its default application.
func Open(path string) error { return exec.Command("open", path).Start() }

// OpenWith lets the user choose an application for a path.
func OpenWith(path string) error {
	return exec.Command("osascript",
		"-e", "on run argv",
		"-e", "set f to POSIX file (item 1 of argv)",
		"-e", "tell application \"Finder\" to open f using (choose application)",
		path,
	).Start()
}

// Reveal shows a path in Finder, selected.
func Reveal(path string) error { return exec.Command("open", "-R", path).Start() }

// OpenTerminal opens Terminal in dir.
func OpenTerminal(dir string) error { return exec.Command("open", "-a", "Terminal", dir).Start() }

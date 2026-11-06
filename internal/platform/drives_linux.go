//go:build linux

package platform

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func diskUsage(path string) (total, free int64) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0
	}
	return int64(st.Blocks) * int64(st.Bsize), int64(st.Bavail) * int64(st.Bsize)
}

var pseudoFS = map[string]bool{
	"proc": true, "sysfs": true, "devtmpfs": true, "devpts": true, "tmpfs": true,
	"cgroup": true, "cgroup2": true, "overlay": true, "squashfs": true, "ramfs": true,
	"autofs": true, "mqueue": true, "debugfs": true, "tracefs": true, "securityfs": true,
	"pstore": true, "bpf": true, "configfs": true, "fusectl": true, "hugetlbfs": true,
	"nsfs": true, "rpc_pipefs": true, "binfmt_misc": true,
}

// Drives lists the mounted filesystems.
func Drives() []Drive {
	seen := map[string]bool{}
	out := []Drive{}
	add := func(path, name, kind string) {
		if seen[path] {
			return
		}
		seen[path] = true
		total, free := diskUsage(path)
		out = append(out, Drive{Name: name, Path: path, Kind: kind, Total: total, Free: free})
	}
	add("/", "", "fixed")
	f, err := os.Open("/proc/mounts")
	if err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) < 3 {
				continue
			}
			mp := unescapeMount(fields[1])
			fstype := fields[2]
			if pseudoFS[fstype] {
				continue
			}
			name := ""
			if mp != "/" {
				name = filepath.Base(mp)
			}
			kind := "fixed"
			if strings.HasPrefix(mp, "/media/") || strings.HasPrefix(mp, "/run/media/") || strings.HasPrefix(mp, "/mnt/") {
				kind = "removable"
			}
			switch fstype {
			case "cifs", "smb3", "smbfs", "nfs", "nfs4", "fuse.sshfs", "sshfs", "davfs", "webdav":
				kind = "network"
			}
			add(mp, name, kind)
		}
		f.Close()
	}
	return out
}

func unescapeMount(s string) string {
	r := strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`)
	return r.Replace(s)
}

// Open opens a path with the desktop's default application.
func Open(path string) error { return exec.Command("xdg-open", path).Start() }

// OpenWith opens a path with its default application on Linux, where the
// desktop has no standard chooser.
func OpenWith(path string) error { return Open(path) }

// Reveal opens the parent folder in the file manager.
func Reveal(path string) error { return exec.Command("xdg-open", filepath.Dir(path)).Start() }

// OpenTerminal opens a terminal in dir.
func OpenTerminal(dir string) error {
	candidates := []struct {
		name string
		args []string
	}{
		{"x-terminal-emulator", []string{"-e", "bash"}},
		{"gnome-terminal", []string{"--working-directory=" + dir}},
		{"konsole", []string{"--workdir", dir}},
		{"xfce4-terminal", []string{"--working-directory=" + dir}},
		{"alacritty", []string{"--working-directory", dir}},
		{"kitty", []string{"--directory", dir}},
		{"xterm", []string{"-e", "bash"}},
	}
	for _, c := range candidates {
		if _, err := exec.LookPath(c.name); err != nil {
			continue
		}
		cmd := exec.Command(c.name, c.args...)
		cmd.Dir = dir
		if err := cmd.Start(); err == nil {
			return nil
		}
	}
	return exec.Command("xdg-open", dir).Start()
}

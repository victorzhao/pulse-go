//go:build windows

package platform

import (
	"os/exec"
	"syscall"
	"unsafe"
)

var (
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procGetLogicalDrives    = kernel32.NewProc("GetLogicalDrives")
	procGetDriveTypeW       = kernel32.NewProc("GetDriveTypeW")
	procGetVolumeInfoW      = kernel32.NewProc("GetVolumeInformationW")
	procGetDiskFreeSpaceExW = kernel32.NewProc("GetDiskFreeSpaceExW")
)

func driveTypeName(t uint32) string {
	switch t {
	case 2:
		return "removable"
	case 3:
		return "fixed"
	case 4:
		return "network"
	case 5:
		return "optical"
	case 6:
		return "ramdisk"
	}
	return "other"
}

func volumeLabel(root string) string {
	rootp, err := syscall.UTF16PtrFromString(root)
	if err != nil {
		return ""
	}
	buf := make([]uint16, 261)
	fsName := make([]uint16, 261)
	var serial, maxLen, flags uint32
	ok, _, _ := procGetVolumeInfoW.Call(
		uintptr(unsafe.Pointer(rootp)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		uintptr(unsafe.Pointer(&serial)),
		uintptr(unsafe.Pointer(&maxLen)),
		uintptr(unsafe.Pointer(&flags)),
		uintptr(unsafe.Pointer(&fsName[0])),
		uintptr(len(fsName)),
	)
	if ok == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

func diskUsage(path string) (total, free int64) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0
	}
	var avail, tot, totFree uint64
	ok, _, _ := procGetDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(p)),
		uintptr(unsafe.Pointer(&avail)),
		uintptr(unsafe.Pointer(&tot)),
		uintptr(unsafe.Pointer(&totFree)),
	)
	if ok == 0 {
		return 0, 0
	}
	return int64(tot), int64(avail)
}

// Drives lists the volumes Windows knows, with their labels and capacity.
func Drives() []Drive {
	mask, _, _ := procGetLogicalDrives.Call()
	out := make([]Drive, 0, 8)
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		letter := string(rune('A'+i)) + ":"
		root := letter + `\`
		rootp, err := syscall.UTF16PtrFromString(root)
		if err != nil {
			continue
		}
		typ, _, _ := procGetDriveTypeW.Call(uintptr(unsafe.Pointer(rootp)))
		total, free := diskUsage(root)
		out = append(out, Drive{
			Name:   volumeLabel(root),
			Path:   root,
			Letter: letter,
			Kind:   driveTypeName(uint32(typ)),
			Total:  total,
			Free:   free,
		})
	}
	return out
}

// Open opens a path with its default application.
func Open(path string) error {
	cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", path)
	return cmd.Start()
}

// OpenWith opens a path with a chosen application.
func OpenWith(path string) error {
	return exec.Command("rundll32.exe", "shell32.dll,OpenAs_RunDLL", path).Start()
}

// Reveal shows a path in Explorer, selected.
func Reveal(path string) error {
	return exec.Command("explorer.exe", "/select,"+path).Start()
}

// OpenTerminal opens a terminal in dir.
func OpenTerminal(dir string) error {
	if _, err := exec.LookPath("wt.exe"); err == nil {
		if err := exec.Command("wt.exe", "-d", dir).Start(); err == nil {
			return nil
		}
	}
	return exec.Command("cmd.exe", "/c", "start", "cmd.exe", "/k", "cd", "/d", dir).Start()
}

//go:build windows

package fsx

import (
	"syscall"
	"unsafe"
)

var (
	kernel32Hidden        = syscall.NewLazyDLL("kernel32.dll")
	procGetFileAttributes = kernel32Hidden.NewProc("GetFileAttributesW")
)

const (
	fileAttributeHidden = 0x2
	fileAttributeSystem = 0x4
)

// osHidden reports whether Windows marks a path hidden or system.
func osHidden(path string) bool {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false
	}
	attrs, _, _ := procGetFileAttributes.Call(uintptr(unsafe.Pointer(p)))
	if attrs == 0xFFFFFFFF {
		return false
	}
	return attrs&(fileAttributeHidden|fileAttributeSystem) != 0
}

//go:build !windows && !linux && !darwin

package platform

import "os/exec"

// Drives lists nothing on platforms without an implementation.
func Drives() []Drive { return nil }

// Open opens a path with the default application.
func Open(path string) error { return exec.Command("xdg-open", path).Start() }

// OpenWith opens a path with the default application.
func OpenWith(path string) error { return Open(path) }

// Reveal opens the parent folder.
func Reveal(path string) error { return exec.Command("xdg-open", path).Start() }

// OpenTerminal is not supported.
func OpenTerminal(dir string) error { return nil }

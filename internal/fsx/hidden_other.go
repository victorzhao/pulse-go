//go:build !windows

package fsx

// osHidden reports no hidden attribute outside Windows.
func osHidden(path string) bool { return false }

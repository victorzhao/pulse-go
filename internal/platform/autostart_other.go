//go:build !windows && !linux && !darwin

package platform

// SetAutostart is not supported on this platform.
func SetAutostart(enable bool) error { return nil }

//go:build linux

package service

// NewManager returns a systemd-based Manager on Linux.
// pmuxPath is the absolute path to the pmux binary (resolved at install time).
// configDir is the path to ~/.config/pmux (for log file paths in service config).
func NewManager(pmuxPath string, configDir string) Manager {
	return newSystemdManager(pmuxPath, configDir)
}

//go:build darwin

package service

// NewManager returns a launchd-based Manager on macOS.
// pmuxPath is the absolute path to the pmux binary (resolved at install time).
// configDir is the path to ~/.config/pmux (for log file paths in service config).
func NewManager(pmuxPath string, configDir string) Manager {
	return newLaunchdManager(pmuxPath, configDir)
}

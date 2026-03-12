//go:build !darwin && !linux

package service

// NewManager returns an unsupported Manager on platforms other than macOS and Linux.
func NewManager(pmuxPath string, configDir string) Manager {
	return &unsupportedManager{platform: "unsupported"}
}

// Package proxy handles tmux passthrough: exec tmux -L pmux [args...].
package proxy

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// ExecTmux replaces the current process with tmux targeting the given socket.
// This function does not return on success (the process is replaced).
func ExecTmux(socket string, args ...string) error {
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux not found in PATH: %w", err)
	}

	// Build args: tmux -L <socket> [user args...]
	tmuxArgs := []string{"tmux", "-L", socket}
	tmuxArgs = append(tmuxArgs, args...)

	// Replace current process with tmux
	return syscall.Exec(tmuxPath, tmuxArgs, os.Environ())
}

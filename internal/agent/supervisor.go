package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/shiftinbits/pmux-agent/internal/auth"
	"github.com/shiftinbits/pmux-agent/internal/config"
)

const pidFileName = "agent.pid"

// EnsureRunning checks if the agent is already running and starts it if not.
// Returns nil if the agent is running (or was started successfully).
// Does nothing if no identity exists (agent can't authenticate without one).
func EnsureRunning(paths config.Paths) error {
	// No identity — agent can't authenticate
	if !auth.HasIdentity(paths.KeysDir) {
		return nil
	}

	pidFile := filepath.Join(paths.ConfigDir, pidFileName)

	if isRunning(pidFile) {
		return nil
	}

	return spawn(pidFile)
}

// RemovePIDFile removes the agent PID file on clean shutdown.
func RemovePIDFile(paths config.Paths) {
	os.Remove(filepath.Join(paths.ConfigDir, pidFileName)) //nolint:errcheck
}

// PIDFilePath returns the path to the agent PID file.
func PIDFilePath(paths config.Paths) string {
	return filepath.Join(paths.ConfigDir, pidFileName)
}

// isRunning checks if a process with the PID from the file is alive.
func isRunning(pidFile string) bool {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false
	}

	// On Unix, FindProcess always succeeds. Check with signal 0.
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	return process.Signal(syscall.Signal(0)) == nil
}

// spawn starts a new agent process in the background.
func spawn(pidFile string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find executable path: %w", err)
	}

	cmd := exec.Command(exe, "--agent")
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	// Start in a new session so it's not tied to the calling terminal
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start agent: %w", err)
	}

	// Write PID file
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0600); err != nil {
		return fmt.Errorf("write PID file: %w", err)
	}

	// Release — don't wait for the background process
	cmd.Process.Release()

	return nil
}

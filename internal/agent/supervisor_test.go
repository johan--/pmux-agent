package agent

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/shiftinbits/pmux-agent/internal/config"
)

func TestIsRunning_CurrentPID(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "agent.pid")

	// Write our own PID — process definitely exists
	pid := os.Getpid()
	os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0600)

	if !isRunning(pidFile) {
		t.Error("expected isRunning=true for current process PID")
	}
}

func TestIsRunning_NoPIDFile(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "agent.pid")

	if isRunning(pidFile) {
		t.Error("expected isRunning=false when PID file doesn't exist")
	}
}

func TestIsRunning_StalePID(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "agent.pid")

	// Write a PID that almost certainly doesn't exist
	os.WriteFile(pidFile, []byte("9999999"), 0600)

	if isRunning(pidFile) {
		t.Error("expected isRunning=false for non-existent PID")
	}
}

func TestIsRunning_InvalidContent(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "agent.pid")

	os.WriteFile(pidFile, []byte("not-a-number"), 0600)

	if isRunning(pidFile) {
		t.Error("expected isRunning=false for invalid PID content")
	}
}

func TestRemovePIDFile(t *testing.T) {
	dir := t.TempDir()
	paths := config.Paths{ConfigDir: dir}

	pidFile := filepath.Join(dir, pidFileName)
	os.WriteFile(pidFile, []byte("12345"), 0600)

	// Verify file exists
	if _, err := os.Stat(pidFile); os.IsNotExist(err) {
		t.Fatal("PID file should exist before removal")
	}

	RemovePIDFile(paths)

	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Error("PID file should be removed")
	}
}

func TestPIDFilePath(t *testing.T) {
	paths := config.Paths{ConfigDir: "/tmp/test-pmux"}
	got := PIDFilePath(paths)
	want := "/tmp/test-pmux/agent.pid"
	if got != want {
		t.Errorf("PIDFilePath = %q, want %q", got, want)
	}
}

func TestEnsureRunning_NoIdentity(t *testing.T) {
	dir := t.TempDir()
	paths := config.Paths{
		ConfigDir: dir,
		KeysDir:   filepath.Join(dir, "keys"),
	}

	// No identity exists — EnsureRunning should be a no-op
	err := EnsureRunning(paths)
	if err != nil {
		t.Errorf("EnsureRunning should not error without identity: %v", err)
	}

	// No PID file should be created
	pidFile := filepath.Join(dir, pidFileName)
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Error("PID file should not exist when no identity")
	}
}

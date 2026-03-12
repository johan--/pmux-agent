//go:build darwin

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeExecCommand and TestHelperProcess are defined in exec_test.go (shared across all platform tests).

func TestLaunchdManager_PlistPath(t *testing.T) {
	mgr := newLaunchdManager("/usr/local/bin/pmux", "/Users/test/.config/pmux")
	path := mgr.plistPath()
	if !strings.HasSuffix(path, "Library/LaunchAgents/io.pmux.agent.plist") {
		t.Errorf("unexpected plist path: %s", path)
	}
}

func TestLaunchdManager_GeneratePlist(t *testing.T) {
	mgr := newLaunchdManager("/usr/local/bin/pmux", "/Users/test/.config/pmux")
	plist := mgr.generatePlist()

	if !strings.Contains(plist, "<string>io.pmux.agent</string>") {
		t.Error("plist missing label")
	}
	if !strings.Contains(plist, "<string>/usr/local/bin/pmux</string>") {
		t.Error("plist missing pmux path")
	}
	if !strings.Contains(plist, "<string>agent</string>") {
		t.Error("plist missing 'agent' argument")
	}
	if !strings.Contains(plist, "<string>run</string>") {
		t.Error("plist missing 'run' argument")
	}
	if !strings.Contains(plist, "agent.log") {
		t.Error("plist missing log path")
	}
}

func TestLaunchdManager_Install_WritesPlist(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := newLaunchdManager("/usr/local/bin/pmux", "/Users/test/.config/pmux")
	mgr.plistDir = tmpDir

	if err := mgr.writePlist(); err != nil {
		t.Fatalf("writePlist: %v", err)
	}

	path := filepath.Join(tmpDir, "io.pmux.agent.plist")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read plist: %v", err)
	}

	if !strings.Contains(string(data), "io.pmux.agent") {
		t.Error("written plist missing label")
	}
}

func TestLaunchdManager_IsInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := newLaunchdManager("/usr/local/bin/pmux", "/Users/test/.config/pmux")
	mgr.plistDir = tmpDir

	if mgr.IsInstalled() {
		t.Error("should not be installed before writing plist")
	}

	if err := mgr.writePlist(); err != nil {
		t.Fatalf("writePlist: %v", err)
	}

	if !mgr.IsInstalled() {
		t.Error("should be installed after writing plist")
	}
}

func TestLaunchctlHint(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		exitCode int
		wantHint string // substring expected in hint; empty means no hint
	}{
		{
			name:     "domain does not support specified action",
			output:   "Bootstrap failed: 125: Domain does not support specified action",
			exitCode: 125,
			wantHint: "Run without sudo",
		},
		{
			name:     "case insensitive match",
			output:   "DOMAIN DOES NOT SUPPORT SPECIFIED ACTION",
			exitCode: 125,
			wantHint: "Run without sudo",
		},
		{
			name:     "could not find specified service",
			output:   "Could not find specified service",
			exitCode: 113,
			wantHint: "pmux agent install",
		},
		{
			name:     "operation not permitted",
			output:   "Operation not permitted",
			exitCode: 1,
			wantHint: "Permission denied",
		},
		{
			name:     "no such file or directory",
			output:   "No such file or directory",
			exitCode: 2,
			wantHint: "plist file is missing",
		},
		{
			name:     "exit 125 unknown message",
			output:   "some unknown error text",
			exitCode: 125,
			wantHint: "pmux agent uninstall && pmux agent install",
		},
		{
			name:     "no hint for generic error",
			output:   "some unknown error text",
			exitCode: 1,
			wantHint: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hint := launchctlHint(tt.output, tt.exitCode)
			if tt.wantHint == "" {
				if hint != "" {
					t.Errorf("expected no hint, got: %s", hint)
				}
			} else {
				if !strings.Contains(hint, tt.wantHint) {
					t.Errorf("hint %q does not contain %q", hint, tt.wantHint)
				}
			}
		})
	}
}

func TestCheckNotRoot_NonRoot(t *testing.T) {
	// This test runs as the current (non-root) user in normal test runs.
	if os.Getuid() == 0 {
		t.Skip("test requires non-root user")
	}
	if err := checkNotRoot(); err != nil {
		t.Errorf("checkNotRoot should pass for non-root user: %v", err)
	}
}

func TestLaunchdInstall_Success(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("test requires non-root user")
	}

	orig := execCommand
	t.Cleanup(func() { execCommand = orig })
	execCommand = fakeExecCommand("success")

	tmpDir := t.TempDir()
	mgr := newLaunchdManager("/usr/local/bin/pmux", "/tmp/test-config")
	mgr.plistDir = tmpDir

	if err := mgr.Install(); err != nil {
		t.Fatalf("Install() returned unexpected error: %v", err)
	}

	// Verify plist was written
	plistPath := filepath.Join(tmpDir, launchdPlistFile)
	if _, err := os.Stat(plistPath); err != nil {
		t.Errorf("plist file not written: %v", err)
	}
}

func TestLaunchdInstall_AlreadyBootstrapped(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("test requires non-root user")
	}

	orig := execCommand
	t.Cleanup(func() { execCommand = orig })

	callCount := 0
	execCommand = func(name string, args ...string) *exec.Cmd {
		callCount++
		// First call (bootstrap) returns "already bootstrapped".
		// All subsequent calls (bootout during Uninstall, re-bootstrap) succeed.
		if callCount == 1 {
			return fakeExecCommand("already_bootstrapped")(name, args...)
		}
		return fakeExecCommand("success")(name, args...)
	}

	tmpDir := t.TempDir()
	mgr := newLaunchdManager("/usr/local/bin/pmux", "/tmp/test-config")
	mgr.plistDir = tmpDir

	if err := mgr.Install(); err != nil {
		t.Fatalf("Install() with already-bootstrapped should succeed: %v", err)
	}

	// Plist should be present after re-install
	plistPath := filepath.Join(tmpDir, launchdPlistFile)
	if _, err := os.Stat(plistPath); err != nil {
		t.Errorf("plist file missing after re-install: %v", err)
	}
}

func TestLaunchdUninstall(t *testing.T) {
	orig := execCommand
	t.Cleanup(func() { execCommand = orig })
	execCommand = fakeExecCommand("success")

	tmpDir := t.TempDir()
	mgr := newLaunchdManager("/usr/local/bin/pmux", "/tmp/test-config")
	mgr.plistDir = tmpDir

	// Write plist so Uninstall has something to remove
	if err := mgr.writePlist(); err != nil {
		t.Fatalf("writePlist: %v", err)
	}

	plistPath := filepath.Join(tmpDir, launchdPlistFile)
	if _, err := os.Stat(plistPath); err != nil {
		t.Fatalf("plist not created: %v", err)
	}

	if err := mgr.Uninstall(); err != nil {
		t.Fatalf("Uninstall() returned unexpected error: %v", err)
	}

	// Plist file should be removed
	if _, err := os.Stat(plistPath); !os.IsNotExist(err) {
		t.Error("plist file should have been removed by Uninstall()")
	}
}

func TestLaunchdStart_KickstartSuccess(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("test requires non-root user")
	}

	orig := execCommand
	t.Cleanup(func() { execCommand = orig })
	execCommand = fakeExecCommand("success")

	tmpDir := t.TempDir()
	mgr := newLaunchdManager("/usr/local/bin/pmux", "/tmp/test-config")
	mgr.plistDir = tmpDir

	if err := mgr.Start(); err != nil {
		t.Fatalf("Start() returned unexpected error: %v", err)
	}
}

func TestLaunchdStart_KickstartFail_BootstrapSuccess(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("test requires non-root user")
	}

	orig := execCommand
	t.Cleanup(func() { execCommand = orig })

	callCount := 0
	execCommand = func(name string, args ...string) *exec.Cmd {
		callCount++
		// First call (kickstart) fails; second call (bootstrap) succeeds.
		if callCount == 1 {
			return fakeExecCommand("failure")(name, args...)
		}
		return fakeExecCommand("success")(name, args...)
	}

	tmpDir := t.TempDir()
	mgr := newLaunchdManager("/usr/local/bin/pmux", "/tmp/test-config")
	mgr.plistDir = tmpDir

	if err := mgr.Start(); err != nil {
		t.Fatalf("Start() should succeed when kickstart fails but bootstrap succeeds: %v", err)
	}
}

func TestLaunchdStop_Success(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("test requires non-root user")
	}

	orig := execCommand
	t.Cleanup(func() { execCommand = orig })
	execCommand = fakeExecCommand("success")

	tmpDir := t.TempDir()
	mgr := newLaunchdManager("/usr/local/bin/pmux", "/tmp/test-config")
	mgr.plistDir = tmpDir

	if err := mgr.Stop(); err != nil {
		t.Fatalf("Stop() returned unexpected error: %v", err)
	}
}

func TestLaunchdStop_NotFound(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("test requires non-root user")
	}

	orig := execCommand
	t.Cleanup(func() { execCommand = orig })
	execCommand = fakeExecCommand("not_found")

	tmpDir := t.TempDir()
	mgr := newLaunchdManager("/usr/local/bin/pmux", "/tmp/test-config")
	mgr.plistDir = tmpDir

	// "Could not find specified service" should be treated as a no-op, not an error.
	if err := mgr.Stop(); err != nil {
		t.Fatalf("Stop() with not-found service should return nil, got: %v", err)
	}
}

func TestLaunchdStatus_Running(t *testing.T) {
	orig := execCommand
	t.Cleanup(func() { execCommand = orig })
	execCommand = fakeExecCommand("launchd_status_running")

	tmpDir := t.TempDir()
	mgr := newLaunchdManager("/usr/local/bin/pmux", "/tmp/test-config")
	mgr.plistDir = tmpDir

	// Write plist so IsInstalled() returns true
	if err := mgr.writePlist(); err != nil {
		t.Fatalf("writePlist: %v", err)
	}

	status, err := mgr.Status()
	if err != nil {
		t.Fatalf("Status() returned unexpected error: %v", err)
	}
	if !status.Running {
		t.Error("expected Status.Running = true")
	}
	if status.PID != 12345 {
		t.Errorf("expected Status.PID = 12345, got %d", status.PID)
	}
}

func TestLaunchdStatus_NotRunning(t *testing.T) {
	orig := execCommand
	t.Cleanup(func() { execCommand = orig })
	execCommand = fakeExecCommand("launchd_status_not_running")

	tmpDir := t.TempDir()
	mgr := newLaunchdManager("/usr/local/bin/pmux", "/tmp/test-config")
	mgr.plistDir = tmpDir

	// Write plist so IsInstalled() returns true
	if err := mgr.writePlist(); err != nil {
		t.Fatalf("writePlist: %v", err)
	}

	status, err := mgr.Status()
	if err != nil {
		t.Fatalf("Status() returned unexpected error: %v", err)
	}
	if status.Running {
		t.Error("expected Status.Running = false")
	}
}

func TestWrapLaunchctlError_WithHint(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		wantHint string
	}{
		{
			name:     "domain does not support specified action",
			output:   "Domain does not support specified action",
			wantHint: "Run without sudo",
		},
		{
			name:     "could not find specified service",
			output:   "Could not find specified service",
			wantHint: "pmux agent install",
		},
		{
			name:     "operation not permitted",
			output:   "Operation not permitted",
			wantHint: "Permission denied",
		},
		{
			name:     "no such file or directory",
			output:   "No such file or directory",
			wantHint: "plist file is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := wrapLaunchctlError("test-verb", tt.output, fmt.Errorf("exit status 1"))
			if err == nil {
				t.Fatal("expected non-nil error")
			}
			if !strings.Contains(err.Error(), tt.wantHint) {
				t.Errorf("error %q does not contain hint %q", err.Error(), tt.wantHint)
			}
		})
	}
}

func TestWrapLaunchctlError_WithoutHint(t *testing.T) {
	err := wrapLaunchctlError("test-verb", "some completely unknown error", fmt.Errorf("exit status 1"))
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	// The error should contain the verb and output but no hint suffix indicator
	errStr := err.Error()
	if !strings.Contains(errStr, "test-verb") {
		t.Errorf("error %q should contain verb 'test-verb'", errStr)
	}
	if !strings.Contains(errStr, "some completely unknown error") {
		t.Errorf("error %q should contain the output", errStr)
	}
	// No hint line (hint is appended after "\n  ")
	if strings.Contains(errStr, "\n  ") {
		t.Errorf("error %q should not contain a hint, but got hint suffix", errStr)
	}
}

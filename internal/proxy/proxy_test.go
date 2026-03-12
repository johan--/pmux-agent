package proxy

import (
	"os"
	"strings"
	"testing"
)

func TestExecTmux_TmuxNotFound(t *testing.T) {
	t.Setenv("PATH", "")

	err := ExecTmux("pmux", "", "ls")
	if err == nil {
		t.Fatal("expected error when tmux is not in PATH, got nil")
	}
	if !strings.Contains(err.Error(), "tmux not found") {
		t.Errorf("expected error to contain 'tmux not found', got: %v", err)
	}
}

func TestExecTmux_ExplicitBin(t *testing.T) {
	var capturedArgv0 string
	var capturedArgv []string
	var capturedEnvv []string

	orig := sysExec
	t.Cleanup(func() { sysExec = orig })
	sysExec = func(argv0 string, argv []string, envv []string) error {
		capturedArgv0 = argv0
		capturedArgv = argv
		capturedEnvv = envv
		return nil
	}

	err := ExecTmux("pmux", "/usr/bin/tmux", "ls", "-t", "work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedArgv0 != "/usr/bin/tmux" {
		t.Errorf("expected argv0 to be '/usr/bin/tmux', got: %q", capturedArgv0)
	}

	wantArgv := []string{"tmux", "-L", "pmux", "ls", "-t", "work"}
	if len(capturedArgv) != len(wantArgv) {
		t.Fatalf("expected argv len %d, got %d: %v", len(wantArgv), len(capturedArgv), capturedArgv)
	}
	for i, want := range wantArgv {
		if capturedArgv[i] != want {
			t.Errorf("argv[%d]: expected %q, got %q", i, want, capturedArgv[i])
		}
	}

	wantEnv := os.Environ()
	if len(capturedEnvv) != len(wantEnv) {
		t.Errorf("expected envv len %d, got %d", len(wantEnv), len(capturedEnvv))
	}
}

func TestExecTmux_LookPathResolves(t *testing.T) {
	var capturedArgv0 string

	orig := sysExec
	t.Cleanup(func() { sysExec = orig })
	sysExec = func(argv0 string, argv []string, envv []string) error {
		capturedArgv0 = argv0
		return nil
	}

	// With an empty tmuxBin, ExecTmux should resolve "tmux" via PATH.
	// If tmux is not installed, skip this test rather than fail.
	err := ExecTmux("pmux", "")
	if err != nil {
		if strings.Contains(err.Error(), "tmux not found") {
			t.Skip("tmux not available in PATH — skipping LookPath resolution test")
		}
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedArgv0 == "" {
		t.Error("expected sysExec to be called with a resolved tmux path, got empty string")
	}
	// The resolved path must be absolute (LookPath always returns absolute paths).
	if !strings.HasPrefix(capturedArgv0, "/") {
		t.Errorf("expected resolved path to be absolute, got: %q", capturedArgv0)
	}
}

func TestExecTmux_NoArgs(t *testing.T) {
	var capturedArgv []string

	orig := sysExec
	t.Cleanup(func() { sysExec = orig })
	sysExec = func(argv0 string, argv []string, envv []string) error {
		capturedArgv = argv
		return nil
	}

	err := ExecTmux("mysocket", "/usr/bin/tmux")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantArgv := []string{"tmux", "-L", "mysocket"}
	if len(capturedArgv) != len(wantArgv) {
		t.Fatalf("expected argv len %d, got %d: %v", len(wantArgv), len(capturedArgv), capturedArgv)
	}
	for i, want := range wantArgv {
		if capturedArgv[i] != want {
			t.Errorf("argv[%d]: expected %q, got %q", i, want, capturedArgv[i])
		}
	}
}

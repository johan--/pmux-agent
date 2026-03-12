package service

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
)

// TestHelperProcess is the subprocess entry point used by fakeExecCommand.
// When the test binary is invoked with -test.run=TestHelperProcess and the
// GO_WANT_HELPER_PROCESS env var set, this function runs the requested behavior
// and exits, acting as a mock for exec.Command calls.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	behavior := os.Getenv("GO_HELPER_BEHAVIOR")
	switch behavior {
	case "success":
		os.Exit(0)
	case "failure":
		fmt.Fprintln(os.Stderr, "mock command failed")
		os.Exit(1)
	case "status_running":
		fmt.Println("ActiveState=active")
		fmt.Println("MainPID=54321")
		os.Exit(0)
	case "status_stopped":
		fmt.Println("ActiveState=inactive")
		fmt.Println("MainPID=0")
		os.Exit(0)
	// Launchd-specific behaviors
	case "already_bootstrapped":
		fmt.Fprintln(os.Stderr, "already bootstrapped")
		os.Exit(1)
	case "launchd_status_running":
		fmt.Println("pid = 12345")
		os.Exit(0)
	case "launchd_status_not_running":
		os.Exit(1)
	case "not_found":
		fmt.Fprintln(os.Stderr, "Could not find specified service")
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "unknown GO_HELPER_BEHAVIOR: %q\n", behavior)
		os.Exit(1)
	}
}

// fakeExecCommand returns a function that replaces execCommand in tests.
// The returned function produces a *exec.Cmd that re-invokes the test binary
// targeting TestHelperProcess, which exits according to the given behavior.
func fakeExecCommand(behavior string) func(string, ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", name}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = append(os.Environ(),
			"GO_WANT_HELPER_PROCESS=1",
			"GO_HELPER_BEHAVIOR="+behavior,
		)
		return cmd
	}
}

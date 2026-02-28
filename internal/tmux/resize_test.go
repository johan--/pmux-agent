package tmux

import (
	"testing"
	"time"
)

func TestPaneSizeTracker_TrackAndResize(t *testing.T) {
	skipIfNoTmux(t)
	tc := testClient(t)

	_, err := tc.CreateSession("resize-track-test", "")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	sessions, err := tc.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	paneID := sessions[0].Windows[0].Panes[0].ID

	tracker := NewPaneSizeTracker(tc)

	if err := tracker.TrackAndResize(paneID, 40, 12); err != nil {
		t.Fatalf("TrackAndResize: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	sessions, err = tc.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	size := sessions[0].Windows[0].Panes[0].Size
	if size.Cols != 40 {
		t.Errorf("cols = %d, want 40", size.Cols)
	}
	if size.Rows != 12 {
		t.Errorf("rows = %d, want 12", size.Rows)
	}

	tracker.mu.Lock()
	count := tracker.attachCount[paneID]
	tracker.mu.Unlock()
	if count != 1 {
		t.Errorf("attachCount = %d, want 1", count)
	}
}

func TestPaneSizeTracker_RestoreIfLast(t *testing.T) {
	skipIfNoTmux(t)
	tc := testClient(t)

	_, err := tc.CreateSession("resize-restore-test", "")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	sessions, err := tc.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	paneID := sessions[0].Windows[0].Panes[0].ID

	tracker := NewPaneSizeTracker(tc)

	if err := tracker.TrackAndResize(paneID, 40, 12); err != nil {
		t.Fatalf("TrackAndResize: %v", err)
	}

	if err := tracker.RestoreIfLast(paneID); err != nil {
		t.Fatalf("RestoreIfLast: %v", err)
	}

	tracker.mu.Lock()
	_, tracked := tracker.attachCount[paneID]
	tracker.mu.Unlock()
	if tracked {
		t.Error("expected attachCount entry to be removed after last detach")
	}
}

func TestPaneSizeTracker_MultipleAttach_RestoreOnlyAfterLast(t *testing.T) {
	skipIfNoTmux(t)
	tc := testClient(t)

	_, err := tc.CreateSession("multi-attach-test", "")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	sessions, err := tc.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	paneID := sessions[0].Windows[0].Panes[0].ID

	tracker := NewPaneSizeTracker(tc)

	if err := tracker.TrackAndResize(paneID, 40, 12); err != nil {
		t.Fatalf("TrackAndResize 1: %v", err)
	}

	if err := tracker.TrackAndResize(paneID, 50, 15); err != nil {
		t.Fatalf("TrackAndResize 2: %v", err)
	}

	// First detach — should NOT auto-resize (one still attached)
	if err := tracker.RestoreIfLast(paneID); err != nil {
		t.Fatalf("RestoreIfLast 1: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	tracker.mu.Lock()
	count := tracker.attachCount[paneID]
	tracker.mu.Unlock()
	if count != 1 {
		t.Errorf("attachCount after first detach = %d, want 1", count)
	}

	// Verify size is still mobile-resized (50x15 from second attach)
	sessions, err = tc.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	size := sessions[0].Windows[0].Panes[0].Size
	if size.Cols != 50 || size.Rows != 15 {
		t.Errorf("size after first detach = %dx%d, want 50x15", size.Cols, size.Rows)
	}

	// Second detach — should auto-resize
	if err := tracker.RestoreIfLast(paneID); err != nil {
		t.Fatalf("RestoreIfLast 2: %v", err)
	}

	tracker.mu.Lock()
	_, tracked := tracker.attachCount[paneID]
	tracker.mu.Unlock()
	if tracked {
		t.Error("expected attachCount entry to be removed after last detach")
	}
}

func TestPaneSizeTracker_RestoreWhenPaneKilled(t *testing.T) {
	skipIfNoTmux(t)
	tc := testClient(t)

	_, err := tc.CreateSession("killed-pane-test", "")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	_, err = tc.CreateSession("keep-alive", "")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	sessions, err := tc.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	var paneID string
	for _, s := range sessions {
		if s.Name == "killed-pane-test" {
			paneID = s.Windows[0].Panes[0].ID
			break
		}
	}
	if paneID == "" {
		t.Fatal("could not find killed-pane-test pane")
	}

	tracker := NewPaneSizeTracker(tc)
	if err := tracker.TrackAndResize(paneID, 40, 12); err != nil {
		t.Fatalf("TrackAndResize: %v", err)
	}

	if err := tc.KillSession("killed-pane-test"); err != nil {
		t.Fatalf("KillSession: %v", err)
	}

	if err := tracker.RestoreIfLast(paneID); err != nil {
		t.Errorf("RestoreIfLast on killed pane should not error, got: %v", err)
	}
}

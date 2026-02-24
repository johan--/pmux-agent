package tmux

import (
	"testing"
	"time"
)

func TestPaneSizeTracker_SaveAndResize(t *testing.T) {
	skipIfNoTmux(t)
	tc := testClient(t)

	// Create a session
	_, err := tc.CreateSession("resize-save-test", "")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	sessions, err := tc.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	paneID := sessions[0].Windows[0].Panes[0].ID
	originalSize := sessions[0].Windows[0].Panes[0].Size

	tracker := NewPaneSizeTracker(tc)

	// SaveAndResize should save original and resize
	if err := tracker.SaveAndResize(paneID, 40, 12); err != nil {
		t.Fatalf("SaveAndResize: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Verify pane was resized
	size, err := tracker.getCurrentSize(paneID)
	if err != nil {
		t.Fatalf("getCurrentSize: %v", err)
	}
	if size.Cols != 40 {
		t.Errorf("cols = %d, want 40", size.Cols)
	}
	if size.Rows != 12 {
		t.Errorf("rows = %d, want 12", size.Rows)
	}

	// Verify original was saved
	tracker.mu.Lock()
	saved, ok := tracker.savedSizes[paneID]
	tracker.mu.Unlock()
	if !ok {
		t.Fatal("expected original size to be saved")
	}
	if saved.Cols != originalSize.Cols || saved.Rows != originalSize.Rows {
		t.Errorf("saved size = %dx%d, want %dx%d", saved.Cols, saved.Rows, originalSize.Cols, originalSize.Rows)
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
	originalSize := sessions[0].Windows[0].Panes[0].Size

	tracker := NewPaneSizeTracker(tc)

	// Attach and resize
	if err := tracker.SaveAndResize(paneID, 40, 12); err != nil {
		t.Fatalf("SaveAndResize: %v", err)
	}

	// Restore
	if err := tracker.RestoreIfLast(paneID); err != nil {
		t.Fatalf("RestoreIfLast: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Verify original size is restored
	size, err := tracker.getCurrentSize(paneID)
	if err != nil {
		t.Fatalf("getCurrentSize: %v", err)
	}
	if size.Cols != originalSize.Cols {
		t.Errorf("cols = %d, want %d", size.Cols, originalSize.Cols)
	}
	if size.Rows != originalSize.Rows {
		t.Errorf("rows = %d, want %d", size.Rows, originalSize.Rows)
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
	originalSize := sessions[0].Windows[0].Panes[0].Size

	tracker := NewPaneSizeTracker(tc)

	// First mobile attaches
	if err := tracker.SaveAndResize(paneID, 40, 12); err != nil {
		t.Fatalf("SaveAndResize 1: %v", err)
	}

	// Second mobile attaches (same pane)
	if err := tracker.SaveAndResize(paneID, 50, 15); err != nil {
		t.Fatalf("SaveAndResize 2: %v", err)
	}

	// First mobile detaches — should NOT restore (one still attached)
	if err := tracker.RestoreIfLast(paneID); err != nil {
		t.Fatalf("RestoreIfLast 1: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Verify size is NOT original (still resized for mobile)
	size, err := tracker.getCurrentSize(paneID)
	if err != nil {
		t.Fatalf("getCurrentSize after first detach: %v", err)
	}
	// Should be 50x15 (last resize) not original
	if size.Cols == originalSize.Cols && size.Rows == originalSize.Rows {
		t.Error("size should NOT be original after first detach (one mobile still attached)")
	}

	// Second mobile detaches — should restore
	if err := tracker.RestoreIfLast(paneID); err != nil {
		t.Fatalf("RestoreIfLast 2: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Verify size is restored to original
	size, err = tracker.getCurrentSize(paneID)
	if err != nil {
		t.Fatalf("getCurrentSize after last detach: %v", err)
	}
	if size.Cols != originalSize.Cols {
		t.Errorf("cols after restore = %d, want %d", size.Cols, originalSize.Cols)
	}
	if size.Rows != originalSize.Rows {
		t.Errorf("rows after restore = %d, want %d", size.Rows, originalSize.Rows)
	}
}

func TestPaneSizeTracker_RestoreWhenPaneKilled(t *testing.T) {
	skipIfNoTmux(t)
	tc := testClient(t)

	_, err := tc.CreateSession("killed-pane-test", "")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	// Create a second session so server stays alive after we kill the first
	_, err = tc.CreateSession("keep-alive", "")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	sessions, err := tc.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	// Find the killed-pane-test session's pane
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
	if err := tracker.SaveAndResize(paneID, 40, 12); err != nil {
		t.Fatalf("SaveAndResize: %v", err)
	}

	// Kill the session (pane no longer exists)
	if err := tc.KillSession("killed-pane-test"); err != nil {
		t.Fatalf("KillSession: %v", err)
	}

	// RestoreIfLast should not error when pane is gone
	if err := tracker.RestoreIfLast(paneID); err != nil {
		t.Errorf("RestoreIfLast on killed pane should not error, got: %v", err)
	}
}

package tmux

import (
	"fmt"
	"sync"
)

// PaneSizeTracker tracks mobile attach counts per pane and auto-restores
// the window size when the last mobile client detaches.
type PaneSizeTracker struct {
	mu          sync.Mutex
	attachCount map[string]int // paneID -> number of mobiles attached
	client      *Client
}

// NewPaneSizeTracker creates a tracker that uses the given tmux client.
func NewPaneSizeTracker(client *Client) *PaneSizeTracker {
	return &PaneSizeTracker{
		attachCount: make(map[string]int),
		client:      client,
	}
}

// TrackAndResize increments the attach count for a pane and resizes
// the pane's window to the given mobile dimensions.
func (t *PaneSizeTracker) TrackAndResize(paneID string, cols, rows int) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.attachCount[paneID]++

	windowTarget, err := t.client.windowForPane(paneID)
	if err != nil {
		return fmt.Errorf("find window for pane: %w", err)
	}
	return t.client.ResizeWindow(windowTarget, cols, rows)
}

// RestoreIfLast decrements the attach count and auto-resizes the window
// when the last mobile client detaches. Uses resize-window -A to let tmux
// adjust the window to the current terminal client's size.
// Returns nil if the pane was already cleaned up or no longer exists.
func (t *PaneSizeTracker) RestoreIfLast(paneID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	count, ok := t.attachCount[paneID]
	if !ok || count <= 0 {
		return nil
	}

	t.attachCount[paneID]--
	if t.attachCount[paneID] > 0 {
		return nil // Other mobiles still attached
	}

	delete(t.attachCount, paneID)

	windowTarget, err := t.client.windowForPane(paneID)
	if err != nil {
		// Pane may have been killed — skip restore
		return nil
	}
	return t.client.ResizeWindowAuto(windowTarget)
}

package tmux

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// PaneSize represents the width and height of a tmux pane.
type PaneSize struct {
	Cols int
	Rows int
}

// PaneSizeTracker saves original pane sizes before mobile resize and
// restores them when the last mobile client detaches.
type PaneSizeTracker struct {
	mu          sync.Mutex
	savedSizes  map[string]PaneSize // paneID -> original size
	attachCount map[string]int      // paneID -> number of mobiles attached
	client      *Client
}

// NewPaneSizeTracker creates a tracker that uses the given tmux client.
func NewPaneSizeTracker(client *Client) *PaneSizeTracker {
	return &PaneSizeTracker{
		savedSizes:  make(map[string]PaneSize),
		attachCount: make(map[string]int),
		client:      client,
	}
}

// SaveAndResize saves the current pane size (on first attach) and resizes
// the pane's window to the given dimensions.
func (t *PaneSizeTracker) SaveAndResize(paneID string, cols, rows int) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.attachCount[paneID] == 0 {
		// First attach — save original size
		size, err := t.getCurrentSize(paneID)
		if err != nil {
			return fmt.Errorf("save original size: %w", err)
		}
		t.savedSizes[paneID] = size
	}
	t.attachCount[paneID]++

	// Resize to mobile dimensions
	windowTarget, err := t.client.windowForPane(paneID)
	if err != nil {
		return fmt.Errorf("find window for pane: %w", err)
	}
	return t.client.ResizeWindow(windowTarget, cols, rows)
}

// RestoreIfLast decrements the attach count and restores the original
// pane size when the last mobile client detaches. Returns nil if the
// pane was already cleaned up or no longer exists.
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

	// Last mobile detached — restore original size
	original, ok := t.savedSizes[paneID]
	if !ok {
		return nil
	}

	delete(t.savedSizes, paneID)
	delete(t.attachCount, paneID)

	windowTarget, err := t.client.windowForPane(paneID)
	if err != nil {
		// Pane may have been killed — skip restore
		return nil
	}
	return t.client.ResizeWindow(windowTarget, original.Cols, original.Rows)
}

// getCurrentSize reads the current pane dimensions via tmux display-message.
func (t *PaneSizeTracker) getCurrentSize(paneID string) (PaneSize, error) {
	out, err := t.client.run("display-message", "-t", paneID, "-p", "#{pane_width}x#{pane_height}")
	if err != nil {
		return PaneSize{}, fmt.Errorf("get pane size: %w: %s", err, out)
	}

	parts := strings.SplitN(strings.TrimSpace(out), "x", 2)
	if len(parts) != 2 {
		return PaneSize{}, fmt.Errorf("unexpected pane size format: %q", out)
	}

	cols, err := strconv.Atoi(parts[0])
	if err != nil {
		return PaneSize{}, fmt.Errorf("parse pane width: %w", err)
	}
	rows, err := strconv.Atoi(parts[1])
	if err != nil {
		return PaneSize{}, fmt.Errorf("parse pane height: %w", err)
	}

	return PaneSize{Cols: cols, Rows: rows}, nil
}

//go:build integration

package integration

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shiftinbits/pmux-agent/internal/agent"
	"github.com/shiftinbits/pmux-agent/internal/protocol"
	"github.com/shiftinbits/pmux-agent/internal/tmux"
)

const multiSocket = "pmux-integ-multi"

// TestMultiConnection_IsolatedPanes verifies that two peers can attach to
// different panes simultaneously with isolated I/O (no cross-talk).
func TestMultiConnection_IsolatedPanes(t *testing.T) {
	skipIfNoTmux(t)
	cleanupTmuxServer(t, multiSocket)

	tc := tmux.NewClient(multiSocket)
	catcher := &messageCatcher{}
	h := agent.NewHandler(tc, catcher.Send, func(data []byte) {}, newTestLogger())

	// Create two sessions (each with one pane)
	_, err := tc.CreateSession("multi-1", "")
	if err != nil {
		t.Fatalf("CreateSession 1: %v", err)
	}
	_, err = tc.CreateSession("multi-2", "")
	if err != nil {
		t.Fatalf("CreateSession 2: %v", err)
	}

	sessions, err := tc.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(sessions) < 2 {
		t.Fatalf("expected at least 2 sessions, got %d", len(sessions))
	}

	pane1 := sessions[0].Windows[0].Panes[0].ID
	pane2 := sessions[1].Windows[0].Panes[0].ID

	// Peer A attaches to pane 1
	h.HandleMessage("peerA", &protocol.AttachRequest{
		Type:   "attach",
		PaneID: pane1,
		Cols:   80,
		Rows:   24,
	})
	catcher.waitForPeer(t, "peerA", "attached", 3*time.Second)

	// Peer B attaches to pane 2
	h.HandleMessage("peerB", &protocol.AttachRequest{
		Type:   "attach",
		PaneID: pane2,
		Cols:   80,
		Rows:   24,
	})
	catcher.waitForPeer(t, "peerB", "attached", 3*time.Second)

	// Send unique markers from each peer
	markerA := fmt.Sprintf("MARKER_A_%d", time.Now().UnixNano())
	markerB := fmt.Sprintf("MARKER_B_%d", time.Now().UnixNano())

	catcher.reset()
	h.HandleMessage("peerA", &protocol.InputRequest{
		Type: "input",
		Data: []byte("echo " + markerA + "\n"),
	})
	h.HandleMessage("peerB", &protocol.InputRequest{
		Type: "input",
		Data: []byte("echo " + markerB + "\n"),
	})

	// Wait for output from both peers
	deadline := time.Now().Add(5 * time.Second)
	foundA := false
	foundB := false
	for time.Now().Before(deadline) {
		msgs := catcher.get()
		for _, m := range msgs {
			if out, ok := m.Msg.(*protocol.OutputEvent); ok {
				data := string(out.Data)
				if m.PeerID == "peerA" && strings.Contains(data, markerA) {
					foundA = true
				}
				if m.PeerID == "peerB" && strings.Contains(data, markerB) {
					foundB = true
				}
			}
		}
		if foundA && foundB {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if !foundA {
		t.Error("peerA did not receive its output marker")
	}
	if !foundB {
		t.Error("peerB did not receive its output marker")
	}

	// Verify NO cross-talk: peerA should not see markerB and vice versa
	msgs := catcher.get()
	for _, m := range msgs {
		if out, ok := m.Msg.(*protocol.OutputEvent); ok {
			data := string(out.Data)
			if m.PeerID == "peerA" && strings.Contains(data, markerB) {
				t.Error("cross-talk detected: peerA received peerB's output")
			}
			if m.PeerID == "peerB" && strings.Contains(data, markerA) {
				t.Error("cross-talk detected: peerB received peerA's output")
			}
		}
	}

	// Cleanup
	h.HandleMessage("peerA", &protocol.DetachRequest{Type: "detach"})
	h.HandleMessage("peerB", &protocol.DetachRequest{Type: "detach"})
	h.PeerDisconnected("peerA")
	h.PeerDisconnected("peerB")
}

// TestMultiConnection_DisconnectOneKeepsOther verifies that disconnecting one
// peer does not affect another peer's session.
func TestMultiConnection_DisconnectOneKeepsOther(t *testing.T) {
	skipIfNoTmux(t)
	cleanupTmuxServer(t, multiSocket)

	tc := tmux.NewClient(multiSocket)
	catcher := &messageCatcher{}
	h := agent.NewHandler(tc, catcher.Send, func(data []byte) {}, newTestLogger())

	// Create two sessions
	_, err := tc.CreateSession("disconnect-1", "")
	if err != nil {
		t.Fatalf("CreateSession 1: %v", err)
	}
	_, err = tc.CreateSession("disconnect-2", "")
	if err != nil {
		t.Fatalf("CreateSession 2: %v", err)
	}

	sessions, err := tc.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}

	pane1 := sessions[0].Windows[0].Panes[0].ID
	pane2 := sessions[1].Windows[0].Panes[0].ID

	// Both peers attach
	h.HandleMessage("peerA", &protocol.AttachRequest{
		Type:   "attach",
		PaneID: pane1,
		Cols:   80,
		Rows:   24,
	})
	catcher.waitForPeer(t, "peerA", "attached", 3*time.Second)

	h.HandleMessage("peerB", &protocol.AttachRequest{
		Type:   "attach",
		PaneID: pane2,
		Cols:   80,
		Rows:   24,
	})
	catcher.waitForPeer(t, "peerB", "attached", 3*time.Second)

	// Disconnect peer A
	h.PeerDisconnected("peerA")

	// Peer B should still be able to send input and get output
	marker := fmt.Sprintf("SURVIVOR_%d", time.Now().UnixNano())
	catcher.reset()
	h.HandleMessage("peerB", &protocol.InputRequest{
		Type: "input",
		Data: []byte("echo " + marker + "\n"),
	})

	deadline := time.Now().Add(5 * time.Second)
	found := false
	for time.Now().Before(deadline) {
		msgs := catcher.get()
		for _, m := range msgs {
			if m.PeerID == "peerB" {
				if out, ok := m.Msg.(*protocol.OutputEvent); ok {
					if strings.Contains(string(out.Data), marker) {
						found = true
						break
					}
				}
			}
		}
		if found {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if !found {
		t.Error("peerB should still receive output after peerA disconnects")
	}

	// Peer A should get an error if it tries to send input
	catcher.reset()
	h.HandleMessage("peerA", &protocol.InputRequest{
		Type: "input",
		Data: []byte("should fail"),
	})
	errMsg := catcher.waitForPeer(t, "peerA", "error", 2*time.Second)
	errEvt, ok := errMsg.(*protocol.ErrorEvent)
	if !ok {
		t.Fatalf("expected ErrorEvent, got %T", errMsg)
	}
	if errEvt.Code != "not_attached" {
		t.Errorf("error code = %q, want not_attached", errEvt.Code)
	}

	// Cleanup
	h.HandleMessage("peerB", &protocol.DetachRequest{Type: "detach"})
	h.PeerDisconnected("peerB")
}

// TestMultiConnection_AllCleanup verifies that disconnecting all peers
// results in complete cleanup with no leaked state.
func TestMultiConnection_AllCleanup(t *testing.T) {
	skipIfNoTmux(t)
	cleanupTmuxServer(t, multiSocket)

	tc := tmux.NewClient(multiSocket)
	catcher := &messageCatcher{}
	h := agent.NewHandler(tc, catcher.Send, func(data []byte) {}, newTestLogger())

	// Create sessions
	_, err := tc.CreateSession("cleanup-1", "")
	if err != nil {
		t.Fatalf("CreateSession 1: %v", err)
	}
	_, err = tc.CreateSession("cleanup-2", "")
	if err != nil {
		t.Fatalf("CreateSession 2: %v", err)
	}
	_, err = tc.CreateSession("cleanup-3", "")
	if err != nil {
		t.Fatalf("CreateSession 3: %v", err)
	}

	sessions, err := tc.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}

	// Three peers attach to three different panes
	for i := 0; i < 3; i++ {
		peerID := fmt.Sprintf("peer%d", i)
		paneID := sessions[i].Windows[0].Panes[0].ID
		h.HandleMessage(peerID, &protocol.AttachRequest{
			Type:   "attach",
			PaneID: paneID,
			Cols:   80,
			Rows:   24,
		})
		catcher.waitForPeer(t, peerID, "attached", 3*time.Second)
	}

	// Send input from all peers to exercise streaming goroutines
	for i := 0; i < 3; i++ {
		peerID := fmt.Sprintf("peer%d", i)
		h.HandleMessage(peerID, &protocol.InputRequest{
			Type: "input",
			Data: []byte("echo hello\n"),
		})
	}
	time.Sleep(300 * time.Millisecond)

	// Disconnect peer0, then peer1
	h.PeerDisconnected("peer0")
	h.PeerDisconnected("peer1")

	// peer2 should still work
	marker := fmt.Sprintf("LAST_%d", time.Now().UnixNano())
	catcher.reset()
	h.HandleMessage("peer2", &protocol.InputRequest{
		Type: "input",
		Data: []byte("echo " + marker + "\n"),
	})

	deadline := time.Now().Add(5 * time.Second)
	found := false
	for time.Now().Before(deadline) {
		msgs := catcher.get()
		for _, m := range msgs {
			if m.PeerID == "peer2" {
				if out, ok := m.Msg.(*protocol.OutputEvent); ok {
					if strings.Contains(string(out.Data), marker) {
						found = true
						break
					}
				}
			}
		}
		if found {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !found {
		t.Error("last remaining peer should still receive output")
	}

	// Disconnect the last peer
	h.PeerDisconnected("peer2")

	// Verify all peers get not_attached error if they try to send
	catcher.reset()
	for i := 0; i < 3; i++ {
		peerID := fmt.Sprintf("peer%d", i)
		h.HandleMessage(peerID, &protocol.InputRequest{
			Type: "input",
			Data: []byte("should fail"),
		})
		errMsg := catcher.waitForPeer(t, peerID, "error", 2*time.Second)
		errEvt, ok := errMsg.(*protocol.ErrorEvent)
		if !ok {
			t.Fatalf("peer%d: expected ErrorEvent, got %T", i, errMsg)
		}
		if errEvt.Code != "not_attached" {
			t.Errorf("peer%d: error code = %q, want not_attached", i, errEvt.Code)
		}
	}
}

// TestMultiConnection_ConcurrentListSessions verifies that multiple peers
// can list sessions concurrently without races.
func TestMultiConnection_ConcurrentListSessions(t *testing.T) {
	skipIfNoTmux(t)
	cleanupTmuxServer(t, multiSocket)

	tc := tmux.NewClient(multiSocket)
	catcher := &messageCatcher{}
	h := agent.NewHandler(tc, catcher.Send, func(data []byte) {}, newTestLogger())

	// Create a few sessions
	for i := 0; i < 3; i++ {
		_, err := tc.CreateSession(fmt.Sprintf("concurrent-%d", i), "")
		if err != nil {
			t.Fatalf("CreateSession %d: %v", i, err)
		}
	}

	// Multiple peers list sessions truly concurrently
	const numPeers = 5
	var wg sync.WaitGroup
	for i := 0; i < numPeers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			peerID := fmt.Sprintf("concurrent-peer-%d", idx)
			h.HandleMessage(peerID, &protocol.ListSessionsRequest{Type: "list_sessions"})
		}(i)
	}
	wg.Wait()

	// Wait for all responses
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		count := 0
		msgs := catcher.get()
		for _, m := range msgs {
			if m.Msg.MessageType() == "sessions" {
				count++
			}
		}
		if count >= numPeers {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Verify each peer got a sessions event
	msgs := catcher.get()
	peerResponses := make(map[string]bool)
	for _, m := range msgs {
		if m.Msg.MessageType() == "sessions" {
			peerResponses[m.PeerID] = true
			sessionsEvt := m.Msg.(*protocol.SessionsEvent)
			if len(sessionsEvt.Sessions) != 3 {
				t.Errorf("peer %s: expected 3 sessions, got %d", m.PeerID, len(sessionsEvt.Sessions))
			}
		}
	}

	for i := 0; i < numPeers; i++ {
		peerID := fmt.Sprintf("concurrent-peer-%d", i)
		if !peerResponses[peerID] {
			t.Errorf("peer %s did not receive sessions response", peerID)
		}
	}
}

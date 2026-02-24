// Package agent implements the core PocketMux agent lifecycle: start, connect, shutdown.
package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/shiftinbits/pmux-agent/internal/auth"
	"github.com/shiftinbits/pmux-agent/internal/config"
	"github.com/shiftinbits/pmux-agent/internal/protocol"
	"github.com/shiftinbits/pmux-agent/internal/tmux"
	"github.com/shiftinbits/pmux-agent/internal/webrtc"
)

const (
	// tmuxStartTimeout is how long the agent waits for the tmux server to start.
	tmuxStartTimeout = 30 * time.Second
	// tmuxPollInterval is how often the agent checks if the tmux server is alive.
	tmuxPollInterval = 5 * time.Second
)

// Run starts the PocketMux agent. It connects to the signaling server,
// handles WebRTC connections, and monitors the tmux server.
// Blocks until the context is canceled or the tmux server exits.
func Run(ctx context.Context, paths config.Paths) error {
	// Set up file logging
	logFile := filepath.Join(paths.ConfigDir, "agent.log")
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()

	logger := slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	logger.Info("agent starting", "pid", os.Getpid())

	// Load identity
	identity, err := auth.LoadIdentity(paths.KeysDir)
	if err != nil {
		return fmt.Errorf("load identity: %w", err)
	}
	logger.Info("identity loaded", "deviceID", identity.DeviceID)

	// Create tmux client targeting the pmux socket
	tmuxClient := tmux.NewClient(tmux.DefaultSocket)

	// TODO: read server URL from config file
	serverURL := config.DefaultServerURL

	// Create components with forward references (resolved via closures)
	var peerManager *webrtc.PeerManager

	handler := NewHandler(tmuxClient, func(peerID string, msg protocol.Message) error {
		return peerManager.SendTo(peerID, msg)
	}, logger)

	signalingClient := webrtc.NewSignalingClient(identity, serverURL, func(msg webrtc.SignalingMessage) {
		peerManager.HandleSignalingMessage(msg)
	}, logger)

	peerManager = webrtc.NewPeerManager(
		logger,
		signalingClient,
		serverURL,
		signalingClient.JWT,
		handler.HandleMessage,
	)

	// Create a cancelable context for the agent
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Watch for tmux server lifecycle
	go watchTmux(ctx, cancel, tmuxClient, logger)

	// Run signaling client (blocks until context is canceled)
	logger.Info("connecting to signaling server", "url", serverURL)
	err = signalingClient.Run(ctx)

	// Cleanup
	logger.Info("agent shutting down")
	peerManager.CloseAll()
	signalingClient.Close()
	RemovePIDFile(paths)

	if err != nil && err != context.Canceled {
		return err
	}
	return nil
}

// watchTmux monitors the tmux server on the pmux socket.
// It waits for the server to start, then monitors until it exits.
// When the tmux server exits, it cancels the agent context.
func watchTmux(ctx context.Context, cancel context.CancelFunc, tc *tmux.Client, logger *slog.Logger) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Wait for tmux server to start (it may start after the agent)
	startDeadline := time.After(tmuxStartTimeout)
	started := false
	for !started {
		select {
		case <-ctx.Done():
			return
		case <-startDeadline:
			logger.Info("tmux server did not start within timeout, shutting down")
			cancel()
			return
		case <-ticker.C:
			if tc.IsServerRunning() {
				started = true
				logger.Info("tmux server detected")
			}
		}
	}

	// Stop the startup ticker before switching to monitoring interval
	ticker.Stop()

	// Monitor for tmux server exit
	monitorTicker := time.NewTicker(tmuxPollInterval)
	defer monitorTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-monitorTicker.C:
			if !tc.IsServerRunning() {
				logger.Info("tmux server exited, shutting down agent")
				cancel()
				return
			}
		}
	}
}

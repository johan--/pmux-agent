# CLAUDE.md — pmux-agent

Go agent binary for PocketMux. `pmux` is a transparent tmux wrapper.

## Key Rules

- Go module: `github.com/shiftinbits/pmux-agent`
- Build: `go build -o bin/pmux ./cmd/pmux`
- `pmux` is a tmux wrapper — dedicated `-L pmux` socket isolates from regular tmux
- Command routing: intercept `init`, `pair`, `config`, `status`, `unpair`, `agent`, `--version`; passthrough everything else to `tmux -L pmux`
- Agent runs as background process tied to tmux server lifecycle
- Standard Go conventions: `gofmt`, `go vet`, error wrapping with `fmt.Errorf("context: %w", err)`
- Structured logging with `slog`
- Table-driven tests
- No global mutable state — pass dependencies via constructor/params

## Architecture

- `cmd/pmux/main.go` — CLI entry point, command routing
- `internal/agent/` — Core agent lifecycle, supervisor, handler, status, unpair, cleanup
- `internal/auth/` — Ed25519 identity, JWT signing, X25519 pairing, secret storage
- `internal/config/` — TOML config file parsing, env overrides
- `internal/protocol/` — MessagePack codec, message types
- `internal/proxy/` — tmux passthrough (syscall.Exec)
- `internal/service/` — OS service management (launchd on macOS, systemd on Linux)
- `internal/tmux/` — tmux CLI wrapper, PTY bridge, pane size tracking
- `internal/webrtc/` — Pion WebRTC, signaling client, dormancy

## Dependencies

- `pion/webrtc/v4` — WebRTC DataChannels
- `vmihailenco/msgpack/v5` — Wire protocol codec
- `creack/pty` — PTY management for pane attachment
- `gorilla/websocket` — Signaling WebSocket client
- `skip2/go-qrcode` — QR code for pairing
- `pelletier/go-toml/v2` — Config file parsing

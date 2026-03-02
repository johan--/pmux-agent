# pmux — PocketMux Agent

`pmux` is a transparent drop-in replacement for `tmux` that makes your terminal sessions accessible from your phone via the [PocketMux](https://github.com/ShiftinBits/pocketmux) mobile app.

## How It Works

Replace `tmux` with `pmux` in your workflow. Every tmux command works identically, but sessions started with `pmux` are automatically accessible from the PocketMux mobile app over an encrypted peer-to-peer WebRTC connection.

```bash
pmux                          # start new session (like tmux)
pmux new-session -s work      # named session
pmux attach -t work           # attach
pmux ls                       # list sessions
pmux split-window -h          # split pane — every tmux command works
```

`pmux` uses a dedicated tmux socket (`-L pmux`) to keep PocketMux sessions separate from your regular tmux sessions.

## PocketMux-Only Commands

```bash
pmux init       # one-time: generate identity, install service
pmux pair       # pair with mobile device (displays QR code)
pmux status     # show agent, service, and pairing status
pmux config     # show effective configuration with sources
pmux unpair     # remove paired mobile device
pmux agent      # manage the background agent (start/stop/install/uninstall)
```

## Installation

### From Source

```bash
go install github.com/shiftinbits/pmux-agent/cmd/pmux@latest
```

### Homebrew

```bash
brew install shiftinbits/tap/pmux
```

### Binary Download

Pre-built binaries available on [GitHub Releases](https://github.com/ShiftinBits/pmux-agent/releases).

## Requirements

- tmux 3.x+ installed and in PATH
- macOS (arm64, amd64) or Linux (arm64, amd64)

## How It Works (Technical)

1. `pmux` forwards all commands to `tmux -L pmux` (dedicated socket)
2. When a tmux server starts, `pmux` launches a background WebRTC agent
3. The agent connects to the PocketMux signaling server and accepts connections from paired mobile devices
4. All terminal data flows peer-to-peer over encrypted WebRTC DataChannels — the server never sees your terminal content

## License

MIT — see [LICENSE](./LICENSE)

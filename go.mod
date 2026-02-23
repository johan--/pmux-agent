module github.com/shiftinbits/pmux-agent

go 1.22

// Dependencies to add with Go 1.22+ toolchain:
//   github.com/pion/webrtc/v4         — WebRTC DataChannels (requires Go 1.22+)
//   github.com/vmihailenco/msgpack/v5 — MessagePack wire protocol codec
//   github.com/creack/pty             — PTY management for pane attachment
//   github.com/gorilla/websocket      — WebSocket signaling client
//   github.com/skip2/go-qrcode        — QR code generation for pairing
//   github.com/pelletier/go-toml/v2   — TOML config file parsing

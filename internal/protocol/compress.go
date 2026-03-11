package protocol

import (
	"bytes"
	"compress/flate"
	"fmt"
)

// OutputCompressor wraps a stateful flate.Writer for compressing terminal
// output across multiple messages. The LZ77 sliding window persists between
// Compress() calls, allowing repeated patterns (ANSI escapes, shell prompts,
// common paths) to compress increasingly well over time.
type OutputCompressor struct {
	w   *flate.Writer
	buf bytes.Buffer
}

// NewOutputCompressor creates a new stateful compressor using deflate level 6.
func NewOutputCompressor() *OutputCompressor {
	c := &OutputCompressor{}
	// flate.NewWriter with a bytes.Buffer never returns an error for valid levels.
	c.w, _ = flate.NewWriter(&c.buf, flate.DefaultCompression)
	return c
}

// Compress compresses data through the stateful deflate stream.
// Uses sync flush to emit a sync marker so the decompressor can produce
// output immediately without waiting for more data. Returns a copy of
// the compressed bytes (the internal buffer is reused across calls).
func (c *OutputCompressor) Compress(data []byte) ([]byte, error) {
	c.buf.Reset()
	if _, err := c.w.Write(data); err != nil {
		return nil, fmt.Errorf("compress write: %w", err)
	}
	if err := c.w.Flush(); err != nil {
		return nil, fmt.Errorf("compress flush: %w", err)
	}
	// Return a copy — buf is reused on next Compress() call
	return append([]byte(nil), c.buf.Bytes()...), nil
}

// Close releases resources held by the compressor.
func (c *OutputCompressor) Close() error {
	return c.w.Close()
}

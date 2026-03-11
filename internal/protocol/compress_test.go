package protocol

import (
	"bytes"
	"compress/flate"
	"io"
	"strings"
	"testing"
)

// decompressSyncFlushed decompresses data produced by a sync-flushed deflate
// stream. A sync flush does not emit a final block marker, so flate.NewReader
// will return io.ErrUnexpectedEOF after consuming all available data. We treat
// that as a successful read since the sync point guarantees all input has been
// flushed to the output.
func decompressSyncFlushed(t *testing.T, compressed []byte) []byte {
	t.Helper()
	r := flate.NewReader(bytes.NewReader(compressed))
	defer r.Close()
	out, err := io.ReadAll(r)
	if err != nil && err != io.ErrUnexpectedEOF {
		t.Fatalf("decompress failed: %v", err)
	}
	return out
}

func TestOutputCompressor_RoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"short string", "hello, world!"},
		{"repeated pattern", strings.Repeat("abcdef", 50)},
		{"binary-like", string([]byte{0x00, 0x01, 0xFF, 0xFE, 0x80, 0x7F})},
		{"unicode", "hello \xe2\x9c\x93 world \xc3\xa9\xc3\xa8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewOutputCompressor()
			defer c.Close()

			compressed, err := c.Compress([]byte(tt.input))
			if err != nil {
				t.Fatalf("Compress: %v", err)
			}

			got := decompressSyncFlushed(t, compressed)
			if string(got) != tt.input {
				t.Errorf("round-trip mismatch: got %q, want %q", got, tt.input)
			}
		})
	}
}

func TestOutputCompressor_StatefulDictionary(t *testing.T) {
	c := NewOutputCompressor()
	defer c.Close()

	// A 200-byte string with repeated patterns
	data := []byte(strings.Repeat("user@host:~/dev$ ls -la\n", 9)) // 216 bytes

	first, err := c.Compress(data)
	if err != nil {
		t.Fatalf("first Compress: %v", err)
	}

	second, err := c.Compress(data)
	if err != nil {
		t.Fatalf("second Compress: %v", err)
	}

	t.Logf("first compressed size: %d, second: %d (original: %d)", len(first), len(second), len(data))

	if len(second) >= len(first) {
		t.Errorf("expected second compression (%d bytes) to be smaller than first (%d bytes) due to stateful dictionary",
			len(second), len(first))
	}

	// Verify both decompress correctly — need a stateful decompressor
	// that mirrors the compressor's state. Sync-flushed stream may return
	// io.ErrUnexpectedEOF after all data is consumed.
	combined := append(first, second...)
	r := flate.NewReader(bytes.NewReader(combined))
	defer r.Close()

	allOut, err := io.ReadAll(r)
	if err != nil && err != io.ErrUnexpectedEOF {
		t.Fatalf("decompress: %v", err)
	}

	expected := append(data, data...)
	if !bytes.Equal(allOut, expected) {
		t.Error("decompressed stateful data does not match")
	}
}

func TestOutputCompressor_TerminalData(t *testing.T) {
	c := NewOutputCompressor()
	defer c.Close()

	// Concatenate all compressed output into a single stream for the
	// stateful decompressor (mirrors how a real client would receive it).
	sequences := []string{
		"\x1b[0m\x1b[1;32muser@host\x1b[0m:\x1b[1;34m~/dev\x1b[0m$ ",
		"ls -la\r\n",
		"\x1b[0mtotal 42\r\n-rw-r--r-- 1 user staff 1234 Jan  1 00:00 file.txt\r\n",
		"\x1b[0m\x1b[1;32muser@host\x1b[0m:\x1b[1;34m~/dev\x1b[0m$ ",
		"\x1b[0m\x1b[1;32muser@host\x1b[0m:\x1b[1;34m~/dev\x1b[0m$ ",
	}

	var allCompressed []byte
	for _, seq := range sequences {
		compressed, err := c.Compress([]byte(seq))
		if err != nil {
			t.Fatalf("Compress(%q): %v", seq, err)
		}
		allCompressed = append(allCompressed, compressed...)
	}

	// Decompress the full stream and verify concatenated output matches.
	// Sync-flushed streams don't have a final block, so io.ErrUnexpectedEOF
	// is expected after all data is consumed.
	r := flate.NewReader(bytes.NewReader(allCompressed))
	defer r.Close()

	allDecompressed, err := io.ReadAll(r)
	if err != nil && err != io.ErrUnexpectedEOF {
		t.Fatalf("decompress: %v", err)
	}

	expected := strings.Join(sequences, "")
	if string(allDecompressed) != expected {
		t.Errorf("decompressed terminal data does not match\ngot:  %q\nwant: %q",
			string(allDecompressed), expected)
	}
}

func TestOutputCompressor_EmptyInput(t *testing.T) {
	c := NewOutputCompressor()
	defer c.Close()

	compressed, err := c.Compress([]byte{})
	if err != nil {
		t.Fatalf("Compress(empty): %v", err)
	}

	// Should return valid compressed bytes (sync flush marker at minimum)
	if compressed == nil {
		t.Error("expected non-nil compressed output for empty input")
	}

	// Should be decompressible (produces empty output).
	// Sync-flushed stream may return io.ErrUnexpectedEOF — that's expected.
	r := flate.NewReader(bytes.NewReader(compressed))
	defer r.Close()
	got, err := io.ReadAll(r)
	if err != nil && err != io.ErrUnexpectedEOF {
		t.Fatalf("decompress empty: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty decompressed output, got %d bytes", len(got))
	}
}

func TestOutputCompressor_SafeCopy(t *testing.T) {
	c := NewOutputCompressor()
	defer c.Close()

	data1 := []byte("first message with some content")
	result1, err := c.Compress(data1)
	if err != nil {
		t.Fatalf("Compress(data1): %v", err)
	}

	// Save a copy of result1 for comparison
	saved := make([]byte, len(result1))
	copy(saved, result1)

	data2 := []byte("second completely different message")
	_, err = c.Compress(data2)
	if err != nil {
		t.Fatalf("Compress(data2): %v", err)
	}

	// result1 should be unchanged — it's a copy, not aliased to internal buffer
	if !bytes.Equal(result1, saved) {
		t.Error("first Compress result was mutated by second Compress call — buffer aliasing detected")
	}
}

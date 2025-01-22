package lfs

import (
	"bytes"
	"io"
	"testing"
)

func TestBinaryDiff(t *testing.T) {
	t.Run("Small Changes", func(t *testing.T) {
		// Original content
		oldData := []byte("Hello, this is a test file for binary diff!")
		// Modified content (changed one word)
		newData := []byte("Hello, this is a sample file for binary diff!")

		// Generate diff
		diff, err := BinaryDiff(bytes.NewReader(oldData), bytes.NewReader(newData))
		if err != nil {
			t.Fatal(err)
		}

		// Apply diff
		var result bytes.Buffer
		if err := ApplyDiff(bytes.NewReader(oldData), diff, &result); err != nil {
			t.Fatal(err)
		}

		// Verify result
		if !bytes.Equal(result.Bytes(), newData) {
			t.Error("Diff application failed to reproduce new content")
		}
	})

	t.Run("Large Block Changes", func(t *testing.T) {
		// Create large test data
		oldData := make([]byte, 100*1024) // 100KB
		newData := make([]byte, 100*1024)

		// Fill with pattern
		for i := range oldData {
			oldData[i] = byte(i % 256)
			newData[i] = byte(i % 256)
		}

		// Modify a block in the middle
		copy(newData[50*1024:], bytes.Repeat([]byte("modified"), 1024))

		// Generate and apply diff
		diff, err := BinaryDiff(bytes.NewReader(oldData), bytes.NewReader(newData))
		if err != nil {
			t.Fatal(err)
		}

		var result bytes.Buffer
		if err := ApplyDiff(bytes.NewReader(oldData), diff, &result); err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(result.Bytes(), newData) {
			t.Error("Failed to reproduce large modified content")
		}
	})

	t.Run("Rolling Hash", func(t *testing.T) {
		rh := NewRollingHash()

		// Test with simple pattern
		data := []byte("abcdefghijklmnop")
		var hashes []uint32

		// Calculate rolling hash for each window
		for i := 0; i <= len(data)-RollingHashWindow; i++ {
			// Reset hash for new window
			rh = NewRollingHash()
			for j := 0; j < RollingHashWindow; j++ {
				rh.Update(data[i+j])
			}
			hashes = append(hashes, rh.hash)
		}

		// Verify we get different hashes for different windows
		seen := make(map[uint32]bool)
		for _, h := range hashes {
			if seen[h] {
				t.Error("Hash collision in rolling hash")
			}
			seen[h] = true
		}
	})

	t.Run("Empty Input", func(t *testing.T) {
		diff, err := BinaryDiff(bytes.NewReader([]byte{}), bytes.NewReader([]byte{}))
		if err != nil {
			t.Fatal(err)
		}
		if len(diff) != 0 {
			t.Error("Expected empty diff for empty input")
		}
	})

	t.Run("Append Content", func(t *testing.T) {
		oldData := []byte("Original content")
		newData := []byte("Original content with appended text")

		diff, err := BinaryDiff(bytes.NewReader(oldData), bytes.NewReader(newData))
		if err != nil {
			t.Fatal(err)
		}

		var result bytes.Buffer
		if err := ApplyDiff(bytes.NewReader(oldData), diff, &result); err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(result.Bytes(), newData) {
			t.Error("Failed to handle appended content")
		}
	})

	t.Run("Streaming Large Content", func(t *testing.T) {
		// Create large content that won't fit in memory
		oldReader := &infiniteReader{limit: 10 * 1024 * 1024} // 10MB
		newReader := &infiniteReader{limit: 10 * 1024 * 1024, modified: true}

		// Generate diff
		diff, err := BinaryDiff(oldReader, newReader)
		if err != nil {
			t.Fatal(err)
		}

		// Verify diff size is reasonable
		if len(diff) > 1024*1024 { // Should be much smaller than original
			t.Error("Diff size too large for similar content")
		}
	})
}

// infiniteReader generates predictable content for testing
type infiniteReader struct {
	pos      int64
	limit    int64
	modified bool
}

func (r *infiniteReader) Read(p []byte) (n int, err error) {
	if r.pos >= r.limit {
		return 0, io.EOF
	}

	for i := range p {
		if r.pos+int64(i) >= r.limit {
			return i, io.EOF
		}
		if r.modified && r.pos+int64(i) >= r.limit/2 {
			p[i] = byte((r.pos + int64(i)) % 251) // Different pattern
		} else {
			p[i] = byte((r.pos + int64(i)) % 250)
		}
	}
	r.pos += int64(len(p))
	return len(p), nil
}

package lfs

import (
	"bytes"
	"io"
)

const (
	// RollingHashWindow is the size of the rolling hash window
	RollingHashWindow = 64

	// MinMatchSize is the minimum size of a matching block
	MinMatchSize = 32
)

// RollingHash implements a simple rolling hash for binary diff
type RollingHash struct {
	window []byte
	pos    int
	hash   uint32
}

// NewRollingHash creates a new rolling hash
func NewRollingHash() *RollingHash {
	return &RollingHash{
		window: make([]byte, RollingHashWindow),
	}
}

// Update updates the rolling hash with a new byte
func (r *RollingHash) Update(b byte) uint32 {
	// Remove old byte's contribution
	old := r.window[r.pos]
	r.hash = (r.hash - uint32(old)) + uint32(b)

	// Add new byte
	r.window[r.pos] = b
	r.pos = (r.pos + 1) % RollingHashWindow

	return r.hash
}

// BinaryDiff generates a binary diff between two readers
func BinaryDiff(old, new io.Reader) ([]DiffEntry, error) {
	// Read old content into memory for efficient matching
	oldData, err := io.ReadAll(old)
	if err != nil {
		return nil, err
	}

	// Read new content into memory for efficient matching
	newData, err := io.ReadAll(new)
	if err != nil {
		return nil, err
	}

	// Initialize rolling hash
	rh := NewRollingHash()
	blockIndex := make(map[uint32][]int)

	// Build block index for old content
	if len(oldData) >= RollingHashWindow {
		for i := 0; i <= len(oldData)-RollingHashWindow; i++ {
			// Update rolling hash
			if i == 0 {
				for j := 0; j < RollingHashWindow && j < len(oldData); j++ {
					rh.Update(oldData[j])
				}
			} else if i+RollingHashWindow-1 < len(oldData) {
				rh.Update(oldData[i+RollingHashWindow-1])
			}
			hash := rh.hash

			// Store position for this hash
			blockIndex[hash] = append(blockIndex[hash], i)
		}
	}

	// Process new content to find matches
	var diff []DiffEntry
	newBuf := &bytes.Buffer{}
	pos := 0

	for pos < len(newData) {
		// Calculate rolling hash for current window
		rh = NewRollingHash()
		windowEnd := pos + RollingHashWindow
		if windowEnd > len(newData) {
			windowEnd = len(newData)
		}
		for i := pos; i < windowEnd; i++ {
			rh.Update(newData[i])
		}
		hash := rh.hash

		// Look for matches
		matched := false
		if positions, ok := blockIndex[hash]; ok {
			for _, oldPos := range positions {
				// Verify full match
				matchLen := 0
				for i := 0; i < MinMatchSize && pos+i < len(newData) && oldPos+i < len(oldData); i++ {
					if oldData[oldPos+i] != newData[pos+i] {
						break
					}
					matchLen++
				}

				if matchLen >= MinMatchSize {
					// Found a match, extend it
					for oldPos+matchLen < len(oldData) && pos+matchLen < len(newData) && 
						oldData[oldPos+matchLen] == newData[pos+matchLen] {
						matchLen++
					}

					// Add any pending new data
					if newBuf.Len() > 0 {
						diff = append(diff, DiffEntry{
							Type: DiffNew,
							Data: newBuf.Bytes(),
						})
						newBuf.Reset()
					}

					// Add the match
					diff = append(diff, DiffEntry{
						Type:     DiffCopy,
						Offset:   int64(oldPos),
						Length:   int64(matchLen),
					})

					pos += matchLen
					matched = true
					break
				}
			}
		}

		if !matched && pos < len(newData) {
			// No match found, add to new data buffer
			newBuf.WriteByte(newData[pos])
			pos++
		}
	}

	// Add any remaining new data
	if newBuf.Len() > 0 {
		diff = append(diff, DiffEntry{
			Type: DiffNew,
			Data: newBuf.Bytes(),
		})
	}

	return diff, nil
}

// DiffType represents the type of a diff entry
type DiffType byte

const (
	DiffCopy DiffType = iota // Copy from old file
	DiffNew                  // New data
)

// DiffEntry represents a single entry in a binary diff
type DiffEntry struct {
	Type   DiffType // Type of entry
	Offset int64    // Offset in old file (for Copy)
	Length int64    // Length to copy (for Copy)
	Data   []byte   // New data (for New)
}

// ApplyDiff applies a binary diff to generate new content
func ApplyDiff(old io.Reader, diff []DiffEntry, w io.Writer) error {
	// Read old content
	oldData, err := io.ReadAll(old)
	if err != nil {
		return err
	}

	// Apply diff entries
	for _, entry := range diff {
		switch entry.Type {
		case DiffCopy:
			// Copy from old file
			if entry.Offset+entry.Length > int64(len(oldData)) {
				return io.ErrUnexpectedEOF
			}
			if _, err := w.Write(oldData[entry.Offset:entry.Offset+entry.Length]); err != nil {
				return err
			}
		case DiffNew:
			// Write new data
			if _, err := w.Write(entry.Data); err != nil {
				return err
			}
		}
	}

	return nil
}

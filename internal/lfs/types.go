package lfs

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"time"
)

const (
	// ChunkSize is the size of each chunk in bytes (1MB)
	ChunkSize = 1024 * 1024
)

// FileInfo contains metadata about a stored file
type FileInfo struct {
	ID          string      `json:"id"`          // Unique file identifier
	Size        int64       `json:"size"`        // Total file size in bytes
	ContentHash string      `json:"contentHash"` // Hash of entire file content
	NumChunks   int         `json:"numChunks"`   // Number of chunks
	Chunks      []ChunkInfo `json:"chunks"`      // List of chunks
	RefCount    int         `json:"refCount"`    // Number of references to this file
	Created     time.Time   `json:"created"`     // When the file was created
}

// ChunkInfo contains metadata about a file chunk
type ChunkInfo struct {
	Hash string `json:"hash"` // Hash of chunk content
	Size int64  `json:"size"` // Size of chunk in bytes
}

// Hash represents a content-addressable hash
type Hash struct {
	h hash.Hash
}

// NewHash creates a new hash
func NewHash() *Hash {
	return &Hash{h: sha256.New()}
}

// Write implements io.Writer
func (h *Hash) Write(p []byte) (n int, err error) {
	return h.h.Write(p)
}

// Sum returns the hash as a hex string
func (h *Hash) Sum() string {
	return hex.EncodeToString(h.h.Sum(nil))
}

// HashBytes returns the hash of a byte slice
func HashBytes(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

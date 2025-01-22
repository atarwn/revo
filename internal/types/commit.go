package types

import (
	"crypto/sha256"
	"evo/internal/crdt"
	"time"
)

// ExtendedOp includes oldContent for update ops
type ExtendedOp struct {
	Op         crdt.Operation `json:"op"`
	OldContent string         `json:"oldContent,omitempty"`
}

// Commit represents a set of operations with metadata
type Commit struct {
	ID          string    `json:"id"`
	Stream      string    `json:"stream"`
	Message     string    `json:"message"`
	AuthorName  string    `json:"authorName"`
	AuthorEmail string    `json:"authorEmail"`
	Timestamp   time.Time `json:"timestamp"`
	Signature   string    `json:"signature,omitempty"`

	Ops []ExtendedOp `json:"ops"`
}

// CommitHashString generates a stable string representation of a commit for signing
func CommitHashString(c *Commit) string {
	// stable representation => ID + stream + message + etc
	h := sha256.New()
	h.Write([]byte(c.ID))
	h.Write([]byte(c.Stream))
	h.Write([]byte(c.Message))
	h.Write([]byte(c.AuthorName))
	h.Write([]byte(c.AuthorEmail))
	h.Write([]byte(c.Timestamp.UTC().Format(time.RFC3339)))
	return string(h.Sum(nil))
}

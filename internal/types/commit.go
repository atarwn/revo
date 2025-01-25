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

// Commit represents a commit in the repository
type Commit struct {
	ID          string       // Unique identifier
	Stream      string       // Stream name
	Message     string       // Commit message
	AuthorName  string       // Author's name
	AuthorEmail string       // Author's email
	Timestamp   time.Time    // When the commit was created
	Operations  []ExtendedOp // Operations included in this commit
	Signature   string       // Optional Ed25519 signature
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

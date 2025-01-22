package crdt

import (
	"time"

	"github.com/google/uuid"
)

// OpType represents the type of operation
type OpType int

const (
	OpInsert OpType = iota
	OpUpdate
	OpDelete
)

// Operation represents a CRDT operation
type Operation struct {
	Type      OpType    // Type of operation
	Lamport   uint64    // Lamport timestamp for ordering
	NodeID    uuid.UUID // ID of the node that created this operation
	FileID    uuid.UUID // ID of the file being modified
	LineID    uuid.UUID // ID of the line being modified
	Content   string    // Content for insert/update operations
	Stream    string    // Stream this operation belongs to
	Timestamp time.Time // When the operation occurred
	Vector    []int64   // Vector clock for causal ordering
}

// CanCombine checks if two operations can be combined
func (o *Operation) CanCombine(other *Operation) bool {
	// Can only combine operations in same stream
	if o.Stream != other.Stream {
		return false
	}

	// Can only combine operations on same file
	if o.FileID != other.FileID {
		return false
	}

	// Can't combine deletes
	if o.Type == OpDelete || other.Type == OpDelete {
		return false
	}

	// Must be sequential in Lamport time
	return o.Lamport < other.Lamport
}

// Combine merges another operation into this one
func (o *Operation) Combine(other *Operation) {
	// Take the latest content and Lamport timestamp
	o.Content = other.Content
	o.Lamport = other.Lamport
	o.Timestamp = other.Timestamp

	// Extend vector clock if needed
	if len(other.Vector) > len(o.Vector) {
		newVec := make([]int64, len(other.Vector))
		copy(newVec, o.Vector)
		o.Vector = newVec
	}

	// Update vector clock values
	for i := 0; i < len(other.Vector); i++ {
		if i < len(o.Vector) {
			o.Vector[i] = other.Vector[i]
		}
	}
}

// LessThan compares operations for ordering
func (o *Operation) LessThan(other *Operation) bool {
	if o.Lamport != other.Lamport {
		return o.Lamport < other.Lamport
	}
	return o.NodeID.String() < other.NodeID.String()
}

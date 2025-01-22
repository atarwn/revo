package crdt

import (
	"fmt"
	"sort"
	"sync"
	"github.com/google/uuid"
)

// RGAOperation extends Operation with additional fields
type RGAOperation struct {
	Operation
	Index int
}

// NewRGAOperation creates a new RGAOperation instance
func NewRGAOperation(op Operation, index int) RGAOperation {
	return RGAOperation{
		Operation: op,
		Index:     index,
	}
}

// RGA represents a Replicated Growable Array CRDT
type RGA struct {
	mu        sync.RWMutex
	ops       []RGAOperation
	tombstone map[string]bool
}

// NewRGA creates a new RGA instance
func NewRGA() *RGA {
	return &RGA{
		ops:       make([]RGAOperation, 0),
		tombstone: make(map[string]bool),
	}
}

// Apply applies an operation to the RGA
func (r *RGA) Apply(op Operation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rgaOp := NewRGAOperation(op, len(r.ops))

	switch op.Type {
	case OpInsert:
		r.ops = append(r.ops, rgaOp)
		sort.Slice(r.ops, func(i, j int) bool {
			return r.ops[i].LessThan(&r.ops[j].Operation)
		})
	case OpDelete:
		r.tombstone[op.LineID.String()] = true
	case OpUpdate:
		found := false
		for i := range r.ops {
			if r.ops[i].LineID == op.LineID {
				r.ops[i].Content = op.Content
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("line not found for update: %s", op.LineID)
		}
	default:
		return fmt.Errorf("unknown operation type: %d", op.Type)
	}

	return nil
}

// Get returns the current state of the RGA
func (r *RGA) Get() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []string
	for _, op := range r.ops {
		if !r.tombstone[op.LineID.String()] {
			result = append(result, op.Content)
		}
	}
	return result
}

// GetOperations returns all operations in order
func (r *RGA) GetOperations() []Operation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Operation, len(r.ops))
	for i, op := range r.ops {
		result[i] = op.Operation
	}
	return result
}

// Clear removes all operations and resets the RGA
func (r *RGA) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ops = make([]RGAOperation, 0)
	r.tombstone = make(map[string]bool)
}

// Materialize returns the current document state as a slice of strings
func (r *RGA) Materialize() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []string
	for _, op := range r.ops {
		if !r.tombstone[op.LineID.String()] {
			result = append(result, op.Content)
		}
	}
	return result
}

// GetPositions returns the positions of all active lines
func (r *RGA) GetPositions() []int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var positions []int
	for i, op := range r.ops {
		if !r.tombstone[op.LineID.String()] {
			positions = append(positions, i)
		}
	}
	return positions
}

// GetLineIDs returns the LineIDs of all active lines in order
func (r *RGA) GetLineIDs() []uuid.UUID {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var lineIDs []uuid.UUID
	for _, op := range r.ops {
		if !r.tombstone[op.LineID.String()] {
			lineIDs = append(lineIDs, op.LineID)
		}
	}
	return lineIDs
}

// LineMap returns a map of LineID to Content for all active lines
func (r *RGA) LineMap() map[uuid.UUID]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[uuid.UUID]string)
	for _, op := range r.ops {
		if !r.tombstone[op.LineID.String()] {
			result[op.LineID] = op.Content
		}
	}
	return result
}

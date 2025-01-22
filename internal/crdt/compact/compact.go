package compact

import (
	"evo/internal/crdt"
	"sort"
	"time"

	"github.com/google/uuid"
)

// CompactOperations compacts a list of operations by:
// 1. Pruning old tombstones
// 2. Collapsing multiple operations on the same line into a single op
// 3. Removing redundant operations that don't affect the final state
func CompactOperations(ops []crdt.Operation, cfg *Config) []crdt.Operation {
	if len(ops) < cfg.MaxOps {
		return ops
	}

	// Build a map of lineID to its operations
	lineOps := make(map[uuid.UUID][]crdt.Operation)
	for _, op := range ops {
		lineOps[op.LineID] = append(lineOps[op.LineID], op)
	}

	var compacted []crdt.Operation
	now := time.Now()

	for _, lineHistory := range lineOps {
		// Sort operations by lamport timestamp
		sortOps(lineHistory)

		// Keep only the latest operation for each line
		finalOp := lineHistory[len(lineHistory)-1]
		
		// Skip old tombstones
		if finalOp.Type == crdt.OpDelete {
			age := now.Sub(finalOp.Timestamp)
			if age > cfg.TombstoneTTL {
				continue
			}
		}

		compacted = append(compacted, finalOp)
	}

	// Sort compacted operations
	sortOps(compacted)

	// Ensure we keep minimum number of ops
	if len(compacted) < cfg.MinOpsToKeep {
		return ops[:cfg.MinOpsToKeep]
	}

	return compacted
}

// sortOps sorts operations by lamport timestamp and nodeID
func sortOps(ops []crdt.Operation) {
	sort.Slice(ops, func(i, j int) bool {
		return ops[i].LessThan(&ops[j])
	})
}

// CompactRGA creates a new RGA with compacted operations
func CompactRGA(rga *crdt.RGA, cfg *Config) *crdt.RGA {
	ops := rga.GetOperations()
	compacted := CompactOperations(ops, cfg)

	newRGA := crdt.NewRGA()
	for _, op := range compacted {
		if err := newRGA.Apply(op); err != nil {
			// Log error but continue
			continue
		}
	}

	return newRGA
}

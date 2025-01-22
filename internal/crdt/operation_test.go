package crdt

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestOperationCombining(t *testing.T) {
	t.Run("Same Stream Operations", func(t *testing.T) {
		fileID := uuid.New()
		lineID := uuid.New()
		nodeID := uuid.New()

		op1 := &Operation{
			Type:      OpUpdate,
			Lamport:   1,
			NodeID:    nodeID,
			FileID:    fileID,
			LineID:    lineID,
			Content:   "value1",
			Stream:    "stream1",
			Timestamp: time.Now(),
			Vector:    []int64{1, 0, 0},
		}

		op2 := &Operation{
			Type:      OpUpdate,
			Lamport:   2,
			NodeID:    nodeID,
			FileID:    fileID,
			LineID:    lineID,
			Content:   "value2",
			Stream:    "stream1",
			Timestamp: op1.Timestamp.Add(time.Second),
			Vector:    []int64{1, 1, 0},
		}

		if !op1.CanCombine(op2) {
			t.Error("Expected operations to be combinable")
		}

		op1.Combine(op2)

		if op1.Content != "value2" {
			t.Errorf("Expected combined content to be 'value2', got '%s'", op1.Content)
		}

		if op1.Vector[1] != 1 {
			t.Errorf("Expected vector clock [1] to be 1, got %d", op1.Vector[1])
		}
	})

	t.Run("Different Stream Operations", func(t *testing.T) {
		fileID := uuid.New()
		lineID := uuid.New()
		nodeID := uuid.New()

		op1 := &Operation{
			Type:      OpUpdate,
			Lamport:   1,
			NodeID:    nodeID,
			FileID:    fileID,
			LineID:    lineID,
			Content:   "value1",
			Stream:    "stream1",
			Timestamp: time.Now(),
			Vector:    []int64{1, 0, 0},
		}

		op2 := &Operation{
			Type:      OpUpdate,
			Lamport:   2,
			NodeID:    nodeID,
			FileID:    fileID,
			LineID:    lineID,
			Content:   "value2",
			Stream:    "stream2",
			Timestamp: op1.Timestamp.Add(time.Second),
			Vector:    []int64{1, 1, 0},
		}

		if op1.CanCombine(op2) {
			t.Error("Expected operations from different streams to not be combinable")
		}
	})

	t.Run("Delete Operations", func(t *testing.T) {
		fileID := uuid.New()
		lineID := uuid.New()
		nodeID := uuid.New()

		op1 := &Operation{
			Type:      OpUpdate,
			Lamport:   1,
			NodeID:    nodeID,
			FileID:    fileID,
			LineID:    lineID,
			Content:   "value1",
			Stream:    "stream1",
			Timestamp: time.Now(),
			Vector:    []int64{1, 0, 0},
		}

		op2 := &Operation{
			Type:      OpDelete,
			Lamport:   2,
			NodeID:    nodeID,
			FileID:    fileID,
			LineID:    lineID,
			Stream:    "stream1",
			Timestamp: op1.Timestamp.Add(time.Second),
			Vector:    []int64{1, 1, 0},
		}

		if op1.CanCombine(op2) {
			t.Error("Expected delete operations to not be combinable")
		}
	})

	t.Run("Vector Clock Extension", func(t *testing.T) {
		fileID := uuid.New()
		lineID := uuid.New()
		nodeID := uuid.New()

		op1 := &Operation{
			Type:      OpUpdate,
			Lamport:   1,
			NodeID:    nodeID,
			FileID:    fileID,
			LineID:    lineID,
			Content:   "value1",
			Stream:    "stream1",
			Timestamp: time.Now(),
			Vector:    []int64{1, 0},
		}

		op2 := &Operation{
			Type:      OpUpdate,
			Lamport:   2,
			NodeID:    nodeID,
			FileID:    fileID,
			LineID:    lineID,
			Content:   "value2",
			Stream:    "stream1",
			Timestamp: op1.Timestamp.Add(time.Second),
			Vector:    []int64{1, 1, 1},
		}

		if !op1.CanCombine(op2) {
			t.Error("Expected operations to be combinable")
		}

		op1.Combine(op2)

		if len(op1.Vector) != 3 {
			t.Errorf("Expected vector clock length to be 3, got %d", len(op1.Vector))
		}

		if op1.Vector[2] != 1 {
			t.Errorf("Expected vector clock [2] to be 1, got %d", op1.Vector[2])
		}
	})

	t.Run("Non-Sequential Operations", func(t *testing.T) {
		fileID := uuid.New()
		lineID := uuid.New()
		nodeID := uuid.New()

		op1 := &Operation{
			Type:      OpUpdate,
			Lamport:   2, // Higher Lamport
			NodeID:    nodeID,
			FileID:    fileID,
			LineID:    lineID,
			Content:   "value1",
			Stream:    "stream1",
			Timestamp: time.Now().Add(time.Second),
			Vector:    []int64{1, 0, 0},
		}

		op2 := &Operation{
			Type:      OpUpdate,
			Lamport:   1, // Lower Lamport
			NodeID:    nodeID,
			FileID:    fileID,
			LineID:    lineID,
			Content:   "value2",
			Stream:    "stream1",
			Timestamp: time.Now(),
			Vector:    []int64{1, 1, 0},
		}

		if op1.CanCombine(op2) {
			t.Error("Expected non-sequential operations to not be combinable")
		}
	})
}

func TestOperationOrdering(t *testing.T) {
	now := time.Now()
	fileID := uuid.New()
	lineID := uuid.New()
	nodeID := uuid.New()

	ops := []Operation{
		{
			Type:      OpInsert,
			Lamport:   1,
			NodeID:    nodeID,
			FileID:    fileID,
			LineID:    lineID,
			Content:   "value1",
			Stream:    "stream1",
			Timestamp: now,
			Vector:    []int64{1, 0, 0},
		},
		{
			Type:      OpUpdate,
			Lamport:   2,
			NodeID:    nodeID,
			FileID:    fileID,
			LineID:    lineID,
			Content:   "value2",
			Stream:    "stream1",
			Timestamp: now.Add(time.Second),
			Vector:    []int64{1, 1, 0},
		},
		{
			Type:      OpDelete,
			Lamport:   3,
			NodeID:    nodeID,
			FileID:    fileID,
			LineID:    lineID,
			Stream:    "stream1",
			Timestamp: now.Add(2 * time.Second),
			Vector:    []int64{1, 1, 1},
		},
	}

	t.Run("Timestamp Order", func(t *testing.T) {
		if !ops[0].Timestamp.Before(ops[1].Timestamp) {
			t.Error("Expected op1 timestamp to be before op2")
		}

		if !ops[1].Timestamp.Before(ops[2].Timestamp) {
			t.Error("Expected op2 timestamp to be before op3")
		}
	})

	t.Run("Vector Clock Order", func(t *testing.T) {
		// Test that vector clocks are monotonically increasing
		for i := 1; i < len(ops); i++ {
			prev := ops[i-1].Vector
			curr := ops[i].Vector
			increasing := false
			for j := 0; j < len(prev) && j < len(curr); j++ {
				if curr[j] > prev[j] {
					increasing = true
					break
				}
			}
			if !increasing {
				t.Errorf("Expected vector clock to increase between op%d and op%d", i, i+1)
			}
		}
	})
}

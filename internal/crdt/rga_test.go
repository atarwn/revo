package crdt

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRGA(t *testing.T) {
	t.Run("Insert Operations", func(t *testing.T) {
		rga := NewRGA()
		fileID := uuid.New()
		lineID1 := uuid.New()
		lineID2 := uuid.New()
		nodeID := uuid.New()

		// Create operations
		op1 := Operation{
			Type:      OpInsert,
			Lamport:   1,
			NodeID:    nodeID,
			FileID:    fileID,
			LineID:    lineID1,
			Content:   "value1",
			Stream:    "stream1",
			Timestamp: time.Now(),
			Vector:    []int64{1},
		}

		op2 := Operation{
			Type:      OpInsert,
			Lamport:   2,
			NodeID:    nodeID,
			FileID:    fileID,
			LineID:    lineID2,
			Content:   "value2",
			Stream:    "stream1",
			Timestamp: time.Now().Add(time.Second),
			Vector:    []int64{2},
		}

		// Apply operations
		err := rga.Apply(op1)
		if err != nil {
			t.Errorf("Failed to apply operation 1: %v", err)
		}

		err = rga.Apply(op2)
		if err != nil {
			t.Errorf("Failed to apply operation 2: %v", err)
		}

		// Check state
		values := rga.Get()
		if len(values) != 2 {
			t.Errorf("Expected 2 values, got %d", len(values))
		}

		if values[0] != "value1" || values[1] != "value2" {
			t.Errorf("Values not in expected order: %v", values)
		}
	})

	t.Run("Delete Operations", func(t *testing.T) {
		rga := NewRGA()
		fileID := uuid.New()
		lineID := uuid.New()
		nodeID := uuid.New()

		// Insert operation
		insertOp := Operation{
			Type:      OpInsert,
			Lamport:   1,
			NodeID:    nodeID,
			FileID:    fileID,
			LineID:    lineID,
			Content:   "value1",
			Stream:    "stream1",
			Timestamp: time.Now(),
			Vector:    []int64{1},
		}

		// Apply insert
		err := rga.Apply(insertOp)
		if err != nil {
			t.Errorf("Failed to apply insert operation: %v", err)
		}

		// Delete operation
		deleteOp := Operation{
			Type:      OpDelete,
			Lamport:   2,
			NodeID:    nodeID,
			FileID:    fileID,
			LineID:    lineID,
			Stream:    "stream1",
			Timestamp: time.Now().Add(time.Second),
			Vector:    []int64{2},
		}

		// Apply delete
		err = rga.Apply(deleteOp)
		if err != nil {
			t.Errorf("Failed to apply delete operation: %v", err)
		}

		// Check state
		values := rga.Get()
		if len(values) != 0 {
			t.Errorf("Expected 0 values after delete, got %d", len(values))
		}
	})

	t.Run("Update Operations", func(t *testing.T) {
		rga := NewRGA()
		fileID := uuid.New()
		lineID := uuid.New()
		nodeID := uuid.New()

		// Insert operation
		insertOp := Operation{
			Type:      OpInsert,
			Lamport:   1,
			NodeID:    nodeID,
			FileID:    fileID,
			LineID:    lineID,
			Content:   "value1",
			Stream:    "stream1",
			Timestamp: time.Now(),
			Vector:    []int64{1},
		}

		// Apply insert
		err := rga.Apply(insertOp)
		if err != nil {
			t.Errorf("Failed to apply insert operation: %v", err)
		}

		// Update operation
		updateOp := Operation{
			Type:      OpUpdate,
			Lamport:   2,
			NodeID:    nodeID,
			FileID:    fileID,
			LineID:    lineID,
			Content:   "updated",
			Stream:    "stream1",
			Timestamp: time.Now().Add(time.Second),
			Vector:    []int64{2},
		}

		// Apply update
		err = rga.Apply(updateOp)
		if err != nil {
			t.Errorf("Failed to apply update operation: %v", err)
		}

		// Check state
		values := rga.Get()
		if len(values) != 1 {
			t.Errorf("Expected 1 value after update, got %d", len(values))
		}

		if values[0] != "updated" {
			t.Errorf("Expected updated value, got %s", values[0])
		}
	})

	t.Run("Invalid Update Operation", func(t *testing.T) {
		rga := NewRGA()
		fileID := uuid.New()
		lineID := uuid.New()
		nodeID := uuid.New()

		// Update operation without insert
		updateOp := Operation{
			Type:      OpUpdate,
			Lamport:   1,
			NodeID:    nodeID,
			FileID:    fileID,
			LineID:    lineID,
			Content:   "updated",
			Stream:    "stream1",
			Timestamp: time.Now(),
			Vector:    []int64{1},
		}

		// Apply update
		err := rga.Apply(updateOp)
		if err == nil {
			t.Error("Expected error when updating non-existent line")
		}
	})

	t.Run("Clear Operations", func(t *testing.T) {
		rga := NewRGA()
		fileID := uuid.New()
		lineID1 := uuid.New()
		lineID2 := uuid.New()
		nodeID := uuid.New()

		// Insert operations
		op1 := Operation{
			Type:      OpInsert,
			Lamport:   1,
			NodeID:    nodeID,
			FileID:    fileID,
			LineID:    lineID1,
			Content:   "value1",
			Stream:    "stream1",
			Timestamp: time.Now(),
			Vector:    []int64{1},
		}

		op2 := Operation{
			Type:      OpInsert,
			Lamport:   2,
			NodeID:    nodeID,
			FileID:    fileID,
			LineID:    lineID2,
			Content:   "value2",
			Stream:    "stream1",
			Timestamp: time.Now().Add(time.Second),
			Vector:    []int64{2},
		}

		// Apply operations
		err := rga.Apply(op1)
		if err != nil {
			t.Errorf("Failed to apply operation 1: %v", err)
		}

		err = rga.Apply(op2)
		if err != nil {
			t.Errorf("Failed to apply operation 2: %v", err)
		}

		// Clear RGA
		rga.Clear()

		// Check state
		values := rga.Get()
		if len(values) != 0 {
			t.Errorf("Expected 0 values after clear, got %d", len(values))
		}

		ops := rga.GetOperations()
		if len(ops) != 0 {
			t.Errorf("Expected 0 operations after clear, got %d", len(ops))
		}
	})
}

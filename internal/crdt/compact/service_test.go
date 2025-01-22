package compact

import (
	"encoding/binary"
	"encoding/json"
	"evo/internal/crdt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCompactionService(t *testing.T) {
	// Create temp directory for testing
	tmpDir, err := os.MkdirTemp("", "evo-compact-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test repository structure
	repoPath := filepath.Join(tmpDir, "test-repo")
	if err := os.MkdirAll(filepath.Join(repoPath, ".evo", "ops"), 0755); err != nil {
		t.Fatal(err)
	}

	t.Run("Service Lifecycle", func(t *testing.T) {
		config := &Config{
			CompactionInterval: 100 * time.Millisecond,
			TombstoneTTL:       1 * time.Hour,
			MinOpsToKeep:       10,
			MaxOps:             100,
		}

		service := NewCompactionService(repoPath, config)
		if err := service.Start(); err != nil {
			t.Fatal(err)
		}

		// Let it run for a bit
		time.Sleep(200 * time.Millisecond)

		service.Stop()
	})

	t.Run("Operation Compaction", func(t *testing.T) {
		fileID := uuid.New()
		lineID := uuid.New()
		nodeID := uuid.New()

		// Create test operations
		ops := []crdt.Operation{
			{
				Type:      crdt.OpUpdate,
				Lamport:   1,
				NodeID:    nodeID,
				FileID:    fileID,
				LineID:    lineID,
				Content:   "value1",
				Stream:    "stream1",
				Timestamp: time.Now().Add(-2 * time.Hour),
				Vector:    []int64{1, 0, 0},
			},
			{
				Type:      crdt.OpUpdate,
				Lamport:   2,
				NodeID:    nodeID,
				FileID:    fileID,
				LineID:    lineID,
				Content:   "value2",
				Stream:    "stream1",
				Timestamp: time.Now().Add(-1 * time.Hour),
				Vector:    []int64{1, 1, 0},
			},
			{
				Type:      crdt.OpDelete,
				Lamport:   3,
				NodeID:    nodeID,
				FileID:    fileID,
				LineID:    lineID,
				Stream:    "stream1",
				Timestamp: time.Now(),
				Vector:    []int64{1, 1, 1},
			},
		}

		// Write operations to file
		opsFile := filepath.Join(repoPath, ".evo", "ops", "test.bin")
		f, err := os.Create(opsFile)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		for _, op := range ops {
			data, err := json.Marshal(op)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.Write(data); err != nil {
				t.Fatal(err)
			}
		}

		// Run compaction
		config := &Config{
			CompactionInterval: 100 * time.Millisecond,
			TombstoneTTL:       30 * time.Minute,
			MinOpsToKeep:       1,
			MaxOps:             2,
		}

		service := NewCompactionService(repoPath, config)
		if err := service.CompactOperations(); err != nil {
			t.Fatal(err)
		}

		// Verify results
		// TODO: Add verification logic
	})

	t.Run("Tombstone Pruning", func(t *testing.T) {
		fileID := uuid.New()
		lineID := uuid.New()
		nodeID := uuid.New()

		// Create test operations including a tombstone
		ops := []crdt.Operation{
			{
				Type:      crdt.OpUpdate,
				Lamport:   1,
				NodeID:    nodeID,
				FileID:    fileID,
				LineID:    lineID,
				Content:   "value1",
				Stream:    "stream1",
				Timestamp: time.Now(),
				Vector:    []int64{1, 0, 0},
			},
			{
				Type:      crdt.OpDelete,
				Lamport:   2,
				NodeID:    nodeID,
				FileID:    fileID,
				LineID:    uuid.New(), // Use a different LineID for the tombstone
				Stream:    "stream1",
				Timestamp: time.Now().Add(-2 * time.Hour), // Old tombstone
				Vector:    []int64{1, 1, 0},
			},
		}

		// Write operations to disk
		opsDir := filepath.Join(repoPath, ".evo", "ops")
		streamDir := filepath.Join(opsDir, "stream1")
		if err := os.MkdirAll(streamDir, 0755); err != nil {
			t.Fatal(err)
		}

		for _, op := range ops {
			data, err := json.Marshal(op)
			if err != nil {
				t.Fatal(err)
			}

			// Write size prefix followed by data
			opFile := filepath.Join(streamDir, op.LineID.String()+".bin")
			f, err := os.Create(opFile)
			if err != nil {
				t.Fatal(err)
			}

			// Write 4-byte size prefix
			size := uint32(len(data))
			var sizeBuf [4]byte
			binary.BigEndian.PutUint32(sizeBuf[:], size)
			if _, err := f.Write(sizeBuf[:]); err != nil {
				f.Close()
				t.Fatal(err)
			}

			// Write operation data
			if _, err := f.Write(data); err != nil {
				f.Close()
				t.Fatal(err)
			}
			f.Close()
		}

		// Create and run compaction service
		config := &Config{
			CompactionInterval: 1 * time.Hour,
			TombstoneTTL:       1 * time.Hour,
			MinOpsToKeep:       1,
			MaxOps:             10,
		}

		service := NewCompactionService(repoPath, config)
		if err := service.PruneTombstones(); err != nil {
			t.Fatal(err)
		}

		// Check that old tombstone was removed
		files, err := os.ReadDir(streamDir)
		if err != nil {
			t.Fatal(err)
		}

		if len(files) != 1 {
			t.Errorf("Expected 1 operation after pruning, got %d", len(files))
		}

		// The remaining operation should be the update
		for _, f := range files {
			data, err := os.ReadFile(filepath.Join(streamDir, f.Name()))
			if err != nil {
				t.Fatal(err)
			}

			// Read size prefix
			if len(data) < 4 {
				t.Fatal("Invalid operation file: too short")
			}
			size := binary.BigEndian.Uint32(data[:4])
			if len(data) < int(4+size) {
				t.Fatalf("Invalid operation file: expected %d bytes after size prefix, got %d", size, len(data)-4)
			}
			opData := data[4 : 4+size]

			var op crdt.Operation
			if err := json.Unmarshal(opData, &op); err != nil {
				t.Fatal(err)
			}

			if op.Type == crdt.OpDelete {
				t.Error("Expected tombstone to be pruned")
			}
		}
	})
}

func TestCompactionConfig(t *testing.T) {
	t.Run("Default Config", func(t *testing.T) {
		cfg := DefaultConfig()
		if cfg.MaxOps <= cfg.MinOpsToKeep {
			t.Error("MaxOps should be greater than MinOpsToKeep")
		}
		if cfg.TombstoneTTL <= 0 {
			t.Error("TombstoneTTL should be positive")
		}
	})

	t.Run("Custom Config", func(t *testing.T) {
		cfg := &Config{
			MaxOps:             5000,
			MinOpsToKeep:       500,
			TombstoneTTL:       48 * time.Hour,
			CompactionInterval: time.Hour,
		}

		service := NewCompactionService("test-path", cfg)
		if service.config.MaxOps != 5000 {
			t.Error("Failed to set custom MaxOps")
		}
		if service.config.MinOpsToKeep != 500 {
			t.Error("Failed to set custom MinOpsToKeep")
		}
		if service.config.TombstoneTTL != 48*time.Hour {
			t.Error("Failed to set custom TombstoneTTL")
		}
	})
}

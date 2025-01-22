package compact

import (
	"encoding/binary"
	"encoding/json"
	"evo/internal/crdt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// CompactionService manages operation compaction and tombstone pruning
type CompactionService struct {
	repoPath string
	config   *Config
	mu       sync.RWMutex
	done     chan struct{}
}

// NewCompactionService creates a new compaction service
func NewCompactionService(repoPath string, config *Config) *CompactionService {
	if config == nil {
		config = DefaultConfig()
	}
	return &CompactionService{
		repoPath: repoPath,
		config:   config,
		done:     make(chan struct{}),
	}
}

// Start begins the compaction service
func (s *CompactionService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create ticker for periodic compaction
	ticker := time.NewTicker(s.config.CompactionInterval)

	// Start background goroutine
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := s.CompactOperations(); err != nil {
					// Log error but continue running
					continue
				}
				if err := s.PruneTombstones(); err != nil {
					// Log error but continue running
					continue
				}
			case <-s.done:
				ticker.Stop()
				return
			}
		}
	}()

	return nil
}

// Stop stops the compaction service
func (s *CompactionService) Stop() {
	close(s.done)
}

// CompactOperations compacts operations by combining sequential operations
func (s *CompactionService) CompactOperations() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	opsDir := filepath.Join(s.repoPath, ".evo", "ops")
	streams, err := os.ReadDir(opsDir)
	if err != nil {
		return err
	}

	for _, stream := range streams {
		if !stream.IsDir() {
			continue
		}

		streamDir := filepath.Join(opsDir, stream.Name())
		files, err := os.ReadDir(streamDir)
		if err != nil {
			continue
		}

		var ops []crdt.Operation
		for _, f := range files {
			if !strings.HasSuffix(f.Name(), ".bin") {
				continue
			}

			data, err := os.ReadFile(filepath.Join(streamDir, f.Name()))
			if err != nil {
				continue
			}

			// Read size prefix
			if len(data) < 4 {
				continue
			}
			size := binary.BigEndian.Uint32(data[:4])
			if len(data) < int(4+size) {
				continue
			}
			opData := data[4 : 4+size]

			var op crdt.Operation
			if err := json.Unmarshal(opData, &op); err != nil {
				continue
			}
			ops = append(ops, op)
		}

		if len(ops) < s.config.MaxOps {
			continue
		}

		// Combine sequential operations
		for i := range ops {
			op := &ops[i]
			if i > 0 && ops[i-1].LineID == op.LineID {
				// Combine with previous operation
				ops[i-1].Content = op.Content
				ops[i-1].Lamport = op.Lamport
				ops[i-1].Timestamp = op.Timestamp
				ops = append(ops[:i], ops[i+1:]...)
				i--
				continue
			}
		}

		// Write compacted operations back
		compacted := make([]crdt.Operation, 0, len(ops))
		for _, op := range ops {
			if op.Type != crdt.OpDelete || time.Since(op.Timestamp) <= s.config.TombstoneTTL {
				compacted = append(compacted, op)
			}
		}

		// Save compacted operations
		for _, op := range compacted {
			data, err := json.Marshal(op)
			if err != nil {
				continue
			}

			// Write size prefix followed by data
			opPath := filepath.Join(streamDir, op.LineID.String()+".bin")
			f, err := os.Create(opPath)
			if err != nil {
				continue
			}

			// Write 4-byte size prefix
			size := uint32(len(data))
			var sizeBuf [4]byte
			binary.BigEndian.PutUint32(sizeBuf[:], size)
			if _, err := f.Write(sizeBuf[:]); err != nil {
				f.Close()
				continue
			}

			// Write operation data
			if _, err := f.Write(data); err != nil {
				f.Close()
				continue
			}
			f.Close()
		}

		// Remove old operations
		for _, op := range ops {
			found := false
			for _, c := range compacted {
				if c.LineID == op.LineID {
					found = true
					break
				}
			}
			if !found {
				os.Remove(filepath.Join(streamDir, op.LineID.String()+".bin"))
			}
		}
	}

	return nil
}

// PruneTombstones removes old tombstones
func (s *CompactionService) PruneTombstones() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	opsDir := filepath.Join(s.repoPath, ".evo", "ops")
	streams, err := os.ReadDir(opsDir)
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-s.config.TombstoneTTL)

	for _, stream := range streams {
		if !stream.IsDir() {
			continue
		}

		streamDir := filepath.Join(opsDir, stream.Name())
		files, err := os.ReadDir(streamDir)
		if err != nil {
			continue
		}

		var ops []crdt.Operation
		var filesToRemove []string

		// Read all operations in this stream
		for _, f := range files {
			if !strings.HasSuffix(f.Name(), ".bin") {
				continue
			}

			data, err := os.ReadFile(filepath.Join(streamDir, f.Name()))
			if err != nil {
				continue
			}

			// Read size prefix
			if len(data) < 4 {
				continue
			}
			size := binary.BigEndian.Uint32(data[:4])
			if len(data) < int(4+size) {
				continue
			}
			opData := data[4 : 4+size]

			var op crdt.Operation
			if err := json.Unmarshal(opData, &op); err != nil {
				continue
			}

			// Keep non-delete operations and recent tombstones
			if op.Type != crdt.OpDelete || op.Timestamp.After(cutoff) {
				ops = append(ops, op)
			} else {
				filesToRemove = append(filesToRemove, f.Name())
			}
		}

		// Remove old tombstones
		for _, name := range filesToRemove {
			if err := os.Remove(filepath.Join(streamDir, name)); err != nil && !os.IsNotExist(err) {
				return err
			}
		}

		// Write remaining operations back
		for _, op := range ops {
			data, err := json.Marshal(op)
			if err != nil {
				return err
			}

			// Write size prefix followed by data
			opPath := filepath.Join(streamDir, op.LineID.String()+".bin")
			tempPath := opPath + ".tmp"
			f, err := os.Create(tempPath)
			if err != nil {
				return err
			}

			// Write 4-byte size prefix
			size := uint32(len(data))
			var sizeBuf [4]byte
			binary.BigEndian.PutUint32(sizeBuf[:], size)
			if _, err := f.Write(sizeBuf[:]); err != nil {
				f.Close()
				os.Remove(tempPath)
				return err
			}

			// Write operation data
			if _, err := f.Write(data); err != nil {
				f.Close()
				os.Remove(tempPath)
				return err
			}
			f.Close()

			// Atomically replace the old file with the new one
			if err := os.Rename(tempPath, opPath); err != nil {
				os.Remove(tempPath)
				return err
			}
		}

		// Remove any remaining files that weren't rewritten
		for _, f := range files {
			if !strings.HasSuffix(f.Name(), ".bin") {
				continue
			}

			found := false
			for _, op := range ops {
				if f.Name() == op.LineID.String()+".bin" {
					found = true
					break
				}
			}

			if !found {
				if err := os.Remove(filepath.Join(streamDir, f.Name())); err != nil && !os.IsNotExist(err) {
					return err
				}
			}
		}
	}

	return nil
}

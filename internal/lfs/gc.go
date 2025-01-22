package lfs

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// GarbageCollector manages cleanup of unreferenced chunks
type GarbageCollector struct {
	store *Store
	mu    sync.Mutex
	done  chan struct{}
}

// NewGarbageCollector creates a new garbage collector
func NewGarbageCollector(store *Store) *GarbageCollector {
	return &GarbageCollector{
		store: store,
		done:  make(chan struct{}),
	}
}

// Start begins periodic garbage collection
func (gc *GarbageCollector) Start() {
	ticker := time.NewTicker(24 * time.Hour) // Run daily
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := gc.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "Error during LFS garbage collection: %v\n", err)
				}
			case <-gc.done:
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop stops the garbage collector
func (gc *GarbageCollector) Stop() {
	close(gc.done)
}

// Run performs garbage collection
func (gc *GarbageCollector) Run() error {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	// Get all chunks
	chunksDir := filepath.Join(gc.store.root, ".evo", "chunks")
	chunks, err := os.ReadDir(chunksDir)
	if err != nil {
		return fmt.Errorf("failed to read chunks directory: %w", err)
	}

	// Check each chunk
	for _, chunk := range chunks {
		if chunk.IsDir() {
			continue
		}

		// Delete if not referenced
		chunkHash := chunk.Name()
		if !gc.store.isChunkReferenced(chunkHash) {
			chunkPath := filepath.Join(chunksDir, chunkHash)
			if err := os.Remove(chunkPath); err != nil {
				return fmt.Errorf("failed to delete unreferenced chunk %s: %w", chunkHash, err)
			}
		}
	}

	return nil
}

// PruneTombstones removes old tombstones
func (gc *GarbageCollector) PruneTombstones(maxAge time.Duration) error {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	// Get all files
	filesDir := filepath.Join(gc.store.root, ".evo", "lfs")
	files, err := os.ReadDir(filesDir)
	if err != nil {
		return fmt.Errorf("failed to read files directory: %w", err)
	}

	cutoff := time.Now().Add(-maxAge)

	// Check each file
	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		// Load file info
		info, err := gc.store.loadFileInfo(file.Name())
		if err != nil {
			continue
		}

		// Delete if it's a tombstone older than maxAge
		if info.RefCount == 0 && info.Created.Before(cutoff) {
			if err := gc.store.DeleteFile(file.Name()); err != nil {
				return fmt.Errorf("failed to delete old tombstone %s: %w", file.Name(), err)
			}
		}
	}

	return nil
}

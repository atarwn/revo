package repo

import (
	"errors"
	"evo/internal/crdt/compact"
	"evo/internal/lfs"
	"os"
	"path/filepath"
	"sync"
)

const EvoDir = ".evo"

var (
	compactionService *compact.CompactionService
	garbageCollector  *lfs.GarbageCollector
	serviceMutex      sync.Mutex
)

// InitRepo creates the .evo folder structure, default stream, config, index, etc.
func InitRepo(path string) error {
	serviceMutex.Lock()
	defer serviceMutex.Unlock()

	evoPath := filepath.Join(path, EvoDir)
	if _, err := os.Stat(evoPath); err == nil {
		return errors.New("Evo repository already exists here")
	}

	dirs := []string{
		filepath.Join(path, EvoDir),
		filepath.Join(path, EvoDir, "ops"),
		filepath.Join(path, EvoDir, "commits"),
		filepath.Join(path, EvoDir, "config"),
		filepath.Join(path, EvoDir, "streams"),
		filepath.Join(path, EvoDir, "largefiles"),
		filepath.Join(path, EvoDir, "cache"),
		filepath.Join(path, EvoDir, "chunks"),
		filepath.Join(path, EvoDir, "lfs"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}

	// Start compaction service
	cs := compact.NewCompactionService(path, compact.DefaultConfig())
	if err := cs.Start(); err != nil {
		return err
	}
	compactionService = cs

	// Start LFS garbage collector
	store := lfs.NewStore(path)
	gc := lfs.NewGarbageCollector(store)
	gc.Start()
	garbageCollector = gc

	// HEAD => "main"
	if err := os.WriteFile(filepath.Join(evoPath, "HEAD"), []byte("main"), 0644); err != nil {
		return err
	}
	// create stream "main"
	if err := os.WriteFile(filepath.Join(evoPath, "streams", "main"), []byte{}, 0644); err != nil {
		return err
	}

	// create empty .evo/index
	if err := os.WriteFile(filepath.Join(evoPath, "index"), []byte{}, 0644); err != nil {
		return err
	}

	return nil
}

// Cleanup stops all background services
func Cleanup() {
	serviceMutex.Lock()
	defer serviceMutex.Unlock()

	if compactionService != nil {
		compactionService.Stop()
		compactionService = nil
	}

	if garbageCollector != nil {
		garbageCollector.Stop()
		garbageCollector = nil
	}
}

// FindRepoRoot searches for .evo directory walking up from start
func FindRepoRoot(start string) (string, error) {
	cur, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(cur, EvoDir)); err == nil {
			return cur, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", os.ErrNotExist
		}
		cur = parent
	}
}

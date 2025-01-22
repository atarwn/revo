package compact

import "time"

// Config defines thresholds for when to perform compaction
type Config struct {
	// Maximum number of operations before triggering compaction
	MaxOps int
	// Maximum age of tombstones before pruning
	TombstoneTTL time.Duration
	// Minimum number of operations to keep after compaction
	MinOpsToKeep int
	// How often to run compaction
	CompactionInterval time.Duration
}

// DefaultConfig returns sensible defaults for compaction
func DefaultConfig() *Config {
	return &Config{
		MaxOps:             10000,                // Compact when we have more than 10k ops
		TombstoneTTL:       7 * 24 * time.Hour,  // Keep tombstones for 1 week
		MinOpsToKeep:       1000,                // Keep at least 1k ops after compaction
		CompactionInterval: 1 * time.Hour,       // Run compaction every hour
	}
}

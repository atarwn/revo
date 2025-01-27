package streams

import (
	"evo/internal/commits"
	"evo/internal/crdt"
	"evo/internal/repo"
	"evo/internal/types"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPartialMerge(t *testing.T) {
	// Create temp repo
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Initialize repo structure
	assert.NoError(t, os.MkdirAll(filepath.Join(repoPath, repo.EvoDir, "commits", "main"), 0755))
	assert.NoError(t, os.MkdirAll(filepath.Join(repoPath, repo.EvoDir, "ops", "main"), 0755))
	assert.NoError(t, CreateStream(repoPath, "feature"))
	assert.NoError(t, os.MkdirAll(filepath.Join(repoPath, repo.EvoDir, "commits", "feature"), 0755))
	assert.NoError(t, os.MkdirAll(filepath.Join(repoPath, repo.EvoDir, "ops", "feature"), 0755))

	// Create test commits with different file IDs and operation types
	file1ID := uuid.New()
	file2ID := uuid.New()
	testCommits := []types.Commit{
		{
			ID:      uuid.New().String(),
			Stream:  "feature",
			Message: "commit 1",
			Operations: []commits.ExtendedOp{
				{
					Op: crdt.Operation{
						Type:      crdt.OpInsert,
						FileID:    file1ID,
						LineID:    uuid.New(),
						Content:   "file1 line1",
						Stream:    "feature",
						Timestamp: time.Now(),
						NodeID:    uuid.New(),
						Lamport:   1,
						Vector:    []int64{1},
					},
				},
				{
					Op: crdt.Operation{
						Type:      crdt.OpDelete,
						FileID:    file2ID,
						LineID:    uuid.New(),
						Content:   "file2 line1",
						Stream:    "feature",
						Timestamp: time.Now(),
						NodeID:    uuid.New(),
						Lamport:   2,
						Vector:    []int64{2},
					},
				},
			},
			Timestamp: time.Now(),
		},
	}

	for _, c := range testCommits {
		assert.NoError(t, commits.SaveCommitFile(filepath.Join(repoPath, repo.EvoDir, "commits", "feature"), &c))
	}

	// Test partial merge with file ID filter
	fileFilter := MergeFilter{
		FileIDs: []string{file1ID.String()},
	}
	var err error
	var mainCommits []types.Commit
	err = PartialMerge(repoPath, "feature", "main", fileFilter)
	assert.NoError(t, err)

	// Verify only file1 operations were merged
	mainCommits, err = ListCommits(repoPath, "main")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(mainCommits))
	assert.Equal(t, file1ID, mainCommits[0].Operations[0].Op.FileID)
	assert.Equal(t, "file1 line1", mainCommits[0].Operations[0].Op.Content)

	// Test partial merge with operation type filter
	typeFilter := MergeFilter{
		OpTypes: []crdt.OpType{crdt.OpDelete},
	}
	err = PartialMerge(repoPath, "feature", "main", typeFilter)
	assert.NoError(t, err)

	// Verify only delete operations were merged
	mainCommits, err = ListCommits(repoPath, "main")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(mainCommits))
	assert.Equal(t, crdt.OpDelete, mainCommits[0].Operations[0].Op.Type)
	assert.Equal(t, file2ID, mainCommits[0].Operations[0].Op.FileID)

	// Test partial merge with empty filter (should merge all)
	emptyFilter := MergeFilter{}
	err = PartialMerge(repoPath, "feature", "main", emptyFilter)
	assert.NoError(t, err)

	// Verify all commits were merged
	mainCommits, err = ListCommits(repoPath, "main")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(mainCommits))               // Since we're preserving commit IDs, we should have one commit
	assert.Equal(t, 2, len(mainCommits[0].Operations)) // But it should contain all operations
}

func TestShouldIncludeOp(t *testing.T) {
	fileID := uuid.New()
	testOp := commits.ExtendedOp{
		Op: crdt.Operation{
			Type:   crdt.OpInsert,
			FileID: fileID,
		},
	}

	// Test empty filter
	assert.True(t, shouldIncludeOp(testOp, MergeFilter{}))

	// Test file ID filter match
	assert.True(t, shouldIncludeOp(testOp, MergeFilter{FileIDs: []string{fileID.String()}}))

	// Test file ID filter no match
	assert.False(t, shouldIncludeOp(testOp, MergeFilter{FileIDs: []string{uuid.New().String()}}))

	// Test operation type filter match
	assert.True(t, shouldIncludeOp(testOp, MergeFilter{OpTypes: []crdt.OpType{crdt.OpInsert}}))

	// Test operation type filter no match
	assert.False(t, shouldIncludeOp(testOp, MergeFilter{OpTypes: []crdt.OpType{crdt.OpDelete}}))

	// Test both filters match
	assert.True(t, shouldIncludeOp(testOp, MergeFilter{
		FileIDs: []string{fileID.String()},
		OpTypes: []crdt.OpType{crdt.OpInsert},
	}))

	// Test both filters, one no match
	assert.False(t, shouldIncludeOp(testOp, MergeFilter{
		FileIDs: []string{fileID.String()},
		OpTypes: []crdt.OpType{crdt.OpDelete},
	}))
}

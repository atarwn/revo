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

func TestCherryPick(t *testing.T) {
	// Create temp repo
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Initialize repo structure
	assert.NoError(t, os.MkdirAll(filepath.Join(repoPath, repo.EvoDir, "commits", "main"), 0755))
	assert.NoError(t, os.MkdirAll(filepath.Join(repoPath, repo.EvoDir, "ops", "main"), 0755))
	assert.NoError(t, CreateStream(repoPath, "feature"))
	assert.NoError(t, os.MkdirAll(filepath.Join(repoPath, repo.EvoDir, "commits", "feature"), 0755))
	assert.NoError(t, os.MkdirAll(filepath.Join(repoPath, repo.EvoDir, "ops", "feature"), 0755))

	// Create a test commit in feature stream
	fileID := uuid.New()
	testOp := commits.ExtendedOp{
		Op: crdt.Operation{
			Type:      crdt.OpInsert,
			FileID:    fileID,
			LineID:    uuid.New(),
			Content:   "test line",
			Stream:    "feature",
			Timestamp: time.Now(),
			NodeID:    uuid.New(),
			Lamport:   1,
			Vector:    []int64{1},
		},
	}
	testCommit := types.Commit{
		ID:         uuid.New().String(),
		Stream:     "feature",
		Message:    "test commit",
		Timestamp:  time.Now(),
		Operations: []commits.ExtendedOp{testOp},
	}
	assert.NoError(t, commits.SaveCommitFile(filepath.Join(repoPath, repo.EvoDir, "commits", "feature"), &testCommit))

	// Test cherry-pick to main stream
	err := CherryPick(repoPath, testCommit.ID, "main")
	assert.NoError(t, err)

	// Verify commit was replicated
	mainCommits, err := ListCommits(repoPath, "main")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(mainCommits))
	assert.Contains(t, mainCommits[0].Message, "[cherry-pick]")
	assert.Equal(t, "main", mainCommits[0].Stream)
	assert.Equal(t, 1, len(mainCommits[0].Operations))
	assert.Equal(t, fileID, mainCommits[0].Operations[0].Op.FileID)
	assert.Equal(t, "test line", mainCommits[0].Operations[0].Op.Content)
}

func TestMergeStreams(t *testing.T) {
	// Create temp repo
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Initialize repo structure
	assert.NoError(t, os.MkdirAll(filepath.Join(repoPath, repo.EvoDir, "commits", "main"), 0755))
	assert.NoError(t, os.MkdirAll(filepath.Join(repoPath, repo.EvoDir, "ops", "main"), 0755))
	assert.NoError(t, CreateStream(repoPath, "feature"))
	assert.NoError(t, os.MkdirAll(filepath.Join(repoPath, repo.EvoDir, "commits", "feature"), 0755))
	assert.NoError(t, os.MkdirAll(filepath.Join(repoPath, repo.EvoDir, "ops", "feature"), 0755))

	// Create multiple test commits in feature stream
	fileID := uuid.New()
	testCommits := []types.Commit{
		{
			ID:      uuid.New().String(),
			Stream:  "feature",
			Message: "commit 1",
			Operations: []commits.ExtendedOp{{
				Op: crdt.Operation{
					Type:      crdt.OpInsert,
					FileID:    fileID,
					LineID:    uuid.New(),
					Content:   "line 1",
					Stream:    "feature",
					Timestamp: time.Now(),
					NodeID:    uuid.New(),
					Lamport:   1,
					Vector:    []int64{1},
				},
			}},
			Timestamp: time.Now(),
		},
		{
			ID:      uuid.New().String(),
			Stream:  "feature",
			Message: "commit 2",
			Operations: []commits.ExtendedOp{{
				Op: crdt.Operation{
					Type:      crdt.OpInsert,
					FileID:    fileID,
					LineID:    uuid.New(),
					Content:   "line 2",
					Stream:    "feature",
					Timestamp: time.Now(),
					NodeID:    uuid.New(),
					Lamport:   2,
					Vector:    []int64{2},
				},
			}},
			Timestamp: time.Now().Add(time.Second),
		},
	}

	for _, c := range testCommits {
		assert.NoError(t, commits.SaveCommitFile(filepath.Join(repoPath, repo.EvoDir, "commits", "feature"), &c))
	}

	// Test merge streams
	err := MergeStreams(repoPath, "feature", "main")
	assert.NoError(t, err)

	// Verify all commits were replicated
	mainCommits, err := ListCommits(repoPath, "main")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(mainCommits))
	assert.Equal(t, "main", mainCommits[0].Stream)
	assert.Equal(t, "main", mainCommits[1].Stream)
	assert.Equal(t, "line 1", mainCommits[0].Operations[0].Op.Content)
	assert.Equal(t, "line 2", mainCommits[1].Operations[0].Op.Content)
}

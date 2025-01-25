package commits

import (
	"evo/internal/crdt"
	"evo/internal/types"
	"path/filepath"
	"testing"

	"evo/internal/config"
	"evo/internal/signing"
)

func TestRevertCommit(t *testing.T) {
	// Create temp directory for test
	testDir := t.TempDir()

	t.Run("Revert_Insert", func(t *testing.T) {
		// Create original commit with insert operation
		ops := []types.ExtendedOp{
			{Op: crdt.Operation{Type: crdt.OpInsert, Content: "test"}},
		}
		commit, err := CreateCommit(testDir, "main", "Test commit", "Test User", "test@example.com", ops, false)
		if err != nil {
			t.Fatalf("Failed to create commit: %v", err)
		}

		// Revert the commit
		revertCommit, err := RevertCommit(testDir, "main", commit.ID)
		if err != nil {
			t.Fatalf("Failed to revert commit: %v", err)
		}

		// Verify revert operations
		if len(revertCommit.Operations) != len(commit.Operations) {
			t.Errorf("Expected %d operations, got %d", len(commit.Operations), len(revertCommit.Operations))
		}

		// Check that insert was reverted to delete
		if revertCommit.Operations[0].Op.Type != crdt.OpDelete {
			t.Error("Expected delete operation in revert commit")
		}
	})

	t.Run("Revert_Delete", func(t *testing.T) {
		// Create original commit with delete operation
		ops := []types.ExtendedOp{
			{Op: crdt.Operation{Type: crdt.OpDelete, Content: "test"}},
		}
		commit, err := CreateCommit(testDir, "main", "Test commit", "Test User", "test@example.com", ops, false)
		if err != nil {
			t.Fatalf("Failed to create commit: %v", err)
		}

		// Revert the commit
		revertCommit, err := RevertCommit(testDir, "main", commit.ID)
		if err != nil {
			t.Fatalf("Failed to revert commit: %v", err)
		}

		// Verify revert operations
		if len(revertCommit.Operations) != len(commit.Operations) {
			t.Errorf("Expected %d operations, got %d", len(commit.Operations), len(revertCommit.Operations))
		}

		// Check that delete was reverted to insert with original content
		if revertCommit.Operations[0].Op.Type != crdt.OpInsert {
			t.Error("Expected insert operation in revert commit")
		}
		if revertCommit.Operations[0].Op.Content != commit.Operations[0].Op.Content {
			t.Error("Content not preserved in revert operation")
		}
	})

	t.Run("Revert_Update", func(t *testing.T) {
		// Create original commit with update operation
		oldContent := "old"
		newContent := "new"
		ops := []types.ExtendedOp{
			{
				Op: crdt.Operation{
					Type:    crdt.OpUpdate,
					Content: newContent,
				},
				OldContent: oldContent,
			},
		}
		commit, err := CreateCommit(testDir, "main", "Test commit", "Test User", "test@example.com", ops, false)
		if err != nil {
			t.Fatalf("Failed to create commit: %v", err)
		}

		// Revert the commit
		revertCommit, err := RevertCommit(testDir, "main", commit.ID)
		if err != nil {
			t.Fatalf("Failed to revert commit: %v", err)
		}

		// Verify revert operations
		if len(revertCommit.Operations) != len(commit.Operations) {
			t.Errorf("Expected %d operations, got %d", len(commit.Operations), len(revertCommit.Operations))
		}

		// Check that update was reverted with old content
		if revertCommit.Operations[0].Op.Type != crdt.OpUpdate {
			t.Error("Expected update operation in revert commit")
		}
		if revertCommit.Operations[0].Op.Content != oldContent {
			t.Error("Old content not restored in revert operation")
		}
	})

	t.Run("Revert_Multiple_Operations", func(t *testing.T) {
		// Create original commit with multiple operations
		ops := []types.ExtendedOp{
			{Op: crdt.Operation{Type: crdt.OpInsert, Content: "test1"}},
			{Op: crdt.Operation{Type: crdt.OpInsert, Content: "test2"}},
		}
		commit, err := CreateCommit(testDir, "main", "Test commit", "Test User", "test@example.com", ops, false)
		if err != nil {
			t.Fatalf("Failed to create commit: %v", err)
		}

		// Revert the commit
		revertCommit, err := RevertCommit(testDir, "main", commit.ID)
		if err != nil {
			t.Fatalf("Failed to revert commit: %v", err)
		}

		// Verify revert operations
		if len(revertCommit.Operations) != len(commit.Operations) {
			t.Errorf("Expected %d operations, got %d", len(commit.Operations), len(revertCommit.Operations))
		}

		// Check that operations were reverted in reverse order
		for i := 0; i < len(revertCommit.Operations); i++ {
			if revertCommit.Operations[i].Op.Type != crdt.OpDelete {
				t.Error("Expected delete operation in revert commit")
			}
		}
	})
}

func TestSignedCommits(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "signing_key")

	// Set up config for test
	err := config.SetConfigValue(tmpDir, "signing.keyPath", keyPath)
	if err != nil {
		t.Fatalf("Failed to set config value: %v", err)
	}

	// Generate key pair for signing
	err = signing.GenerateKeyPair(tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	t.Run("Create_Signed_Commit", func(t *testing.T) {
		ops := []types.ExtendedOp{
			{Op: crdt.Operation{Type: crdt.OpInsert, Content: "test"}},
		}

		commit, err := CreateCommit(tmpDir, "main", "Test commit", "Test User", "test@example.com", ops, true)
		if err != nil {
			t.Fatalf("Failed to create signed commit: %v", err)
		}

		if commit.Signature == "" {
			t.Error("Commit not signed")
		}

		// Load and verify commit
		loaded, err := LoadCommit(tmpDir, "main", commit.ID)
		if err != nil {
			t.Fatalf("Failed to load commit: %v", err)
		}

		if loaded.Signature != commit.Signature {
			t.Error("Loaded commit signature does not match")
		}
	})

	t.Run("Create_Unsigned_Commit", func(t *testing.T) {
		ops := []types.ExtendedOp{
			{Op: crdt.Operation{Type: crdt.OpInsert, Content: "test"}},
		}

		commit, err := CreateCommit(tmpDir, "main", "Test commit", "Test User", "test@example.com", ops, false)
		if err != nil {
			t.Fatalf("Failed to create unsigned commit: %v", err)
		}

		if commit.Signature != "" {
			t.Error("Unsigned commit has signature")
		}

		// List commits and verify both signed and unsigned are present
		commits, err := ListCommits(tmpDir, "main")
		if err != nil {
			t.Fatalf("Failed to list commits: %v", err)
		}

		if len(commits) != 2 {
			t.Errorf("Expected 2 commits, got %d", len(commits))
		}
	})

	t.Run("Invalid_Signature", func(t *testing.T) {
		commit := &types.Commit{
			ID:        "test",
			Stream:    "main",
			Message:   "Test commit",
			Signature: "invalid",
		}

		// Save commit with invalid signature
		err := SaveCommit(tmpDir, commit)
		if err != nil {
			t.Fatalf("Failed to save commit: %v", err)
		}

		// Try to load commit - should fail verification
		_, err = LoadCommit(tmpDir, "main", commit.ID)
		if err == nil {
			t.Error("Expected error loading commit with invalid signature")
		}
	})
}

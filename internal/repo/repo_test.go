package repo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepo(t *testing.T) {
	// Create temp dir for testing
	tmpDir, err := os.MkdirTemp("", "evo-repo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("Init Repository", func(t *testing.T) {
		repoPath := filepath.Join(tmpDir, "test-repo")
		if err := InitRepo(repoPath); err != nil {
			t.Fatal(err)
		}

		// Verify directory structure
		dirs := []string{
			".evo",
			".evo/ops",
			".evo/commits",
			".evo/config",
			".evo/streams",
			".evo/chunks",
			".evo/lfs",
		}

		for _, dir := range dirs {
			path := filepath.Join(repoPath, dir)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("Directory %s not created", dir)
			}
		}

		// Verify HEAD file
		head, err := os.ReadFile(filepath.Join(repoPath, ".evo", "HEAD"))
		if err != nil {
			t.Fatal(err)
		}
		if string(head) != "main" {
			t.Errorf("Expected HEAD to be 'main', got '%s'", string(head))
		}
	})

	t.Run("Find Repository Root", func(t *testing.T) {
		// Create test repository
		repoPath := filepath.Join(tmpDir, "find-repo-test")
		if err := InitRepo(repoPath); err != nil {
			t.Fatal(err)
		}

		// Create nested directory structure
		nestedPath := filepath.Join(repoPath, "dir1", "dir2", "dir3")
		if err := os.MkdirAll(nestedPath, 0755); err != nil {
			t.Fatal(err)
		}

		// Test finding root from nested directory
		found, err := FindRepoRoot(nestedPath)
		if err != nil {
			t.Fatal(err)
		}
		if found != repoPath {
			t.Errorf("Expected root %s, got %s", repoPath, found)
		}

		// Test finding root from repository root
		found, err = FindRepoRoot(repoPath)
		if err != nil {
			t.Fatal(err)
		}
		if found != repoPath {
			t.Errorf("Expected root %s, got %s", repoPath, found)
		}

		// Test finding root from non-repository directory
		nonRepoPath := filepath.Join(tmpDir, "non-repo")
		if err := os.MkdirAll(nonRepoPath, 0755); err != nil {
			t.Fatal(err)
		}

		_, err = FindRepoRoot(nonRepoPath)
		if err == nil {
			t.Error("Expected error when finding root in non-repository")
		}
	})

	t.Run("Multiple Init Prevention", func(t *testing.T) {
		repoPath := filepath.Join(tmpDir, "multi-init-test")
		
		// First init should succeed
		if err := InitRepo(repoPath); err != nil {
			t.Fatal(err)
		}

		// Second init should fail
		if err := InitRepo(repoPath); err == nil {
			t.Error("Expected error on second init")
		}
	})

	t.Run("Init with Existing Files", func(t *testing.T) {
		repoPath := filepath.Join(tmpDir, "existing-files-test")
		
		// Create some existing files
		if err := os.MkdirAll(repoPath, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repoPath, "test.txt"), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		// Init should succeed with existing files
		if err := InitRepo(repoPath); err != nil {
			t.Fatal(err)
		}

		// Verify existing files are untouched
		if _, err := os.Stat(filepath.Join(repoPath, "test.txt")); os.IsNotExist(err) {
			t.Error("Existing file was removed during init")
		}
	})

	t.Run("Init Permission Handling", func(t *testing.T) {
		repoPath := filepath.Join(tmpDir, "permission-test")
		
		// Create directory with restricted permissions
		if err := os.MkdirAll(repoPath, 0444); err != nil {
			t.Fatal(err)
		}

		// Init should fail with insufficient permissions
		err := InitRepo(repoPath)
		if err == nil {
			t.Error("Expected error with insufficient permissions")
		}

		// Reset permissions
		if err := os.Chmod(repoPath, 0755); err != nil {
			t.Fatal(err)
		}

		// Init should now succeed
		if err := InitRepo(repoPath); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("Service Initialization", func(t *testing.T) {
		repoPath := filepath.Join(tmpDir, "service-test")
		
		if err := InitRepo(repoPath); err != nil {
			t.Fatal(err)
		}

		// Verify services are running by checking their directories
		services := []string{
			".evo/chunks",  // LFS chunks directory
			".evo/lfs",     // LFS metadata directory
		}

		for _, dir := range services {
			path := filepath.Join(repoPath, dir)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("Service directory %s not created", dir)
			}
		}
	})
}

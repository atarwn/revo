package status

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestRepo(t *testing.T) string {
	// Create a temporary directory for the test repository
	tmpDir, err := os.MkdirTemp("", "evo-status-test")
	if err != nil {
		t.Fatal(err)
	}

	// Create .evo directory structure
	evoDir := filepath.Join(tmpDir, ".evo")
	for _, dir := range []string{
		"objects",
		"streams",
		"commits",
	} {
		if err := os.MkdirAll(filepath.Join(evoDir, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create main stream
	if err := os.WriteFile(filepath.Join(evoDir, "streams", "main"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	// Set current stream
	if err := os.WriteFile(filepath.Join(evoDir, "HEAD"), []byte("main"), 0644); err != nil {
		t.Fatal(err)
	}

	return tmpDir
}

func TestGetStatus(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer os.RemoveAll(repoPath)

	// Create .evo-ignore file first
	ignoreContent := `
# Test ignore file
*.log
build/
**/*.tmp
`
	if err := os.WriteFile(filepath.Join(repoPath, ".evo-ignore"), []byte(ignoreContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create some test files
	files := map[string]string{
		"file1.txt":     "content1",
		"file2.txt":     "content2",
		"dir/file3.txt": "content3",
	}

	for path, content := range files {
		fullPath := filepath.Join(repoPath, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create some files that should be ignored
	ignoredFiles := map[string]string{
		"test.log":         "log content",
		"build/output.txt": "build output",
		"temp.tmp":         "temporary file",
	}

	for path, content := range ignoredFiles {
		fullPath := filepath.Join(repoPath, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Get initial status (before index exists)
	status, err := GetStatus(repoPath)
	if err != nil {
		t.Fatal(err)
	}

	// Verify all non-ignored files are marked as new
	newFiles := make(map[string]bool)
	for _, f := range status.Files {
		newFiles[f.Path] = true
		if f.Status != "new" {
			t.Errorf("Expected file %s to be new, got %s", f.Path, f.Status)
		}
		// Verify no ignored files are included
		for ignoredPath := range ignoredFiles {
			if f.Path == ignoredPath {
				t.Errorf("Found ignored file in status: %s", f.Path)
			}
		}
	}

	// Check that we found all expected files
	for path := range files {
		if !newFiles[path] {
			t.Errorf("Expected to find %s in status, but it was missing", path)
		}
	}

	// Create object files first
	objects := map[string]string{
		"id1": "content1",
		"id2": "content2",
	}

	for id, content := range objects {
		objPath := filepath.Join(repoPath, ".evo", "objects", id)
		if err := os.WriteFile(objPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create index file after objects
	indexContent := map[string]string{
		"file1.txt": "id1",
		"file2.txt": "id2",
	}

	var indexLines []string
	for path, id := range indexContent {
		indexLines = append(indexLines, path+":"+id)
	}
	if err := os.WriteFile(filepath.Join(repoPath, ".evo", "index"), []byte(strings.Join(indexLines, "\n")+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Modify file2.txt
	if err := os.WriteFile(filepath.Join(repoPath, "file2.txt"), []byte("modified content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Get status again
	status, err = GetStatus(repoPath)
	if err != nil {
		t.Fatal(err)
	}

	// Verify status
	expectedStatuses := map[string]string{
		"file2.txt":     "modified",
		"dir/file3.txt": "new",
	}

	foundFiles := make(map[string]bool)
	for _, f := range status.Files {
		foundFiles[f.Path] = true
		expectedStatus, exists := expectedStatuses[f.Path]
		if !exists {
			t.Errorf("Unexpected file in status: %s", f.Path)
			continue
		}
		if f.Status != expectedStatus {
			t.Errorf("Expected file %s to be %s, got %s", f.Path, expectedStatus, f.Status)
		}
	}

	// Check that we found all expected files
	for path := range expectedStatuses {
		if !foundFiles[path] {
			t.Errorf("Expected to find %s in status, but it was missing", path)
		}
	}

	// Test rename detection
	if err := os.Rename(filepath.Join(repoPath, "file1.txt"), filepath.Join(repoPath, "file1_renamed.txt")); err != nil {
		t.Fatal(err)
	}

	status, err = GetStatus(repoPath)
	if err != nil {
		t.Fatal(err)
	}

	foundRename := false
	for _, f := range status.Files {
		if f.Status == "renamed" && f.Path == "file1_renamed.txt" && f.OldPath == "file1.txt" {
			foundRename = true
			break
		}
	}
	if !foundRename {
		t.Error("Failed to detect renamed file")
	}

	// Test deletion detection
	if err := os.Remove(filepath.Join(repoPath, "file2.txt")); err != nil {
		t.Fatal(err)
	}

	status, err = GetStatus(repoPath)
	if err != nil {
		t.Fatal(err)
	}

	foundDelete := false
	for _, f := range status.Files {
		if f.Status == "deleted" && f.Path == "file2.txt" {
			foundDelete = true
			break
		}
	}
	if !foundDelete {
		t.Error("Failed to detect deleted file")
	}
}

func TestGetStatusErrors(t *testing.T) {
	// Test with non-existent directory
	_, err := GetStatus("/nonexistent/path")
	if err == nil {
		t.Error("Expected error when repository path doesn't exist")
	}

	// Test with invalid repository (no .evo directory)
	tmpDir, err := os.MkdirTemp("", "invalid-repo")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_, err = GetStatus(tmpDir)
	if err == nil {
		t.Error("Expected error when .evo directory doesn't exist")
	}

	// Test with invalid HEAD file
	repoPath := setupTestRepo(t)
	defer os.RemoveAll(repoPath)

	if err := os.WriteFile(filepath.Join(repoPath, ".evo", "HEAD"), []byte("invalid-stream\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = GetStatus(repoPath)
	if err == nil {
		t.Error("Expected error when HEAD points to non-existent stream")
	}
}

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   *RepoStatus
		contains []string
		excludes []string
	}{
		{
			name: "Empty status",
			status: &RepoStatus{
				CurrentStream: "main",
				Files:         []FileStatus{},
			},
			contains: []string{
				"On stream main",
				"nothing to commit, working tree clean",
			},
			excludes: []string{
				"Changes not staged",
				"Untracked files",
				"Deleted files",
				"Renamed files",
			},
		},
		{
			name: "Modified files only",
			status: &RepoStatus{
				CurrentStream: "main",
				Files: []FileStatus{
					{Path: "file1.txt", Status: "modified"},
					{Path: "dir/file2.txt", Status: "modified"},
				},
			},
			contains: []string{
				"On stream main",
				"Changes not staged for commit:",
				"modified: file1.txt",
				"modified: dir/file2.txt",
			},
			excludes: []string{
				"nothing to commit",
				"Untracked files",
				"Deleted files",
				"Renamed files",
			},
		},
		{
			name: "All status types",
			status: &RepoStatus{
				CurrentStream: "feature",
				Files: []FileStatus{
					{Path: "file1.txt", Status: "modified"},
					{Path: "file2.txt", Status: "new"},
					{Path: "file3.txt", Status: "deleted"},
					{Path: "new.txt", Status: "renamed", OldPath: "old.txt"},
				},
			},
			contains: []string{
				"On stream feature",
				"Changes not staged for commit:",
				"modified: file1.txt",
				"Untracked files:",
				"file2.txt",
				"Deleted files:",
				"file3.txt",
				"Renamed files:",
				"old.txt -> new.txt",
			},
			excludes: []string{
				"nothing to commit",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := FormatStatus(tt.status)

			for _, s := range tt.contains {
				if !strings.Contains(output, s) {
					t.Errorf("Expected output to contain %q", s)
				}
			}

			for _, s := range tt.excludes {
				if strings.Contains(output, s) {
					t.Errorf("Expected output to not contain %q", s)
				}
			}
		})
	}
}

func TestLoadIndex(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer os.RemoveAll(repoPath)

	// Test loading non-existent index
	idx, err := loadIndex(repoPath)
	if err != nil {
		t.Errorf("Expected no error when index doesn't exist, got %v", err)
	}
	if len(idx) != 0 {
		t.Errorf("Expected empty index, got %v", idx)
	}

	// Test loading valid index
	indexContent := "file1.txt:id1\nfile2.txt:id2\n"
	if err := os.WriteFile(filepath.Join(repoPath, ".evo", "index"), []byte(indexContent), 0644); err != nil {
		t.Fatal(err)
	}

	idx, err = loadIndex(repoPath)
	if err != nil {
		t.Errorf("Failed to load index: %v", err)
	}

	expected := map[string]string{
		"file1.txt": "id1",
		"file2.txt": "id2",
	}

	if len(idx) != len(expected) {
		t.Errorf("Expected %d entries, got %d", len(expected), len(idx))
	}

	for path, id := range expected {
		if idx[path] != id {
			t.Errorf("Expected %s -> %s, got %s -> %s", path, id, path, idx[path])
		}
	}

	// Test loading malformed index
	malformedContent := "file1.txt:id1\nmalformed-line\nfile2.txt:id2\n"
	if err := os.WriteFile(filepath.Join(repoPath, ".evo", "index"), []byte(malformedContent), 0644); err != nil {
		t.Fatal(err)
	}

	idx, err = loadIndex(repoPath)
	if err != nil {
		t.Errorf("Failed to load index with malformed line: %v", err)
	}

	if len(idx) != 2 {
		t.Errorf("Expected 2 valid entries, got %d", len(idx))
	}
}

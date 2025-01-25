package ignore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadIgnoreFile(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "evo-ignore-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case 1: No .evo-ignore file
	il, err := LoadIgnoreFile(tmpDir)
	if err != nil {
		t.Errorf("Expected no error when .evo-ignore doesn't exist, got %v", err)
	}
	if len(il.patterns) != 0 {
		t.Errorf("Expected empty patterns list, got %v", il.patterns)
	}

	// Test case 2: With .evo-ignore file
	ignoreContent := `
# Comment line
*.log
build/
**/*.tmp
test/*.txt
node_modules/
*.bak
!important.bak
`
	ignorePath := filepath.Join(tmpDir, ".evo-ignore")
	if err := os.WriteFile(ignorePath, []byte(ignoreContent), 0644); err != nil {
		t.Fatal(err)
	}

	il, err = LoadIgnoreFile(tmpDir)
	if err != nil {
		t.Errorf("Failed to load .evo-ignore file: %v", err)
	}

	expectedPatterns := []string{
		"*.log",
		"build/**",
		"**/*.tmp",
		"test/*.txt",
		"node_modules/**",
		"*.bak",
		"!important.bak",
	}
	patterns := il.GetPatterns()
	if len(patterns) != len(expectedPatterns) {
		t.Errorf("Expected %d patterns, got %d", len(expectedPatterns), len(patterns))
	}

	for i, pattern := range patterns {
		if pattern != expectedPatterns[i] {
			t.Errorf("Pattern %d: expected %s, got %s", i, expectedPatterns[i], pattern)
		}
	}

	// Test case 3: Invalid file permissions
	if err := os.Chmod(ignorePath, 0000); err != nil {
		t.Fatal(err)
	}
	_, err = LoadIgnoreFile(tmpDir)
	if err == nil {
		t.Error("Expected error when loading file with no permissions")
	}
}

func TestIsIgnored(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		paths    map[string]bool // path -> should be ignored
	}{
		{
			name:     "Empty patterns",
			patterns: []string{},
			paths: map[string]bool{
				"file.txt":     false,
				".evo/config":  true, // .evo is always ignored
				".evo/objects": true,
			},
		},
		{
			name: "Simple glob patterns",
			patterns: []string{
				"*.log",
				"*.tmp",
			},
			paths: map[string]bool{
				"test.log":      true,
				"logs/test.log": true,
				"test.txt":      false,
				"test.tmp":      true,
			},
		},
		{
			name: "Directory patterns",
			patterns: []string{
				"build/",
				"node_modules/",
				"test/fixtures/",
			},
			paths: map[string]bool{
				"build/output.txt":          true,
				"build/temp/file.txt":       true,
				"src/build/file.txt":        false,
				"node_modules/package.json": true,
				"test/fixtures/data.json":   true,
				"test/file.txt":             false,
			},
		},
		{
			name: "Double-star patterns",
			patterns: []string{
				"**/*.tmp",
				"**/vendor/**",
				"**/__pycache__/**",
			},
			paths: map[string]bool{
				"file.tmp":                    true,
				"temp/file.tmp":               true,
				"a/b/c/file.tmp":              true,
				"vendor/lib.js":               true,
				"src/vendor/lib.js":           true,
				"src/__pycache__/module.pyc":  true,
				"test/__pycache__/cache.json": true,
			},
		},
		{
			name: "Complex patterns",
			patterns: []string{
				"*.{log,tmp}",
				"**/{test,mock}_*.go",
				"**/.DS_Store",
			},
			paths: map[string]bool{
				"error.log":           true,
				"temp.tmp":            true,
				"test_handler.go":     true,
				"mock_service.go":     true,
				"internal/test_db.go": true,
				".DS_Store":           true,
				"src/.DS_Store":       true,
				"handler.go":          false,
				"service_test.go":     false,
			},
		},
		{
			name: "Path normalization",
			patterns: []string{
				"build/",
				"**/temp/**",
			},
			paths: map[string]bool{
				"build/file.txt":        true,
				"./build/file.txt":      true,
				"build/../build/file":   true,
				"temp/file.txt":         true,
				"./temp/file.txt":       true,
				"a/temp/b/file.txt":     true,
				"./a/temp/b/file.txt":   true,
				"../repo/temp/file.txt": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			il := &IgnoreList{patterns: tt.patterns}
			for path, shouldIgnore := range tt.paths {
				if got := il.IsIgnored(path); got != shouldIgnore {
					t.Errorf("IsIgnored(%q) = %v, want %v", path, got, shouldIgnore)
				}
			}
		})
	}
}

func TestAddPattern(t *testing.T) {
	il := &IgnoreList{}

	// Test adding various pattern types
	patterns := []struct {
		input    string
		expected string
	}{
		{"*.log", "*.log"},
		{"build/", "build/**"},
		{"node_modules/", "node_modules/**"},
		{"**/*.tmp", "**/*.tmp"},
		{"test/*.txt", "test/*.txt"},
		{".env", ".env"},
		{"dist/", "dist/**"},
	}

	for _, p := range patterns {
		il.AddPattern(p.input)
		found := false
		for _, pattern := range il.patterns {
			if pattern == p.expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Pattern %q not found in patterns after AddPattern, expected %q", p.input, p.expected)
		}
	}

	// Test pattern order preservation
	il = &IgnoreList{}
	var expectedPatterns []string
	for _, p := range patterns {
		il.AddPattern(p.input)
		expectedPatterns = append(expectedPatterns, p.expected)
	}

	actualPatterns := il.GetPatterns()
	if len(actualPatterns) != len(expectedPatterns) {
		t.Errorf("Expected %d patterns, got %d", len(expectedPatterns), len(actualPatterns))
	}

	for i, pattern := range actualPatterns {
		if pattern != expectedPatterns[i] {
			t.Errorf("Pattern at index %d: expected %q, got %q", i, expectedPatterns[i], pattern)
		}
	}
}

func TestGetPatterns(t *testing.T) {
	// Test that GetPatterns returns a copy of the patterns slice
	il := &IgnoreList{patterns: []string{"*.log", "build/**", "**/*.tmp"}}

	patterns1 := il.GetPatterns()
	patterns2 := il.GetPatterns()

	// Verify both slices have the same content
	if len(patterns1) != len(patterns2) {
		t.Errorf("Pattern slices have different lengths: %d vs %d", len(patterns1), len(patterns2))
	}
	for i := range patterns1 {
		if patterns1[i] != patterns2[i] {
			t.Errorf("Pattern mismatch at index %d: %q vs %q", i, patterns1[i], patterns2[i])
		}
	}

	// Modify the first slice and verify it doesn't affect the second
	patterns1[0] = "modified"
	if patterns1[0] == patterns2[0] {
		t.Error("Modifying one pattern slice affected the other")
	}

	// Verify the original patterns are unchanged
	originalPatterns := il.GetPatterns()
	if originalPatterns[0] != "*.log" {
		t.Errorf("Original patterns were modified: expected %q, got %q", "*.log", originalPatterns[0])
	}
}

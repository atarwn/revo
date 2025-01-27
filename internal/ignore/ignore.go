package ignore

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// IgnoreList represents a collection of ignore patterns
type IgnoreList struct {
	patterns []string
}

// LoadIgnoreFile reads and parses the .evo-ignore file from the given repository path
func LoadIgnoreFile(repoPath string) (*IgnoreList, error) {
	ignorePath := filepath.Join(repoPath, ".evo-ignore")
	file, err := os.Open(ignorePath)
	if os.IsNotExist(err) {
		return &IgnoreList{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		pattern := strings.TrimSpace(scanner.Text())
		if pattern != "" && !strings.HasPrefix(pattern, "#") {
			// Handle directory patterns
			if strings.HasSuffix(pattern, "/") {
				pattern = strings.TrimSuffix(pattern, "/")
				if !strings.Contains(pattern, "**") {
					pattern = pattern + "/**"
				}
			}
			patterns = append(patterns, pattern)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &IgnoreList{patterns: patterns}, nil
}

// IsIgnored checks if a given path should be ignored based on the ignore patterns
func (il *IgnoreList) IsIgnored(path string) bool {
	// Always ignore .evo directory
	if strings.HasPrefix(path, ".evo") {
		return true
	}

	// Clean and normalize the path
	path = filepath.ToSlash(filepath.Clean(path))
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimPrefix(path, "../")

	for _, pattern := range il.patterns {
		// Handle negation patterns
		if strings.HasPrefix(pattern, "!") {
			matched, err := doublestar.Match(pattern[1:], path)
			if err == nil && matched {
				return false
			}
			continue
		}

		// For directory patterns ending with /**, try prefix matching first
		if strings.HasSuffix(pattern, "/**") {
			base := strings.TrimSuffix(pattern, "/**")
			if path == base || strings.HasPrefix(path, base+"/") {
				return true
			}
		}

		// Try matching the pattern directly
		matched, err := doublestar.Match(pattern, path)
		if err == nil && matched {
			return true
		}

		// Try matching with **/ prefix
		if !strings.HasPrefix(pattern, "**/") {
			matched, err := doublestar.Match("**/"+pattern, path)
			if err == nil && matched {
				return true
			}
		}

		// For directory patterns without /**, try matching with /** suffix
		if !strings.HasSuffix(pattern, "/**") {
			// Try with /** suffix
			matched, err := doublestar.Match(pattern+"/**", path)
			if err == nil && matched {
				return true
			}

			// Try with **/ prefix and /** suffix
			matched, err = doublestar.Match("**/"+pattern+"/**", path)
			if err == nil && matched {
				return true
			}

			// Try with /** suffix for each path component
			parts := strings.Split(path, "/")
			for i := range parts {
				prefix := strings.Join(parts[:i+1], "/")
				if prefix == pattern {
					return true
				}
				if strings.HasSuffix(pattern, "/") {
					pattern = strings.TrimSuffix(pattern, "/")
					if prefix == pattern {
						return true
					}
				}
			}
		}
	}

	return false
}

// AddPattern adds a new ignore pattern
func (il *IgnoreList) AddPattern(pattern string) {
	// Handle directory patterns
	if strings.HasSuffix(pattern, "/") {
		pattern = strings.TrimSuffix(pattern, "/")
		if !strings.Contains(pattern, "**") {
			pattern = pattern + "/**"
		}
	}
	il.patterns = append(il.patterns, pattern)
}

// GetPatterns returns all current ignore patterns
func (il *IgnoreList) GetPatterns() []string {
	return append([]string{}, il.patterns...)
}

package status

import (
	"bufio"
	"evo/internal/ignore"
	"evo/internal/streams"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type FileStatus struct {
	Path    string
	Status  string // "modified", "new", "deleted", "renamed"
	OldPath string // only set for renamed files
}

type RepoStatus struct {
	CurrentStream string
	Files         []FileStatus
}

// loadIndex loads the index file directly to avoid dependency cycles
func loadIndex(repoPath string) (map[string]string, error) {
	indexPath := filepath.Join(repoPath, ".evo", "index")
	file, err := os.Open(indexPath)
	if os.IsNotExist(err) {
		return make(map[string]string), nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	idx := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ":")
		if len(parts) == 2 {
			idx[parts[0]] = parts[1]
		}
	}
	return idx, scanner.Err()
}

func GetStatus(repoPath string) (*RepoStatus, error) {
	// Get current stream
	stream, err := streams.CurrentStream(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get current stream: %w", err)
	}

	// Verify stream exists
	streamPath := filepath.Join(repoPath, ".evo", "streams", stream)
	if _, err := os.Stat(streamPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("stream %s does not exist", stream)
	}

	// Load ignore patterns
	ignoreList, err := ignore.LoadIgnoreFile(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load ignore file: %w", err)
	}

	// Get current index state
	idx, err := loadIndex(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load index: %w", err)
	}

	status := &RepoStatus{
		CurrentStream: stream,
	}

	// Track processed files and their content hashes
	processedFiles := make(map[string]string) // path -> content hash

	// Walk the repository to find new and modified files
	err = filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(repoPath, path)
		if err != nil {
			return err
		}

		// Skip the .evo directory
		if strings.HasPrefix(relPath, ".evo") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip ignored files
		if ignoreList.IsIgnored(relPath) {
			return nil
		}

		// Read current file content
		currentContent, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Store content hash for rename detection
		processedFiles[relPath] = string(currentContent)

		// Check if file is in index
		fileID, exists := idx[relPath]
		if !exists {
			// Check if this might be a renamed file
			var foundRename bool
			for oldPath, oldID := range idx {
				if oldPath == relPath {
					continue
				}
				storedContent, err := os.ReadFile(filepath.Join(repoPath, ".evo", "objects", oldID))
				if err == nil && string(currentContent) == string(storedContent) {
					// Found a rename
					status.Files = append(status.Files, FileStatus{
						Path:    relPath,
						Status:  "renamed",
						OldPath: oldPath,
					})
					foundRename = true
					break
				}
			}
			if !foundRename {
				// New file
				status.Files = append(status.Files, FileStatus{
					Path:   relPath,
					Status: "new",
				})
			}
			return nil
		}

		// Check if file has been modified
		storedContent, err := os.ReadFile(filepath.Join(repoPath, ".evo", "objects", fileID))
		if err != nil || string(currentContent) != string(storedContent) {
			status.Files = append(status.Files, FileStatus{
				Path:   relPath,
				Status: "modified",
			})
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk repository: %w", err)
	}

	// Check for deleted files
	for path, id := range idx {
		// Skip if file was already processed
		if _, exists := processedFiles[path]; exists {
			continue
		}

		// Check if file was renamed by looking for matching content
		var renamed bool
		for newPath, content := range processedFiles {
			storedContent, err := os.ReadFile(filepath.Join(repoPath, ".evo", "objects", id))
			if err == nil && content == string(storedContent) {
				// Found a rename
				status.Files = append(status.Files, FileStatus{
					Path:    newPath,
					Status:  "renamed",
					OldPath: path,
				})
				renamed = true
				break
			}
		}

		if !renamed {
			status.Files = append(status.Files, FileStatus{
				Path:   path,
				Status: "deleted",
			})
		}
	}

	// Sort files by status and path
	sort.Slice(status.Files, func(i, j int) bool {
		if status.Files[i].Status != status.Files[j].Status {
			return status.Files[i].Status < status.Files[j].Status
		}
		return status.Files[i].Path < status.Files[j].Path
	})

	return status, nil
}

// FormatStatus returns a formatted string representation of the repository status
func FormatStatus(status *RepoStatus) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("On stream %s\n\n", status.CurrentStream))

	if len(status.Files) == 0 {
		sb.WriteString("nothing to commit, working tree clean\n")
		return sb.String()
	}

	// Group files by status
	var modified, new, deleted, renamed []FileStatus
	for _, f := range status.Files {
		switch f.Status {
		case "modified":
			modified = append(modified, f)
		case "new":
			new = append(new, f)
		case "deleted":
			deleted = append(deleted, f)
		case "renamed":
			renamed = append(renamed, f)
		}
	}

	if len(modified) > 0 {
		sb.WriteString("Changes not staged for commit:\n")
		for _, f := range modified {
			sb.WriteString(fmt.Sprintf("  modified: %s\n", f.Path))
		}
		sb.WriteString("\n")
	}

	if len(new) > 0 {
		sb.WriteString("Untracked files:\n")
		for _, f := range new {
			sb.WriteString(fmt.Sprintf("  %s\n", f.Path))
		}
		sb.WriteString("\n")
	}

	if len(deleted) > 0 {
		sb.WriteString("Deleted files:\n")
		for _, f := range deleted {
			sb.WriteString(fmt.Sprintf("  %s\n", f.Path))
		}
		sb.WriteString("\n")
	}

	if len(renamed) > 0 {
		sb.WriteString("Renamed files:\n")
		for _, f := range renamed {
			sb.WriteString(fmt.Sprintf("  %s -> %s\n", f.OldPath, f.Path))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

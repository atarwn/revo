package streams

import (
	"evo/internal/commits"
	"evo/internal/crdt"
	"evo/internal/repo"
	"evo/internal/types"
	"fmt"
	"path/filepath"
)

// MergeFilter defines criteria for selecting operations during a partial merge
type MergeFilter struct {
	FileIDs []string      // Only merge operations for these files
	OpTypes []crdt.OpType // Only merge these operation types
}

// PartialMerge merges selected operations from source to target stream based on filter criteria
func PartialMerge(repoPath, source, target string, filter MergeFilter) error {
	srcCommits, err := ListCommits(repoPath, source)
	if err != nil {
		return err
	}

	tgtCommits, err := ListCommits(repoPath, target)
	if err != nil {
		return err
	}

	// Build map of target commits for quick lookup
	tgtMap := make(map[string]bool)
	for _, c := range tgtCommits {
		tgtMap[c.ID] = true
	}

	// For empty filter, merge all operations into a single commit
	if len(filter.FileIDs) == 0 && len(filter.OpTypes) == 0 {
		var allOps []commits.ExtendedOp
		var lastCommit *types.Commit

		for _, sc := range srcCommits {
			lastCommit = &sc
			for _, op := range sc.Operations {
				newOp := op
				newOp.Op.Stream = target
				allOps = append(allOps, newOp)
			}
		}

		if len(allOps) > 0 && lastCommit != nil {
			// Create single commit with all operations
			newCommit := types.Commit{
				ID:         lastCommit.ID,
				Stream:     target,
				Message:    fmt.Sprintf("[merge] %s", lastCommit.Message),
				Operations: allOps,
				Timestamp:  lastCommit.Timestamp,
			}

			// Save the commit
			commitPath := filepath.Join(repoPath, repo.EvoDir, "commits", target)
			if err := commits.SaveCommitFile(commitPath, &newCommit); err != nil {
				return err
			}

			// Replicate all operations
			if err := replicateOps(repoPath, target, allOps); err != nil {
				return err
			}
		}

		return nil
	}

	// Process each source commit for non-empty filters
	for _, sc := range srcCommits {
		// Filter operations based on criteria
		var filteredOps []commits.ExtendedOp
		for _, op := range sc.Operations {
			if shouldIncludeOp(op, filter) {
				newOp := op
				newOp.Op.Stream = target
				filteredOps = append(filteredOps, newOp)
			}
		}

		// Skip if no operations match filter
		if len(filteredOps) == 0 {
			continue
		}

		// Create new commit with filtered operations
		newCommit := types.Commit{
			ID:         sc.ID,
			Stream:     target,
			Message:    fmt.Sprintf("[merge] %s", sc.Message),
			Operations: filteredOps,
			Timestamp:  sc.Timestamp,
		}

		// Save the commit
		commitPath := filepath.Join(repoPath, repo.EvoDir, "commits", target)
		if err := commits.SaveCommitFile(commitPath, &newCommit); err != nil {
			return err
		}

		// Replicate filtered operations
		if err := replicateOps(repoPath, target, filteredOps); err != nil {
			return err
		}
	}

	return nil
}

// shouldIncludeOp checks if an operation matches the filter criteria
func shouldIncludeOp(op commits.ExtendedOp, filter MergeFilter) bool {
	// If no filters specified, include everything
	if len(filter.FileIDs) == 0 && len(filter.OpTypes) == 0 {
		return true
	}

	// Check file ID filter
	if len(filter.FileIDs) > 0 {
		fileMatch := false
		for _, fid := range filter.FileIDs {
			if op.Op.FileID.String() == fid {
				fileMatch = true
				break
			}
		}
		if !fileMatch {
			return false
		}
	}

	// Check operation type filter
	if len(filter.OpTypes) > 0 {
		typeMatch := false
		for _, ot := range filter.OpTypes {
			if op.Op.Type == ot {
				typeMatch = true
				break
			}
		}
		if !typeMatch {
			return false
		}
	}

	return true
}

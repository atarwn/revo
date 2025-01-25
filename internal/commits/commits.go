package commits

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"evo/internal/crdt"
	"evo/internal/ops"
	"evo/internal/signing"
	"evo/internal/types"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ExtendedOp includes oldContent for update ops
type ExtendedOp = types.ExtendedOp

// CreateCommit creates a new commit with the given operations
func CreateCommit(repoPath, stream, message, authorName, authorEmail string, ops []types.ExtendedOp, sign bool) (*types.Commit, error) {
	commit := &types.Commit{
		ID:          uuid.New().String(),
		Stream:      stream,
		Message:     message,
		AuthorName:  authorName,
		AuthorEmail: authorEmail,
		Timestamp:   time.Now().UTC(),
		Operations:  ops,
	}

	// Sign commit if requested
	if sign {
		sig, err := signing.SignCommit(commit, repoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to sign commit: %w", err)
		}
		commit.Signature = sig

		// Verify signature immediately
		valid, err := signing.VerifyCommit(commit, repoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to verify commit signature: %w", err)
		}
		if !valid {
			return nil, fmt.Errorf("commit signature verification failed")
		}
	}

	// Save commit
	if err := SaveCommit(repoPath, commit); err != nil {
		return nil, fmt.Errorf("failed to save commit: %w", err)
	}

	return commit, nil
}

// LoadCommit loads a commit from disk
func LoadCommit(repoPath, stream, commitID string) (*types.Commit, error) {
	commitPath := filepath.Join(repoPath, ".evo", "commits", stream, commitID+".bin")
	data, err := os.ReadFile(commitPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read commit file: %w", err)
	}

	var commit types.Commit
	if err := json.Unmarshal(data, &commit); err != nil {
		return nil, fmt.Errorf("failed to unmarshal commit: %w", err)
	}

	// Verify signature if present
	if commit.Signature != "" {
		valid, err := signing.VerifyCommit(&commit, repoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to verify commit signature: %w", err)
		}
		if !valid {
			return nil, fmt.Errorf("commit signature verification failed")
		}
	}

	return &commit, nil
}

// SaveCommit saves a commit to disk
func SaveCommit(repoPath string, commit *types.Commit) error {
	commitDir := filepath.Join(repoPath, ".evo", "commits", commit.Stream)
	if err := os.MkdirAll(commitDir, 0755); err != nil {
		return fmt.Errorf("failed to create commit directory: %w", err)
	}

	data, err := json.Marshal(commit)
	if err != nil {
		return fmt.Errorf("failed to marshal commit: %w", err)
	}

	commitPath := filepath.Join(commitDir, commit.ID+".bin")
	if err := os.WriteFile(commitPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write commit file: %w", err)
	}

	return nil
}

// gatherNewOps => find ops not in prior commits, augment 'update' ops with oldContent
func gatherNewOps(repoPath, stream string) ([]ExtendedOp, error) {
	all, err := ListCommits(repoPath, stream)
	if err != nil {
		return nil, err
	}
	known := make(map[string]bool)
	for _, cc := range all {
		for _, eop := range cc.Operations {
			known[opKey(eop.Op)] = true
		}
	}

	// load all current ops
	var allOps []ExtendedOp
	opsDir := filepath.Join(repoPath, ".evo", "ops", stream)
	if err := filepath.WalkDir(opsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".bin" {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			var size int64
			if err := binary.Read(f, binary.LittleEndian, &size); err != nil {
				return err
			}
			data := make([]byte, size)
			if _, err := f.Read(data); err != nil {
				return err
			}
			var op crdt.Operation
			if err := json.Unmarshal(data, &op); err != nil {
				return err
			}
			allOps = append(allOps, ExtendedOp{Op: op})
		}
		return nil
	}); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// build doc states to find old text
	docStates := buildDocStates(repoPath, stream)
	var newEops []ExtendedOp
	for _, op := range allOps {
		k := opKey(op.Op)
		if !known[k] {
			var old string
			if op.Op.Type == crdt.OpUpdate {
				old = findOldContent(docStates, op.Op.LineID)
			}
			newEops = append(newEops, ExtendedOp{
				Op:         op.Op,
				OldContent: old,
			})
		}
	}
	sort.Slice(newEops, func(i, j int) bool {
		return newEops[i].Op.LessThan(&newEops[j].Op)
	})
	return newEops, nil
}

func opKey(op crdt.Operation) string {
	return fmt.Sprintf("%d_%s_%s", op.Lamport, op.NodeID.String(), op.LineID.String())
}

func buildDocStates(repoPath, stream string) map[uuid.UUID]map[uuid.UUID]string {
	res := make(map[uuid.UUID]map[uuid.UUID]string)
	root := filepath.Join(repoPath, ".evo", "ops", stream)
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".bin") {
			fn := filepath.Base(path)
			fidStr := strings.TrimSuffix(fn, ".bin")
			fid, err := uuid.Parse(fidStr)
			if err == nil {
				ops2, _ := ops.LoadAllOps(path)
				doc := crdt.NewRGA()
				for _, op := range ops2 {
					doc.Apply(op)
				}
				res[fid] = doc.LineMap()
			}
		}
		return nil
	}); err != nil && !os.IsNotExist(err) {
		return nil
	}
	return res
}

func findOldContent(ds map[uuid.UUID]map[uuid.UUID]string, lineID uuid.UUID) string {
	for _, linesMap := range ds {
		if txt, ok := linesMap[lineID]; ok {
			return txt
		}
	}
	return ""
}

// ListCommits returns all commits in a stream, sorted by timestamp
func ListCommits(repoPath, stream string) ([]types.Commit, error) {
	commitDir := filepath.Join(repoPath, ".evo", "commits", stream)
	entries, err := os.ReadDir(commitDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read commit directory: %w", err)
	}

	var commits []types.Commit
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".bin") {
			commit, err := LoadCommit(repoPath, stream, strings.TrimSuffix(entry.Name(), ".bin"))
			if err != nil {
				return nil, fmt.Errorf("failed to load commit %s: %w", entry.Name(), err)
			}
			commits = append(commits, *commit)
		}
	}

	// Sort by timestamp
	sort.Slice(commits, func(i, j int) bool {
		return commits[i].Timestamp.Before(commits[j].Timestamp)
	})

	return commits, nil
}

func saveCommit(repoPath string, c *types.Commit) error {
	dir := filepath.Join(repoPath, ".evo", "commits", c.Stream)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	fp := filepath.Join(dir, c.ID+".bin")
	b, _ := json.Marshal(c)
	sz := make([]byte, 4)
	binary.BigEndian.PutUint32(sz, uint32(len(b)))
	f, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer f.Close()
	f.Write(sz)
	f.Write(b)
	return nil
}

func SaveCommitFile(dir string, c *types.Commit) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	fp := filepath.Join(dir, c.ID+".bin")
	b, _ := json.Marshal(c)
	sz := make([]byte, 4)
	binary.BigEndian.PutUint32(sz, uint32(len(b)))
	f, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer f.Close()
	f.Write(sz)
	f.Write(b)
	return nil
}

func loadCommit(fp string) (*types.Commit, error) {
	f, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	szBuf := make([]byte, 4)
	if _, err := f.Read(szBuf); err != nil {
		return nil, err
	}
	sz := binary.BigEndian.Uint32(szBuf)
	data := make([]byte, sz)
	if _, err := f.Read(data); err != nil {
		return nil, err
	}
	var c types.Commit
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// RevertCommit creates a new commit that reverts the changes in the specified commit
func RevertCommit(repoPath, stream, commitID string) (*types.Commit, error) {
	target, err := LoadCommit(repoPath, stream, commitID)
	if err != nil {
		return nil, fmt.Errorf("failed to load commit %s: %w", commitID, err)
	}

	// Generate inverse operations
	inverted, err := invertOps(target.Operations)
	if err != nil {
		return nil, fmt.Errorf("failed to invert operations: %w", err)
	}

	// Create revert commit
	revert := &types.Commit{
		ID:          uuid.New().String(),
		Stream:      stream,
		Message:     fmt.Sprintf("Revert commit %s", commitID),
		AuthorName:  target.AuthorName,
		AuthorEmail: target.AuthorEmail,
		Timestamp:   time.Now().UTC(),
		Operations:  inverted,
	}

	// Save revert commit
	if err := SaveCommit(repoPath, revert); err != nil {
		return nil, fmt.Errorf("failed to save revert commit: %w", err)
	}

	return revert, nil
}

// invertOps generates inverse operations for a commit
func invertOps(ops []types.ExtendedOp) ([]types.ExtendedOp, error) {
	var inverted []types.ExtendedOp

	// Process operations in reverse order
	for i := len(ops) - 1; i >= 0; i-- {
		op := ops[i]
		switch op.Op.Type {
		case crdt.OpInsert:
			// Invert insert -> delete
			inverted = append(inverted, types.ExtendedOp{
				Op: crdt.Operation{
					Type:      crdt.OpDelete,
					LineID:    op.Op.LineID,
					Timestamp: time.Now(),
				},
			})

		case crdt.OpDelete:
			// Invert delete -> insert with original content
			if op.Op.Content == "" {
				return nil, fmt.Errorf("cannot revert delete operation: missing original content")
			}
			inverted = append(inverted, types.ExtendedOp{
				Op: crdt.Operation{
					Type:      crdt.OpInsert,
					LineID:    op.Op.LineID,
					Content:   op.Op.Content,
					Timestamp: time.Now(),
				},
			})

		case crdt.OpUpdate:
			// Invert update -> update with old content
			if op.OldContent == "" {
				return nil, fmt.Errorf("cannot revert update operation: missing old content")
			}
			inverted = append(inverted, types.ExtendedOp{
				Op: crdt.Operation{
					Type:      crdt.OpUpdate,
					LineID:    op.Op.LineID,
					Content:   op.OldContent,
					Timestamp: time.Now(),
				},
				OldContent: op.Op.Content,
			})
		}
	}

	return inverted, nil
}

func newLamport() uint64 {
	return uint64(time.Now().UnixNano())
}

func applyOps(repoPath, stream string, eops []ExtendedOp) error {
	// for each extended op, append to .evo/ops/<stream>/<fileID>.bin
	opsRoot := filepath.Join(repoPath, ".evo", "ops", stream)
	if err := os.MkdirAll(opsRoot, 0755); err != nil {
		return err
	}
	for _, eop := range eops {
		fid := eop.Op.FileID.String()
		binFile := filepath.Join(opsRoot, fid+".bin")
		if err := ops.AppendOp(binFile, eop.Op); err != nil {
			return err
		}
	}
	return nil
}

// For signing
func CommitHashString(c *types.Commit) string {
	// stable representation => ID + stream + message + etc
	h := sha256.New()
	h.Write([]byte(c.ID))
	h.Write([]byte(c.Stream))
	h.Write([]byte(c.Message))
	h.Write([]byte(c.AuthorName))
	h.Write([]byte(c.AuthorEmail))
	h.Write([]byte(c.Timestamp.String()))
	for _, eop := range c.Operations {
		// incorporate lamport, node, lineID, content, oldContent
		h.Write([]byte(fmt.Sprintf("%d_%s_%s_%s_old=%s",
			eop.Op.Lamport, eop.Op.NodeID, eop.Op.LineID, eop.Op.Content, eop.OldContent)))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

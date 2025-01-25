package streams

import (
	"encoding/binary"
	"encoding/json"
	"evo/internal/commits"
	"evo/internal/ops"
	"evo/internal/repo"
	"evo/internal/types"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/uuid"
)

func CreateStream(repoPath, name string) error {
	sdir := filepath.Join(repoPath, repo.EvoDir, "streams")
	if err := os.MkdirAll(sdir, 0755); err != nil {
		return err
	}
	fpath := filepath.Join(sdir, name)
	if _, err := os.Stat(fpath); err == nil {
		return fmt.Errorf("stream '%s' already exists", name)
	}
	return os.WriteFile(fpath, []byte{}, 0644)
}

func SwitchStream(repoPath, name string) error {
	fpath := filepath.Join(repoPath, repo.EvoDir, "streams", name)
	if _, err := os.Stat(fpath); os.IsNotExist(err) {
		return fmt.Errorf("stream '%s' does not exist", name)
	}
	head := filepath.Join(repoPath, repo.EvoDir, "HEAD")
	return os.WriteFile(head, []byte(name), 0644)
}

func ListStreams(repoPath string) ([]string, error) {
	dir := filepath.Join(repoPath, repo.EvoDir, "streams")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

func CurrentStream(repoPath string) (string, error) {
	head := filepath.Join(repoPath, repo.EvoDir, "HEAD")
	b, err := os.ReadFile(head)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// MergeStreams => merges all missing commits from source => target
func MergeStreams(repoPath, source, target string) error {
	srcCommits, err := ListCommits(repoPath, source)
	if err != nil {
		return err
	}
	tgtCommits, err := ListCommits(repoPath, target)
	if err != nil {
		return err
	}
	tgtMap := make(map[string]bool)
	for _, c := range tgtCommits {
		tgtMap[c.ID] = true
	}
	var missing []types.Commit
	for _, sc := range srcCommits {
		if !tgtMap[sc.ID] {
			missing = append(missing, sc)
		}
	}
	for _, mc := range missing {
		// replicate each op into .evo/ops/<target>/<fileID>.bin
		if err := replicateOps(repoPath, target, mc.Operations); err != nil {
			return err
		}
		// store a commit copy in target
		c2 := mc
		c2.Stream = target
		if err := commits.SaveCommitFile(filepath.Join(repoPath, repo.EvoDir, "commits", target), &c2); err != nil {
			return err
		}
	}
	return nil
}

func replicateOps(repoPath, stream string, eops []commits.ExtendedOp) error {
	for _, eop := range eops {
		fileID := eop.Op.FileID.String()
		binPath := filepath.Join(repoPath, repo.EvoDir, "ops", stream, fileID+".bin")
		if err := ops.AppendOp(binPath, eop.Op); err != nil {
			return err
		}
	}
	return nil
}

// CherryPick => replicate a single commit into the target
func CherryPick(repoPath, commitID, target string) error {
	allStreams, err := ListStreams(repoPath)
	if err != nil {
		return err
	}
	var found *types.Commit
OUTER:
	for _, s := range allStreams {
		cc, _ := ListCommits(repoPath, s)
		for _, c := range cc {
			if c.ID == commitID {
				found = &c
				break OUTER
			}
		}
	}
	if found == nil {
		return fmt.Errorf("commit %s not found in any stream", commitID)
	}
	// replicate ops
	if err := replicateOps(repoPath, target, found.Operations); err != nil {
		return err
	}
	// store new commit with new ID
	newID := uuid.New().String()
	nc := *found
	nc.ID = newID
	nc.Stream = target
	nc.Message = "[cherry-pick] " + found.Message
	return commits.SaveCommitFile(filepath.Join(repoPath, repo.EvoDir, "commits", target), &nc)
}

func ListCommits(repoPath, stream string) ([]types.Commit, error) {
	dir := filepath.Join(repoPath, repo.EvoDir, "commits", stream)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []types.Commit{}, nil
	}
	if err != nil {
		return nil, err
	}
	var out []types.Commit
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".bin" {
			c, err := loadCommit(filepath.Join(dir, e.Name()))
			if err != nil {
				return nil, err
			}
			out = append(out, *c)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Timestamp.Before(out[j].Timestamp)
	})
	return out, nil
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
	size := binary.BigEndian.Uint32(szBuf)
	data := make([]byte, size)
	if _, err := f.Read(data); err != nil {
		return nil, err
	}
	var c types.Commit
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func getCommit(repoPath, stream, commitID string) (*types.Commit, error) {
	cc, err := ListCommits(repoPath, stream)
	if err != nil {
		return nil, err
	}
	for _, c := range cc {
		if c.ID == commitID {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("commit %s not found in stream %s", commitID, stream)
}

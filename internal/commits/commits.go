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

// CreateCommit => gather new ops, sign if requested
func CreateCommit(repoPath, stream, msg, name, email string, sign bool) (*types.Commit, error) {
	cid := uuid.New().String()
	c := &types.Commit{
		ID:          cid,
		Stream:      stream,
		Message:     msg,
		AuthorName:  name,
		AuthorEmail: email,
		Timestamp:   time.Now(),
	}
	newOps, err := gatherNewOps(repoPath, stream)
	if err != nil {
		return nil, err
	}
	c.Ops = newOps

	if sign {
		sig, err := signing.SignCommit(c)
		if err != nil {
			return nil, err
		}
		c.Signature = sig
	}
	if err := saveCommit(repoPath, c); err != nil {
		return nil, err
	}
	return c, nil
}

// gatherNewOps => find ops not in prior commits, augment 'update' ops with oldContent
func gatherNewOps(repoPath, stream string) ([]ExtendedOp, error) {
	all, err := ListCommits(repoPath, stream)
	if err != nil {
		return nil, err
	}
	known := make(map[string]bool)
	for _, cc := range all {
		for _, eop := range cc.Ops {
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

// ListCommits => load from .evo/commits/<stream>/
func ListCommits(repoPath, stream string) ([]types.Commit, error) {
	dir := filepath.Join(repoPath, ".evo", "commits", stream)
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
			c, er := loadCommit(filepath.Join(dir, e.Name()))
			if er == nil {
				out = append(out, *c)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Timestamp.Before(out[j].Timestamp)
	})
	return out, nil
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

// RevertCommit => invert each op, store in a new commit
func RevertCommit(repoPath, stream, commitID string) (string, error) {
	all, err := ListCommits(repoPath, stream)
	if err != nil {
		return "", err
	}
	var target *types.Commit
	for _, c := range all {
		if c.ID == commitID {
			target = &c
			break
		}
	}
	if target == nil {
		return "", fmt.Errorf("commit %s not found in stream %s", commitID, stream)
	}
	newID := uuid.New().String()
	now := time.Now()
	invOps := invertOps(target.Ops)
	rcommit := &types.Commit{
		ID:          newID,
		Stream:      stream,
		Message:     "Revert of " + commitID + ": " + target.Message,
		AuthorName:  "Reverter",
		AuthorEmail: "revert@evo",
		Timestamp:   now,
		Ops:         invOps,
	}
	// apply these ops to the .evo/ops
	if err := applyOps(repoPath, stream, invOps); err != nil {
		return "", err
	}
	if err := saveCommit(repoPath, rcommit); err != nil {
		return "", err
	}
	return newID, nil
}

func invertOps(eops []ExtendedOp) []ExtendedOp {
	var out []ExtendedOp
	for _, eop := range eops {
		op := eop.Op
		switch op.Type {
		case crdt.OpInsert:
			// revert => delete
			out = append(out, ExtendedOp{
				Op: crdt.Operation{
					FileID:  op.FileID,
					Type:    crdt.OpDelete,
					Lamport: newLamport(),
					NodeID:  uuid.New(),
					LineID:  op.LineID,
				},
			})
		case crdt.OpDelete:
			// revert => insert with old content if we had it.
			// But in a real system we'd store the old content of the line.
			// We'll skip if we don't have it.
		case crdt.OpUpdate:
			// revert => update with OldContent
			out = append(out, ExtendedOp{
				Op: crdt.Operation{
					FileID:  op.FileID,
					Type:    crdt.OpUpdate,
					Lamport: newLamport(),
					NodeID:  uuid.New(),
					LineID:  op.LineID,
					Content: eop.OldContent,
				},
				OldContent: op.Content,
			})
		}
	}
	return out
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
	for _, eop := range c.Ops {
		// incorporate lamport, node, lineID, content, oldContent
		h.Write([]byte(fmt.Sprintf("%d_%s_%s_%s_old=%s",
			eop.Op.Lamport, eop.Op.NodeID, eop.Op.LineID, eop.Op.Content, eop.OldContent)))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

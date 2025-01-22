package ops

import (
	"evo/internal/crdt"
	"evo/internal/index"
	"evo/internal/lfs"
	"evo/internal/util"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// IngestLocalChanges checks each file in the working directory, handles large-file threshold, stable fileID, then line CRDT logic.
func IngestLocalChanges(repoPath, stream string) ([]string, error) {
	files, err := util.ListAllFiles(repoPath)
	if err != nil {
		return nil, err
	}
	var changed []string
	var mu sync.Mutex
	var wg sync.WaitGroup
	chWork := make(chan string, len(files))
	chErr := make(chan error, 8)

	for _, f := range files {
		chWork <- f
	}
	close(chWork)

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for rel := range chWork {
				if strings.HasPrefix(rel, ".evo") {
					continue
				}
				abs := filepath.Join(repoPath, rel)
				fi, errStat := os.Stat(abs)
				if errStat != nil || fi.IsDir() {
					continue
				}
				ok, e2 := processFile(repoPath, stream, rel, abs, fi.Size())
				if e2 != nil {
					chErr <- e2
					return
				}
				if ok {
					mu.Lock()
					changed = append(changed, rel)
					mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()
	close(chErr)
	for e := range chErr {
		if e != nil {
			return nil, e
		}
	}
	return changed, nil
}

func processFile(repoPath, stream, relPath, absPath string, fsize int64) (bool, error) {
	fileID, err := index.LookupFileID(repoPath, relPath)
	if err != nil {
		// not tracked => skip
		return false, nil
	}
	opsFile := filepath.Join(repoPath, ".evo", "ops", stream, fileID+".bin")
	existing, _ := LoadAllOps(opsFile)

	// build doc
	doc := crdt.NewRGA()
	for _, op := range existing {
		if err := doc.Apply(op); err != nil {
			return false, fmt.Errorf("applying operation: %v", err)
		}
	}

	threshold := readLargeThreshold(repoPath)
	if fsize > threshold {
		// large file => store stub
		return storeLargeFile(repoPath, stream, fileID, relPath, absPath, doc, opsFile)
	}

	// normal text => read lines
	data, err := os.ReadFile(absPath)
	if err != nil {
		return false, err
	}
	diskLines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	docLines := doc.Materialize()
	if eqLines(docLines, diskLines) {
		return false, nil
	}
	changed := false
	var lamport uint64 = uint64(time.Now().UnixNano())
	nodeID := uuid.New()

	lineIDs := doc.GetLineIDs()
	prefix := 0
	minLen := len(docLines)
	if len(diskLines) < minLen {
		minLen = len(diskLines)
	}
	for prefix < minLen && docLines[prefix] == diskLines[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < minLen-prefix && docLines[len(docLines)-1-suffix] == diskLines[len(diskLines)-1-suffix] {
		suffix++
	}
	docMid := docLines[prefix : len(docLines)-suffix]
	diskMid := diskLines[prefix : len(diskLines)-suffix]

	startPos := prefix
	var i int
	for i = 0; i < len(docMid) && i < len(diskMid); i++ {
		if docMid[i] != diskMid[i] {
			op := crdt.Operation{
				Type:      crdt.OpUpdate,
				Lamport:   lamport + uint64(i),
				NodeID:    nodeID,
				FileID:    parseUUID(fileID),
				LineID:    lineIDs[startPos+i],
				Content:   diskMid[i],
				Stream:    stream,
				Timestamp: time.Now(),
			}
			if err := AppendOp(opsFile, op); err != nil {
				return false, err
			}
			changed = true
		}
	}
	for j := len(diskMid); j < len(docMid); j++ {
		op := crdt.Operation{
			Type:      crdt.OpDelete,
			Lamport:   lamport + uint64(j),
			NodeID:    nodeID,
			FileID:    parseUUID(fileID),
			LineID:    lineIDs[startPos+j],
			Stream:    stream,
			Timestamp: time.Now(),
		}
		if err := AppendOp(opsFile, op); err != nil {
			return false, err
		}
		changed = true
	}
	if i < len(diskMid) {
		// disk has extra => insert
		for j := i; j < len(diskMid); j++ {
			insOp := crdt.Operation{
				FileID:  parseUUID(fileID),
				Type:    crdt.OpInsert,
				Lamport: lamport + uint64(j),
				NodeID:  uuid.New(),
				LineID:  uuid.New(),
				Content: diskMid[j],
			}
			AppendOp(opsFile, insOp)
			lamport++
			changed = true
		}
	}
	return changed, nil
}

func storeLargeFile(repoPath, stream, fileID, relPath, absPath string, doc *crdt.RGA, opsFile string) (bool, error) {
	// Initialize LFS store
	store := lfs.NewStore(repoPath)

	// Open file
	f, err := os.Open(absPath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	// Get file info
	stat, err := f.Stat()
	if err != nil {
		return false, err
	}

	// Store in LFS
	info, err := store.StoreFile(fileID, f, stat.Size())
	if err != nil {
		return false, err
	}

	// Add LFS stub line
	docLines := doc.Materialize()
	if len(docLines) == 1 && strings.HasPrefix(docLines[0], "EVO-LFS:") {
		// already a stub
		return false, nil
	}

	// Replace content with LFS stub
	lop := crdt.Operation{
		FileID:  parseUUID(fileID),
		Type:    crdt.OpInsert,
		Lamport: uint64(time.Now().UnixNano()),
		NodeID:  uuid.New(),
		LineID:  uuid.New(),
		Content: fmt.Sprintf("EVO-LFS:%s:%d", fileID, info.Size),
	}
	if err := AppendOp(opsFile, lop); err != nil {
		return false, err
	}

	return true, nil
}

func copyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()
	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()
	buf := make([]byte, 64*1024)
	for {
		n, e := s.Read(buf)
		if n > 0 {
			d.Write(buf[:n])
		}
		if e != nil {
			break
		}
	}
	return nil
}

func readLargeThreshold(repoPath string) int64 {
	// read config: files.largeThreshold
	// fallback 1MB
	return 1_000_000
}

func parseUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}

func eqLines(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

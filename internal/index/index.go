package index

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// The .evo/index is lines: "<fileID> <path>"

func LoadIndex(repoPath string) (map[string]string, map[string]string, error) {
	// path->fileID, fileID->path
	path2id := make(map[string]string)
	id2path := make(map[string]string)
	idxPath := filepath.Join(repoPath, ".evo", "index")
	f, err := os.Open(idxPath)
	if os.IsNotExist(err) {
		return path2id, id2path, nil
	}
	if err != nil {
		return path2id, id2path, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			fid := parts[0]
			p := parts[1]
			path2id[p] = fid
			id2path[fid] = p
		}
	}
	return path2id, id2path, nil
}

func SaveIndex(repoPath string, path2id map[string]string) error {
	idxPath := filepath.Join(repoPath, ".evo", "index")
	f, err := os.OpenFile(idxPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	for p, fid := range path2id {
		fmt.Fprintf(f, "%s %s\n", fid, p)
	}
	return nil
}

// UpdateIndex => scans working dir, assigns stable fileIDs, removes missing files
func UpdateIndex(repoPath string) error {
	p2id, id2p, err := LoadIndex(repoPath)
	if err != nil {
		return err
	}
	var working []string
	filepath.Walk(repoPath, func(path string, info os.FileInfo, e error) error {
		if e != nil {
			return nil
		}
		if !info.IsDir() {
			rel, _ := filepath.Rel(repoPath, path)
			if !strings.HasPrefix(rel, ".evo") {
				working = append(working, rel)
			}
		}
		return nil
	})
	// detect new files
	for _, w := range working {
		if _, ok := p2id[w]; !ok {
			// assign new fileID
			fid := uuid.New().String()
			p2id[w] = fid
			id2p[fid] = w
		}
	}
	// detect removed
	for p, fid := range p2id {
		found := false
		for _, w := range working {
			if w == p {
				found = true
				break
			}
		}
		if !found {
			delete(p2id, p)
			delete(id2p, fid)
		}
	}
	return SaveIndex(repoPath, p2id)
}

// LookupFileID => returns stable fileID for a given path
func LookupFileID(repoPath, relPath string) (string, error) {
	p2id, _, err := LoadIndex(repoPath)
	if err != nil {
		return "", err
	}
	fid, ok := p2id[relPath]
	if !ok {
		return "", errors.New("file not tracked in index: " + relPath)
	}
	return fid, nil
}

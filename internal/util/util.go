package util

import (
	"os"
	"path/filepath"
)

func ListAllFiles(repoPath string) ([]string, error) {
	var out []string
	filepath.Walk(repoPath, func(path string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		if !info.IsDir() {
			rel, _ := filepath.Rel(repoPath, path)
			out = append(out, rel)
		}
		return nil
	})
	return out, nil
}

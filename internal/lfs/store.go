package lfs

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Store manages large file storage with deduplication
type Store struct {
	mu   sync.RWMutex
	root string
}

// NewStore creates a new LFS store at the given root path
func NewStore(root string) *Store {
	// Create necessary directories
	os.MkdirAll(filepath.Join(root, ".evo", "lfs"), 0755)
	os.MkdirAll(filepath.Join(root, ".evo", "chunks"), 0755)

	return &Store{
		root: root,
	}
}

// StoreFile stores a file in chunks and returns file info
func (s *Store) StoreFile(id string, r io.Reader, size int64) (*FileInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create file directory
	fileDir := filepath.Join(s.root, ".evo", "lfs", id)
	if err := os.MkdirAll(fileDir, 0755); err != nil {
		return nil, err
	}

	// Calculate content hash and split into chunks
	chunks := make([]ChunkInfo, 0)
	contentHash := NewHash()

	// Read file in chunks to calculate hash and store chunks
	var totalSize int64
	buf := make([]byte, ChunkSize)
	for totalSize < size {
		// Calculate remaining size and read size
		remaining := size - totalSize
		readSize := ChunkSize
		if remaining < ChunkSize {
			readSize = int(remaining)
		}

		// Read chunk
		n, err := io.ReadFull(r, buf[:readSize])
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, err
		}
		if n == 0 {
			break
		}

		// Calculate content hash for this chunk
		contentHash.Write(buf[:n])

		// Calculate chunk hash and store chunk
		chunk := make([]byte, n)
		copy(chunk, buf[:n])
		chunkHash := HashBytes(chunk)

		// Store chunk if it doesn't exist
		chunkPath := filepath.Join(s.root, ".evo", "chunks", chunkHash)
		if _, err := os.Stat(chunkPath); os.IsNotExist(err) {
			// Store new chunk
			chunkData := make([]byte, n)
			copy(chunkData, chunk)
			if err := os.WriteFile(chunkPath, chunkData, 0644); err != nil {
				return nil, err
			}
		}

		chunks = append(chunks, ChunkInfo{
			Hash: chunkHash,
			Size: int64(n),
		})

		totalSize += int64(n)

		// Break if we've read all the data
		if totalSize >= size {
			break
		}
	}

	// Verify total size matches expected size
	if totalSize != size {
		return nil, fmt.Errorf("expected size %d, got %d", size, totalSize)
	}

	hashStr := contentHash.Sum()

	// Check for existing file with same content hash
	existingFiles, err := os.ReadDir(filepath.Join(s.root, ".evo", "lfs"))
	if err == nil {
		for _, f := range existingFiles {
			if !f.IsDir() {
				continue
			}
			existingInfo, err := s.loadFileInfo(f.Name())
			if err != nil {
				continue
			}
			if existingInfo.ContentHash == hashStr {
				// Found existing file with same content
				existingInfo.RefCount++
				if err := s.saveFileInfo(f.Name(), existingInfo); err != nil {
					return nil, err
				}

				// Create new file info pointing to same chunks
				newInfo := &FileInfo{
					ID:          id,
					Size:        existingInfo.Size,
					ContentHash: existingInfo.ContentHash,
					NumChunks:   existingInfo.NumChunks,
					Chunks:      existingInfo.Chunks,
					RefCount:    existingInfo.RefCount, // Use same ref count as existing file
					Created:     time.Now(),
				}
				if err := s.saveFileInfo(id, newInfo); err != nil {
					return nil, err
				}
				return newInfo, nil
			}
		}
	}

	// Create file info
	info := &FileInfo{
		ID:          id,
		Size:        size,
		ContentHash: hashStr,
		NumChunks:   len(chunks),
		Chunks:      chunks,
		RefCount:    1,
		Created:     time.Now(),
	}

	// Save file info
	if err := s.saveFileInfo(id, info); err != nil {
		return nil, err
	}

	return info, nil
}

func (s *Store) saveFileInfo(id string, info *FileInfo) error {
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.root, ".evo", "lfs", id, "info.json"), data, 0644)
}

func (s *Store) loadFileInfo(id string) (*FileInfo, error) {
	data, err := os.ReadFile(filepath.Join(s.root, ".evo", "lfs", id, "info.json"))
	if err != nil {
		return nil, err
	}
	var info FileInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// ReadFile reads a file from chunks into the writer
func (s *Store) ReadFile(id string, w io.Writer) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Load file info
	info, err := s.loadFileInfo(id)
	if err != nil {
		return err
	}

	// Read chunks
	for _, chunk := range info.Chunks {
		data, err := os.ReadFile(filepath.Join(s.root, ".evo", "chunks", chunk.Hash))
		if err != nil {
			return err
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
	}

	return nil
}

// DeleteFile deletes a file and its chunks if no longer referenced
func (s *Store) DeleteFile(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load file info
	info, err := s.loadFileInfo(id)
	if err != nil {
		return err
	}

	// Delete file info
	fileDir := filepath.Join(s.root, ".evo", "lfs", id)
	if err := os.RemoveAll(fileDir); err != nil {
		return err
	}

	// Find other files with same content hash
	existingFiles, err := os.ReadDir(filepath.Join(s.root, ".evo", "lfs"))
	if err == nil {
		for _, f := range existingFiles {
			if !f.IsDir() || f.Name() == id {
				continue
			}
			existingInfo, err := s.loadFileInfo(f.Name())
			if err != nil {
				continue
			}
			if existingInfo.ContentHash == info.ContentHash {
				// Found another file with same content, decrement its ref count
				existingInfo.RefCount--
				if err := s.saveFileInfo(f.Name(), existingInfo); err != nil {
					return err
				}
				break
			}
		}
	}

	// Delete unreferenced chunks
	for _, chunk := range info.Chunks {
		chunkPath := filepath.Join(s.root, ".evo", "chunks", chunk.Hash)
		if s.isChunkReferenced(chunk.Hash) {
			continue
		}
		if err := os.Remove(chunkPath); err != nil {
			return err
		}
	}

	return nil
}

// isChunkReferenced checks if a chunk is referenced by any file
func (s *Store) isChunkReferenced(hash string) bool {
	files, err := os.ReadDir(filepath.Join(s.root, ".evo", "lfs"))
	if err != nil {
		return false
	}

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		info, err := s.loadFileInfo(file.Name())
		if err != nil {
			continue
		}

		for _, chunk := range info.Chunks {
			if chunk.Hash == hash {
				return true
			}
		}
	}

	return false
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

package lfs

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestStore(t *testing.T) {
	// Create temp dir for testing
	tmpDir, err := os.MkdirTemp("", "evo-lfs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	testData := []byte("Hello, this is test data for LFS!")
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize store
	store := NewStore(tmpDir)

	t.Run("Store and Read File", func(t *testing.T) {
		// Store file
		f, err := os.Open(testFile)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		info, err := store.StoreFile("test123", f, int64(len(testData)))
		if err != nil {
			t.Fatal(err)
		}

		// Verify file info
		if info.Size != int64(len(testData)) {
			t.Errorf("Expected size %d, got %d", len(testData), info.Size)
		}
		if info.RefCount != 1 {
			t.Errorf("Expected refCount 1, got %d", info.RefCount)
		}

		// Read file back
		var buf bytes.Buffer
		if err := store.ReadFile("test123", &buf); err != nil {
			t.Fatal(err)
		}

		// Verify content
		if !bytes.Equal(buf.Bytes(), testData) {
			t.Error("Read data doesn't match original")
		}
	})

	t.Run("Deduplication", func(t *testing.T) {
		// Store same file again
		f, err := os.Open(testFile)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		info, err := store.StoreFile("test456", f, int64(len(testData)))
		if err != nil {
			t.Fatal(err)
		}

		// Verify increased ref count
		if info.RefCount != 2 {
			t.Errorf("Expected refCount 2, got %d", info.RefCount)
		}

		// Check chunks directory
		chunksDir := filepath.Join(tmpDir, ".evo", "chunks")
		entries, err := os.ReadDir(chunksDir)
		if err != nil {
			t.Fatal(err)
		}

		// Should only have one chunk since content is identical
		if len(entries) != 1 {
			t.Errorf("Expected 1 chunk, got %d", len(entries))
		}
	})

	t.Run("Reference Counting", func(t *testing.T) {
		// Delete first file
		if err := store.DeleteFile("test123"); err != nil {
			t.Fatal(err)
		}

		// Verify file still exists
		info, err := store.loadFileInfo("test456")
		if err != nil {
			t.Fatal(err)
		}
		if info.RefCount != 1 {
			t.Errorf("Expected refCount 1, got %d", info.RefCount)
		}

		// Delete last reference
		if err := store.DeleteFile("test456"); err != nil {
			t.Fatal(err)
		}

		// Verify file is gone
		if _, err := store.loadFileInfo("test456"); err == nil {
			t.Error("File should not exist")
		}
	})
}

func TestLargeFileChunking(t *testing.T) {
	// Create temp dir
	tmpDir, err := os.MkdirTemp("", "evo-lfs-chunks-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create large test data (5MB)
	size := 5 * 1024 * 1024
	data := make([]byte, size)

	// Ensure each 1MB chunk is unique:
	for i := 0; i < size; i++ {
		chunkIndex := i >> 20 // i / 1 MB
		data[i] = byte(chunkIndex)
	}

	// Write to testFile, then store in LFS, expecting 5 distinct chunks
	testFile := filepath.Join(tmpDir, "large.bin")
	if err := os.WriteFile(testFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	store := NewStore(tmpDir)

	t.Run("Chunk Storage", func(t *testing.T) {
		// Store large file
		f, err := os.Open(testFile)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		info, err := store.StoreFile("large123", f, int64(size))
		if err != nil {
			t.Fatal(err)
		}

		// Verify number of chunks
		expectedChunks := (size + ChunkSize - 1) / ChunkSize
		if info.NumChunks != expectedChunks {
			t.Errorf("Expected %d chunks, got %d", expectedChunks, info.NumChunks)
		}

		// Check chunks directory
		chunksDir := filepath.Join(tmpDir, ".evo", "chunks")
		entries, err := os.ReadDir(chunksDir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != expectedChunks {
			t.Errorf("Expected %d chunk files, got %d", expectedChunks, len(entries))
		}
	})

	t.Run("Streaming Read", func(t *testing.T) {
		// Read file back in chunks
		var buf bytes.Buffer
		if err := store.ReadFile("large123", &buf); err != nil {
			t.Fatal(err)
		}

		// Verify content
		if !bytes.Equal(buf.Bytes(), data) {
			t.Error("Read data doesn't match original")
		}
	})
}

func TestGarbageCollection(t *testing.T) {
	// Create temp dir
	tmpDir, err := os.MkdirTemp("", "evo-lfs-gc-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewStore(tmpDir)
	gc := NewGarbageCollector(store)

	// Create test files
	files := []struct {
		name    string
		content []byte
	}{
		{"file1", []byte("content1")},
		{"file2", []byte("content2")},
		{"file3", []byte("content3")},
	}

	for _, f := range files {
		r := bytes.NewReader(f.content)
		if _, err := store.StoreFile(f.name, r, int64(len(f.content))); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("GC Cleanup", func(t *testing.T) {
		// Delete some files
		if err := store.DeleteFile("file1"); err != nil {
			t.Fatal(err)
		}
		if err := store.DeleteFile("file2"); err != nil {
			t.Fatal(err)
		}

		// Run GC
		if err := gc.Run(); err != nil {
			t.Fatal(err)
		}

		// Verify only file3's chunks remain
		chunksDir := filepath.Join(tmpDir, ".evo", "chunks")
		entries, err := os.ReadDir(chunksDir)
		if err != nil {
			t.Fatal(err)
		}

		expectedChunks := 1 // Only file3's chunk should remain
		if len(entries) != expectedChunks {
			t.Errorf("Expected %d chunks after GC, got %d", expectedChunks, len(entries))
		}
	})
}

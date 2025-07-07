package core

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestFileExists(t *testing.T) {
	// Test with a file that should exist
	tmpDir := createTestDir(t)
	testFile := writeTestFile(t, tmpDir, "test.txt", []byte("test"))

	if !FileExists(testFile) {
		t.Error("FileExists should return true for existing file")
	}

	if FileExists(tmpDir + "/nonexistent.txt") {
		t.Error("FileExists should return false for non-existent file")
	}
}

func TestReadWriteFile(t *testing.T) {
	tmpDir := createTestDir(t)
	testContent := []byte("test content")
	testFile := tmpDir + "/test.txt"

	// Test write
	err := WriteFile(testFile, testContent)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Test read
	readContent, err := ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if !bytes.Equal(testContent, readContent) {
		t.Errorf("Content mismatch: expected %q, got %q", testContent, readContent)
	}
}

func TestWipeData(t *testing.T) {
	data := []byte("sensitive data")
	WipeData(data)

	for i, b := range data {
		if b != 0 {
			t.Errorf("Byte at index %d not wiped: got %d", i, b)
		}
	}
}

// createTestDir creates a temporary directory for testing
func createTestDir(t *testing.T) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "kiln-core-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	return tmpDir
}

// writeTestFile writes content to a file in the given directory
func writeTestFile(t *testing.T, dir, name string, content []byte) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("Failed to write test file %s: %v", path, err)
	}

	return path
}

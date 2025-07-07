package core

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestFileExists(t *testing.T) {
	tmpDir := createTestDir(t)
	testFile := writeTestFile(t, tmpDir, "test.txt", []byte("test"))

	if !FileExists(testFile) {
		t.Error("FileExists should return true for existing file")
	}

	if FileExists(filepath.Join(tmpDir, "nonexistent.txt")) {
		t.Error("FileExists should return false for non-existent file")
	}
}

func TestReadWriteFile(t *testing.T) {
	tmpDir := createTestDir(t)
	testContent := []byte("test content")
	testFile := filepath.Join(tmpDir, "test.txt")

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

	// Verify file permissions
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	if info.Mode().Perm() != 0o600 {
		t.Errorf("File permissions: expected 0600, got %v", info.Mode().Perm())
	}
}

func TestWipeData(t *testing.T) {
	data := []byte("sensitive data")
	originalData := make([]byte, len(data))
	copy(originalData, data)

	WipeData(data)

	for i, b := range data {
		if b != 0 {
			t.Errorf("Byte at index %d not wiped: got %d", i, b)
		}
	}

	// Ensure original data was actually different
	allZero := true

	for _, b := range originalData {
		if b != 0 {
			allZero = false

			break
		}
	}

	if allZero {
		t.Error("Test setup error: original data was all zeros")
	}
}

func TestSortedKeys(t *testing.T) {
	vars := map[string][]byte{
		"ZEBRA": []byte("z"),
		"ALPHA": []byte("a"),
		"BETA":  []byte("b"),
	}

	keys := SortedKeys(vars)
	expected := []string{"ALPHA", "BETA", "ZEBRA"}

	if len(keys) != len(expected) {
		t.Errorf("Length mismatch: expected %d, got %d", len(expected), len(keys))
	}

	for i, key := range keys {
		if i >= len(expected) || key != expected[i] {
			t.Errorf("Key at position %d: expected %q, got %q", i, expected[i], key)
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
	if err := WriteFile(path, content); err != nil {
		t.Fatalf("Failed to write test file %s: %v", path, err)
	}

	return path
}

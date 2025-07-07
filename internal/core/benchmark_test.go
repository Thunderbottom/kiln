package core

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"filippo.io/age"

	"github.com/thunderbottom/kiln/internal/config"
)

func BenchmarkSetGetWorkflow(b *testing.B) {
	tmpDir := setupBenchDir(b)
	identity, cfg := setupBenchConfig(b, tmpDir)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := "BENCH_VAR"
		value := []byte("benchmark_value")

		// Set variable
		if err := SetEnvVar(identity, cfg, "default", key, value); err != nil {
			b.Fatalf("SetEnvVar failed: %v", err)
		}

		// Get variable
		retrieved, cleanup, err := GetEnvVar(identity, cfg, "default", key)
		if err != nil {
			b.Fatalf("GetEnvVar failed: %v", err)
		}

		cleanup()

		_ = retrieved
	}
}

func BenchmarkLargeVariableSet(b *testing.B) {
	tmpDir := setupBenchDir(b)
	identity, cfg := setupBenchConfig(b, tmpDir)

	largeVars := make(map[string][]byte)

	for i := range 50 {
		key := fmt.Sprintf("VAR_%03d", i)
		value := fmt.Sprintf("value-for-variable-%03d-with-content", i)
		largeVars[key] = []byte(value)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if err := SaveAllEnvVars(identity, cfg, "default", largeVars); err != nil {
			b.Fatalf("SaveAllEnvVars failed: %v", err)
		}

		retrieved, cleanup, err := GetAllEnvVars(identity, cfg, "default")
		if err != nil {
			b.Fatalf("GetAllEnvVars failed: %v", err)
		}

		cleanup()

		_ = retrieved
	}
}

func BenchmarkMemoryUsage(b *testing.B) {
	tmpDir := setupBenchDir(b)
	identity, cfg := setupBenchConfig(b, tmpDir)

	largeValue := make([]byte, 1024)
	for i := range largeValue {
		largeValue[i] = byte(i % 256)
	}

	b.ResetTimer()
	b.ReportAllocs()

	var startMem, endMem runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&startMem)

	for i := 0; i < b.N; i++ {
		if err := SetEnvVar(identity, cfg, "default", "LARGE_VAR", largeValue); err != nil {
			b.Fatalf("SetEnvVar failed: %v", err)
		}

		retrieved, cleanup, err := GetEnvVar(identity, cfg, "default", "LARGE_VAR")
		if err != nil {
			b.Fatalf("GetEnvVar failed: %v", err)
		}

		cleanup()

		_ = retrieved
	}

	runtime.GC()
	runtime.ReadMemStats(&endMem)

	memPerOp := float64(endMem.TotalAlloc-startMem.TotalAlloc) / float64(b.N)
	b.ReportMetric(memPerOp, "bytes/op-actual")
}

func BenchmarkCryptoOperations(b *testing.B) {
	tmpDir := setupBenchDir(b)

	privateKey, publicKey, err := GenerateKeyPair()
	if err != nil {
		b.Fatalf("GenerateKeyPair failed: %v", err)
	}
	defer WipeData(privateKey)

	keyPath := filepath.Join(tmpDir, "bench.key")
	if err := SaveKeys(privateKey, publicKey, keyPath); err != nil {
		b.Fatalf("SaveKeys failed: %v", err)
	}

	identity, err := NewIdentityFromKey(keyPath)
	if err != nil {
		b.Fatalf("NewIdentityFromKey failed: %v", err)
	}

	recipients, err := ParseRecipients([]string{publicKey})
	if err != nil {
		b.Fatalf("ParseRecipients failed: %v", err)
	}

	manager := NewAgeManager(recipients, []age.Identity{identity.AgeIdentity()})
	testData := []byte("test data for encryption benchmark")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		encrypted, err := manager.Encrypt(testData)
		if err != nil {
			b.Fatalf("Encrypt failed: %v", err)
		}

		decrypted, err := manager.Decrypt(encrypted)
		if err != nil {
			b.Fatalf("Decrypt failed: %v", err)
		}

		WipeData(decrypted)

		_ = encrypted
	}
}

func BenchmarkEnvParsing(b *testing.B) {
	vars := make(map[string][]byte)

	for i := range 100 {
		key := fmt.Sprintf("VAR_%03d", i)
		value := fmt.Sprintf("value-for-variable-%03d", i)
		vars[key] = []byte(value)
	}

	envContent := FormatEnv(vars)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		parsed, err := ParseEnv(envContent)
		if err != nil {
			b.Fatalf("ParseEnv failed: %v", err)
		}

		_ = parsed
	}
}

func setupBenchDir(b *testing.B) string {
	b.Helper()

	tmpDir, err := os.MkdirTemp("", "kiln-bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}

	b.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	return tmpDir
}

func setupBenchConfig(b *testing.B, tmpDir string) (*Identity, *config.Config) {
	b.Helper()

	privateKey, publicKey, err := GenerateKeyPair()
	if err != nil {
		b.Fatalf("GenerateKeyPair failed: %v", err)
	}
	defer WipeData(privateKey)

	keyPath := filepath.Join(tmpDir, "test.key")
	if err := SaveKeys(privateKey, publicKey, keyPath); err != nil {
		b.Fatalf("SaveKeys failed: %v", err)
	}

	identity, err := NewIdentityFromKey(keyPath)
	if err != nil {
		b.Fatalf("NewIdentityFromKey failed: %v", err)
	}

	cfg := config.NewConfig()
	cfg.AddRecipient("bench-user", publicKey)
	cfg.Files["default"] = config.FileConfig{
		Filename: filepath.Join(tmpDir, ".kiln.env"),
		Access:   []string{"*"},
	}

	return identity, cfg
}

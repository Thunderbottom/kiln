package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/core"
)

func TestApplyCmd_validate(t *testing.T) {
	tests := []struct {
		name    string
		cmd     ApplyCmd
		wantErr bool
	}{
		{
			name: "valid inputs",
			cmd: ApplyCmd{
				File:     "test",
				Template: "template.txt",
				Output:   "output.txt",
			},
			wantErr: false,
		},
		{
			name: "invalid file name",
			cmd: ApplyCmd{
				File:     "../test",
				Template: "template.txt",
			},
			wantErr: true,
		},
		{
			name: "empty template path",
			cmd: ApplyCmd{
				File:     "test",
				Template: "",
			},
			wantErr: true,
		},
		{
			name: "mismatched delimiters - left only",
			cmd: ApplyCmd{
				File:          "test",
				Template:      "template.txt",
				LeftDelimiter: "[[",
			},
			wantErr: true,
		},
		{
			name: "mismatched delimiters - right only",
			cmd: ApplyCmd{
				File:           "test",
				Template:       "template.txt",
				RightDelimiter: "]]",
			},
			wantErr: true,
		},
		{
			name: "valid custom delimiters",
			cmd: ApplyCmd{
				File:           "test",
				Template:       "template.txt",
				LeftDelimiter:  "[[",
				RightDelimiter: "]]",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cmd.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyCmd.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestApplyCmd_buildPatterns(t *testing.T) {
	tests := []struct {
		name     string
		cmd      ApplyCmd
		expected int
	}{
		{
			name:     "default delimiters",
			cmd:      ApplyCmd{},
			expected: 2, // ${VAR} and $VAR patterns
		},
		{
			name: "custom delimiters",
			cmd: ApplyCmd{
				LeftDelimiter:  "[[",
				RightDelimiter: "]]",
			},
			expected: 1, // [[VAR]] pattern only
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patterns := tt.cmd.buildPatterns()

			if len(patterns) != tt.expected {
				t.Errorf("ApplyCmd.buildPatterns() got %d patterns, want %d", len(patterns), tt.expected)
			}
		})
	}
}

func TestApplyCmd_substituteVariables(t *testing.T) {
	variables := map[string][]byte{
		"DATABASE_URL": []byte("postgres://localhost:5432/test"),
		"API_KEY":      []byte("secret-123"),
		"PORT":         []byte("8080"),
	}

	tests := []struct {
		name     string
		cmd      ApplyCmd
		template string
		want     string
		wantErr  bool
	}{
		{
			name:     "default braces substitution",
			cmd:      ApplyCmd{},
			template: "db=${DATABASE_URL} api=${API_KEY}",
			want:     "db=postgres://localhost:5432/test api=secret-123",
			wantErr:  false,
		},
		{
			name:     "default simple substitution",
			cmd:      ApplyCmd{},
			template: "port=$PORT",
			want:     "port=8080",
			wantErr:  false,
		},
		{
			name:     "mixed substitution",
			cmd:      ApplyCmd{},
			template: "${DATABASE_URL}:$PORT",
			want:     "postgres://localhost:5432/test:8080",
			wantErr:  false,
		},
		{
			name:     "custom delimiters",
			cmd:      ApplyCmd{LeftDelimiter: "[[", RightDelimiter: "]]"},
			template: "url=[[DATABASE_URL]] key=[[API_KEY]]",
			want:     "url=postgres://localhost:5432/test key=secret-123",
			wantErr:  false,
		},
		{
			name:     "custom delimiters with spacing",
			cmd:      ApplyCmd{LeftDelimiter: "[[", RightDelimiter: "]]"},
			template: "url=[[ DATABASE_URL ]] key=[[API_KEY]] port=[[ PORT]]",
			want:     "url=postgres://localhost:5432/test key=secret-123 port=8080",
			wantErr:  false,
		},
		{
			name:     "custom delimiters ignore default patterns",
			cmd:      ApplyCmd{LeftDelimiter: "[[", RightDelimiter: "]]"},
			template: "${DATABASE_URL} [[API_KEY]]",
			want:     "${DATABASE_URL} secret-123",
			wantErr:  false,
		},
		{
			name:     "missing variable non-strict",
			cmd:      ApplyCmd{},
			template: "${MISSING_VAR}",
			want:     "${MISSING_VAR}",
			wantErr:  false,
		},
		{
			name:     "missing variable strict",
			cmd:      ApplyCmd{Strict: true},
			template: "${MISSING_VAR}",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.cmd.substituteVariables([]byte(tt.template), variables)

			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyCmd.substituteVariables() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !tt.wantErr && string(got) != tt.want {
				t.Errorf("ApplyCmd.substituteVariables() = %v, want %v", string(got), tt.want)
			}
		})
	}
}

func TestApplyCmd_Run(t *testing.T) {
	tmpDir := createTempDir(t)
	configPath, keyPath := setupTestEnvironment(t, tmpDir)

	identity, err := core.NewIdentityFromKey(keyPath)
	if err != nil {
		t.Fatalf("NewIdentityFromKey failed: %v", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("config.Load failed: %v", err)
	}

	testVars := map[string][]byte{
		"DATABASE_URL": []byte("postgres://localhost:5432/test"),
		"API_KEY":      []byte("secret-123"),
	}

	err = core.SaveAllEnvVars(identity, cfg, "default", testVars)
	if err != nil {
		t.Fatalf("SaveAllEnvVars failed: %v", err)
	}

	templatePath := filepath.Join(tmpDir, "template.txt")
	templateContent := "database: ${DATABASE_URL}\napi_key: ${API_KEY}"

	err = os.WriteFile(templatePath, []byte(templateContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	outputPath := filepath.Join(tmpDir, "output.txt")

	cmd := &ApplyCmd{
		File:     "default",
		Template: templatePath,
		Output:   outputPath,
	}

	runtime, err := NewRuntime(configPath, keyPath, false)
	if err != nil {
		t.Fatalf("NewRuntime failed: %v", err)
	}
	defer runtime.Cleanup()

	err = cmd.Run(runtime)
	if err != nil {
		t.Fatalf("ApplyCmd.Run() failed: %v", err)
	}

	result, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	expectedResult := "database: postgres://localhost:5432/test\napi_key: secret-123"
	if string(result) != expectedResult {
		t.Errorf("Output mismatch: got %q, want %q", string(result), expectedResult)
	}
}

func setupTestEnvironment(t *testing.T, tmpDir string) (configPath, keyPath string) {
	t.Helper()

	privateKey, publicKey, err := core.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}
	defer core.WipeData(privateKey)

	keyPath = filepath.Join(tmpDir, "test.key")
	if err := core.SaveKeys(privateKey, publicKey, keyPath); err != nil {
		t.Fatalf("SaveKeys failed: %v", err)
	}

	configPath = filepath.Join(tmpDir, "kiln.toml")
	configContent := fmt.Sprintf(`[recipients]
test-user = "%s"

[files.default]
filename = "%s"
access = ["*"]
`, publicKey, filepath.Join(tmpDir, ".kiln.env"))

	err = os.WriteFile(configPath, []byte(configContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	return configPath, keyPath
}

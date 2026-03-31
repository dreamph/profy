package envloader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseEnvValueDoubleQuotedEscapes(t *testing.T) {
	got, err := parseEnvValue(`"line1\nline2\tok"`)
	if err != nil {
		t.Fatalf("parseEnvValue() error = %v", err)
	}
	want := "line1\nline2\tok"
	if got != want {
		t.Fatalf("parseEnvValue() = %q, want %q", got, want)
	}
}

func TestBuildMergedEnvNestedExpansionIsDeterministic(t *testing.T) {
	projectDir := t.TempDir()
	envFile := filepath.Join(projectDir, "dev.env")
	data := []byte("C=1\nB=${C}\nA=${B}\n")
	if err := os.WriteFile(envFile, data, 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	for i := 0; i < 30; i++ {
		merged, err := BuildMergedEnv(projectDir, []string{"dev.env"}, true)
		if err != nil {
			t.Fatalf("BuildMergedEnv() error = %v", err)
		}
		got := envSliceToMap(merged)["A"]
		if got != "1" {
			t.Fatalf("iteration %d: A=%q, want %q", i+1, got, "1")
		}
	}
}

func TestBuildMergedEnvMergeOrderAndOverride(t *testing.T) {
	t.Setenv("EXISTING", "from-os")

	projectDir := t.TempDir()
	baseEnv := filepath.Join(projectDir, "base.env")
	devEnv := filepath.Join(projectDir, "dev.env")
	if err := os.WriteFile(baseEnv, []byte("A=from-base\nB=from-base\nEXISTING=from-file\n"), 0o600); err != nil {
		t.Fatalf("write base env file: %v", err)
	}
	if err := os.WriteFile(devEnv, []byte("B=from-dev\nC=${A}-${B}\nD='keep # raw'\nE=value # comment\n"), 0o600); err != nil {
		t.Fatalf("write dev env file: %v", err)
	}

	mergedNoOverride, err := BuildMergedEnv(projectDir, []string{"base.env", "dev.env"}, false)
	if err != nil {
		t.Fatalf("BuildMergedEnv() error = %v", err)
	}
	noOverrideMap := envSliceToMap(mergedNoOverride)
	if noOverrideMap["A"] != "from-base" {
		t.Fatalf("A = %q, want %q", noOverrideMap["A"], "from-base")
	}
	if noOverrideMap["B"] != "from-dev" {
		t.Fatalf("B = %q, want %q", noOverrideMap["B"], "from-dev")
	}
	if noOverrideMap["C"] != "from-base-from-dev" {
		t.Fatalf("C = %q, want %q", noOverrideMap["C"], "from-base-from-dev")
	}
	if noOverrideMap["D"] != "keep # raw" {
		t.Fatalf("D = %q, want %q", noOverrideMap["D"], "keep # raw")
	}
	if noOverrideMap["E"] != "value" {
		t.Fatalf("E = %q, want %q", noOverrideMap["E"], "value")
	}
	if noOverrideMap["EXISTING"] != "from-os" {
		t.Fatalf("EXISTING = %q, want %q when override=false", noOverrideMap["EXISTING"], "from-os")
	}

	mergedOverride, err := BuildMergedEnv(projectDir, []string{"base.env", "dev.env"}, true)
	if err != nil {
		t.Fatalf("BuildMergedEnv() error = %v", err)
	}
	overrideMap := envSliceToMap(mergedOverride)
	if overrideMap["EXISTING"] != "from-file" {
		t.Fatalf("EXISTING = %q, want %q when override=true", overrideMap["EXISTING"], "from-file")
	}
}

func TestParseEnvFileParsesExportCommentsAndQuotes(t *testing.T) {
	projectDir := t.TempDir()
	envFile := filepath.Join(projectDir, "dev.env")
	content := strings.Join([]string{
		"# this is a comment",
		"export APP_ENV=dev",
		`GREETING="hello\nworld"`,
		"TIMEOUT=30s  # inline comment",
		`RAW='x # y'`,
		"",
	}, "\n")
	if err := os.WriteFile(envFile, []byte(content), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	parsed, err := parseEnvFile(envFile)
	if err != nil {
		t.Fatalf("parseEnvFile() error = %v", err)
	}
	if parsed["APP_ENV"] != "dev" {
		t.Fatalf("APP_ENV = %q, want %q", parsed["APP_ENV"], "dev")
	}
	if parsed["GREETING"] != "hello\nworld" {
		t.Fatalf("GREETING = %q, want %q", parsed["GREETING"], "hello\\nworld with real newline")
	}
	if parsed["TIMEOUT"] != "30s" {
		t.Fatalf("TIMEOUT = %q, want %q", parsed["TIMEOUT"], "30s")
	}
	if parsed["RAW"] != "x # y" {
		t.Fatalf("RAW = %q, want %q", parsed["RAW"], "x # y")
	}
}

func TestParseEnvFileRejectsInvalidLines(t *testing.T) {
	projectDir := t.TempDir()

	invalidFormat := filepath.Join(projectDir, "invalid-format.env")
	if err := os.WriteFile(invalidFormat, []byte("NOT_VALID_LINE\n"), 0o600); err != nil {
		t.Fatalf("write invalid format file: %v", err)
	}
	if _, err := parseEnvFile(invalidFormat); err == nil {
		t.Fatal("expected parseEnvFile to fail for invalid env format")
	}

	invalidKey := filepath.Join(projectDir, "invalid-key.env")
	if err := os.WriteFile(invalidKey, []byte("1BAD=value\n"), 0o600); err != nil {
		t.Fatalf("write invalid key file: %v", err)
	}
	if _, err := parseEnvFile(invalidKey); err == nil {
		t.Fatal("expected parseEnvFile to fail for invalid env key")
	}
}

func TestValidateRequiredKeys(t *testing.T) {
	env := []string{"A=1", "B=  "}
	err := ValidateRequiredKeys(env, []string{"A", "B", "C"})
	if err == nil {
		t.Fatal("expected missing required keys error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "B") || !strings.Contains(msg, "C") {
		t.Fatalf("error = %q, expected keys B and C in message", msg)
	}
}

package projectref

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadProjectIDRejectsTraversalSegment(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	projectFile := filepath.Join(dir, ".profy.yml")
	if err := os.WriteFile(projectFile, []byte("project_id: ..\n"), 0o600); err != nil {
		t.Fatalf("write project file: %v", err)
	}

	_, err := ReadProjectID(projectFile)
	if err == nil {
		t.Fatal("expected an error for invalid traversal project id")
	}
}

func TestReadProjectIDAcceptsNormalID(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	projectFile := filepath.Join(dir, ".profy.yml")
	if err := os.WriteFile(projectFile, []byte("project_id: myapp-prod\n"), 0o600); err != nil {
		t.Fatalf("write project file: %v", err)
	}

	id, err := ReadProjectID(projectFile)
	if err != nil {
		t.Fatalf("ReadProjectID() error = %v", err)
	}
	if id != "myapp-prod" {
		t.Fatalf("ReadProjectID() = %q, want %q", id, "myapp-prod")
	}
}

func TestReadProjectIDRejectsPathSeparator(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	projectFile := filepath.Join(dir, ".profy.yml")
	if err := os.WriteFile(projectFile, []byte("project_id: a/b\n"), 0o600); err != nil {
		t.Fatalf("write project file: %v", err)
	}

	_, err := ReadProjectID(projectFile)
	if err == nil {
		t.Fatal("expected an error for invalid project id with path separator")
	}
}

func TestParseProjectIDYAMLFallbackPlainScalar(t *testing.T) {
	t.Parallel()

	id, err := parseProjectIDYAML([]byte("my-plain-project\n"))
	if err != nil {
		t.Fatalf("parseProjectIDYAML() error = %v", err)
	}
	if id != "my-plain-project" {
		t.Fatalf("id = %q, want %q", id, "my-plain-project")
	}
}

func TestParseProjectIDYAMLMissingKey(t *testing.T) {
	t.Parallel()

	_, err := parseProjectIDYAML([]byte("name: app\n"))
	if err == nil {
		t.Fatal("expected error for missing required key")
	}
	if !strings.Contains(err.Error(), "project_id") {
		t.Fatalf("error = %q, expected message to mention project_id", err.Error())
	}
}

func TestParseProjectIDYAMLEmptyFile(t *testing.T) {
	t.Parallel()

	_, err := parseProjectIDYAML([]byte("   \n"))
	if err == nil {
		t.Fatal("expected error for empty project file")
	}
}

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseArgsUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "no args", args: nil},
		{name: "profile only", args: []string{"dev"}},
		{name: "print env without profile", args: []string{"--print-env"}},
		{name: "watch interval zero", args: []string{"--watch-interval", "0s", "dev", "echo", "ok"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, err := parseArgs(tc.args)
			if err == nil {
				t.Fatalf("parseArgs(%v) expected error, got nil", tc.args)
			}
		})
	}
}

func TestParseArgsPrintEnvWithoutCommand(t *testing.T) {
	opts, profile, command, err := parseArgs([]string{"--print-env", "dev"})
	if err != nil {
		t.Fatalf("parseArgs() error = %v", err)
	}
	if !opts.PrintEnv {
		t.Fatal("PrintEnv should be true")
	}
	if profile != "dev" {
		t.Fatalf("profile = %q, want %q", profile, "dev")
	}
	if len(command) != 0 {
		t.Fatalf("len(command) = %d, want 0", len(command))
	}
}

func TestParseArgsFlagsAndCommand(t *testing.T) {
	opts, profile, command, err := parseArgs([]string{
		"--override",
		"--verbose",
		"--config-home", "/tmp/profy-home",
		"--project-file", "custom.yml",
		"prod",
		"go", "version",
	})
	if err != nil {
		t.Fatalf("parseArgs() error = %v", err)
	}

	if !opts.Override {
		t.Fatal("Override should be true")
	}
	if !opts.Verbose {
		t.Fatal("Verbose should be true")
	}
	if opts.ConfigHome != "/tmp/profy-home" {
		t.Fatalf("ConfigHome = %q, want %q", opts.ConfigHome, "/tmp/profy-home")
	}
	if opts.ProjectFile != "custom.yml" {
		t.Fatalf("ProjectFile = %q, want %q", opts.ProjectFile, "custom.yml")
	}
	if profile != "prod" {
		t.Fatalf("profile = %q, want %q", profile, "prod")
	}
	if len(command) != 2 || command[0] != "go" || command[1] != "version" {
		t.Fatalf("command = %v, want [go version]", command)
	}
}

func TestBuildWatchFiles(t *testing.T) {
	projectFile := ".profy.yml"
	projectDir := filepath.Join(string(filepath.Separator), "tmp", "myapp")
	files := buildWatchFiles(projectFile, projectDir, []string{"base.env", "secret/dev.env"})

	if len(files) != 4 {
		t.Fatalf("len(files) = %d, want 4", len(files))
	}
	if files[0] != projectFile {
		t.Fatalf("files[0] = %q, want %q", files[0], projectFile)
	}
	if files[1] != filepath.Join(projectDir, "profy.json") {
		t.Fatalf("files[1] = %q, want profy.json path under project dir", files[1])
	}
	if files[2] != filepath.Join(projectDir, "base.env") {
		t.Fatalf("files[2] = %q, want base.env path under project dir", files[2])
	}
	if files[3] != filepath.Join(projectDir, "secret/dev.env") {
		t.Fatalf("files[3] = %q, want secret/dev.env path under project dir", files[3])
	}
}

func TestSnapshotFilesAndEqual(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "app.env")
	missing := filepath.Join(dir, "missing.env")

	if err := os.WriteFile(existing, []byte("A=1\n"), 0o600); err != nil {
		t.Fatalf("write existing file: %v", err)
	}
	s1 := snapshotFiles([]string{existing, missing})
	s2 := snapshotFiles([]string{existing, missing})
	if !fileStateEqual(s1, s2) {
		t.Fatal("fileStateEqual should be true for unchanged snapshots")
	}

	time.Sleep(2 * time.Millisecond)
	if err := os.WriteFile(existing, []byte("A=22\n"), 0o600); err != nil {
		t.Fatalf("rewrite existing file: %v", err)
	}
	s3 := snapshotFiles([]string{existing, missing})
	if fileStateEqual(s2, s3) {
		t.Fatal("fileStateEqual should be false after file change")
	}
}

func TestWatchForFileChanges(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, "dev.env")
	if err := os.WriteFile(envFile, []byte("A=1\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	changes := make(chan struct{}, 1)
	stop := watchForFileChanges([]string{envFile}, 10*time.Millisecond, changes)
	defer stop()

	time.Sleep(30 * time.Millisecond)
	if err := os.WriteFile(envFile, []byte("A=22\n"), 0o600); err != nil {
		t.Fatalf("rewrite env file: %v", err)
	}

	select {
	case <-changes:
	case <-time.After(800 * time.Millisecond):
		t.Fatal("expected file change notification")
	}
}

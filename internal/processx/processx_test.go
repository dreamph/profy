package processx

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"testing"
)

func TestRunEmptyCommand(t *testing.T) {
	code, err := Run(nil, os.Environ(), false)
	if err == nil {
		t.Fatal("expected error for empty command")
	}
	if code != 2 {
		t.Fatalf("code = %d, want %d", code, 2)
	}
}

func TestRunStartError(t *testing.T) {
	code, err := Run([]string{"profy-command-that-does-not-exist-123456"}, os.Environ(), false)
	if err == nil {
		t.Fatal("expected start error for invalid command")
	}
	if code != 1 {
		t.Fatalf("code = %d, want %d", code, 1)
	}
}

func TestRunReturnsChildExitCode(t *testing.T) {
	code, err := Run(exitCommand(7), os.Environ(), false)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code != 7 {
		t.Fatalf("code = %d, want %d", code, 7)
	}
}

func TestRunWithReloadEmptyCommand(t *testing.T) {
	code, err := RunWithReload(nil, func() ([]string, error) {
		return os.Environ(), nil
	}, make(chan struct{}), false)
	if err == nil {
		t.Fatal("expected error for empty command")
	}
	if code != 2 {
		t.Fatalf("code = %d, want %d", code, 2)
	}
}

func TestRunWithReloadBuilderError(t *testing.T) {
	wantErr := errors.New("builder failed")
	code, err := RunWithReload(exitCommand(0), func() ([]string, error) {
		return nil, wantErr
	}, make(chan struct{}), false)
	if err == nil {
		t.Fatal("expected env builder error")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want wrapped %v", err, wantErr)
	}
	if code != 1 {
		t.Fatalf("code = %d, want %d", code, 1)
	}
}

func TestRunWithReloadReturnsChildExitCode(t *testing.T) {
	code, err := RunWithReload(exitCommand(0), func() ([]string, error) {
		return os.Environ(), nil
	}, make(chan struct{}), false)
	if err != nil {
		t.Fatalf("RunWithReload() error = %v", err)
	}
	if code != 0 {
		t.Fatalf("code = %d, want %d", code, 0)
	}
}

func exitCommand(code int) []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/C", fmt.Sprintf("exit %d", code)}
	}
	return []string{"sh", "-c", fmt.Sprintf("exit %d", code)}
}

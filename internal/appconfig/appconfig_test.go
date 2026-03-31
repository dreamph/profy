package appconfig

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadProjectConfigAndResolveProfile(t *testing.T) {
	configHome := t.TempDir()
	projectID := "myapp"
	projectDir := filepath.Join(configHome, projectID)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}

	configPath := filepath.Join(projectDir, "profy.json")
	configContent := `{
  "configs": {
    "dev": {
      "files": ["base.env", "dev.env", "secret/dev.env"],
      "required_keys": ["APP_ENV", "APP_PORT"]
    }
  }
}`
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("write profy.json: %v", err)
	}

	cfg, err := LoadProjectConfig(projectID, configHome)
	if err != nil {
		t.Fatalf("LoadProjectConfig() error = %v", err)
	}
	if cfg.ProjectID != projectID {
		t.Fatalf("ProjectID = %q, want %q", cfg.ProjectID, projectID)
	}
	if cfg.ProjectDir != projectDir {
		t.Fatalf("ProjectDir = %q, want %q", cfg.ProjectDir, projectDir)
	}

	profile, err := cfg.ResolveProfile("dev")
	if err != nil {
		t.Fatalf("ResolveProfile() error = %v", err)
	}
	if len(profile.Files) != 3 {
		t.Fatalf("len(profile.Files) = %d, want %d", len(profile.Files), 3)
	}
	if len(profile.RequiredKeys) != 2 {
		t.Fatalf("len(profile.RequiredKeys) = %d, want %d", len(profile.RequiredKeys), 2)
	}
}

func TestLoadProjectConfigValidation(t *testing.T) {
	t.Run("empty project id", func(t *testing.T) {
		_, err := LoadProjectConfig("", t.TempDir())
		if err == nil {
			t.Fatal("expected error for empty project id")
		}
	})

	t.Run("empty config home", func(t *testing.T) {
		_, err := LoadProjectConfig("myapp", "")
		if err == nil {
			t.Fatal("expected error for empty config home")
		}
	})

	t.Run("missing profy.json", func(t *testing.T) {
		_, err := LoadProjectConfig("myapp", t.TempDir())
		if err == nil {
			t.Fatal("expected error when profy.json is missing")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		configHome := t.TempDir()
		projectDir := filepath.Join(configHome, "myapp")
		if err := os.MkdirAll(projectDir, 0o755); err != nil {
			t.Fatalf("mkdir project dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(projectDir, "profy.json"), []byte("{"), 0o600); err != nil {
			t.Fatalf("write profy.json: %v", err)
		}

		_, err := LoadProjectConfig("myapp", configHome)
		if err == nil {
			t.Fatal("expected parse error for invalid json")
		}
	})

	t.Run("no profiles", func(t *testing.T) {
		configHome := t.TempDir()
		projectDir := filepath.Join(configHome, "myapp")
		if err := os.MkdirAll(projectDir, 0o755); err != nil {
			t.Fatalf("mkdir project dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(projectDir, "profy.json"), []byte(`{"configs":{}}`), 0o600); err != nil {
			t.Fatalf("write profy.json: %v", err)
		}

		_, err := LoadProjectConfig("myapp", configHome)
		if err == nil {
			t.Fatal("expected error for missing profile definitions")
		}
	})
}

func TestResolveProfileValidation(t *testing.T) {
	absPath := filepath.Join(string(filepath.Separator), "tmp", "abs.env")
	if runtime.GOOS == "windows" {
		absPath = `C:\tmp\abs.env`
	}

	cfg := &ProjectConfig{
		Config: File{
			Configs: map[string]Profile{
				"no-files":   {Files: []string{}},
				"dot":        {Files: []string{"."}},
				"empty":      {Files: []string{" "}},
				"abs":        {Files: []string{absPath}},
				"traversal":  {Files: []string{"../secret.env"}},
				"valid":      {Files: []string{"base.env", "dev.env"}},
				"valid-nest": {Files: []string{"secret/dev.env"}},
			},
		},
	}

	cases := []struct {
		profile string
		wantErr bool
	}{
		{profile: "missing", wantErr: true},
		{profile: "no-files", wantErr: true},
		{profile: "dot", wantErr: true},
		{profile: "empty", wantErr: true},
		{profile: "abs", wantErr: true},
		{profile: "traversal", wantErr: true},
		{profile: "valid", wantErr: false},
		{profile: "valid-nest", wantErr: false},
	}

	for _, tc := range cases {
		t.Run(tc.profile, func(t *testing.T) {
			_, err := cfg.ResolveProfile(tc.profile)
			if tc.wantErr && err == nil {
				t.Fatalf("ResolveProfile(%q) expected error, got nil", tc.profile)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("ResolveProfile(%q) unexpected error: %v", tc.profile, err)
			}
		})
	}
}

package appconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type File struct {
	Configs map[string]Profile `json:"configs"`
}

type Profile struct {
	Files        []string `json:"files"`
	RequiredKeys []string `json:"required_keys"`
}

type ProjectConfig struct {
	ProjectID  string
	ProjectDir string
	Config     File
}

func LoadProjectConfig(projectID, configHome string) (*ProjectConfig, error) {
	if projectID == "" {
		return nil, fmt.Errorf("empty project id")
	}
	if configHome == "" {
		return nil, fmt.Errorf("empty config home")
	}

	projectDir := filepath.Join(configHome, projectID)
	configPath := filepath.Join(projectDir, "profy.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read external config: %w", err)
	}

	var cfg File
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse external config: %w", err)
	}
	if len(cfg.Configs) == 0 {
		return nil, fmt.Errorf("external config for project %q has no profiles", projectID)
	}

	return &ProjectConfig{ProjectID: projectID, ProjectDir: projectDir, Config: cfg}, nil
}

func (p *ProjectConfig) ResolveProfile(profile string) (Profile, error) {
	cfg, ok := p.Config.Configs[profile]
	if !ok {
		return Profile{}, fmt.Errorf("unknown profile %q", profile)
	}
	if len(cfg.Files) == 0 {
		return Profile{}, fmt.Errorf("profile %q has no env files", profile)
	}
	for _, file := range cfg.Files {
		clean := filepath.Clean(strings.TrimSpace(file))
		if clean == "." || clean == "" {
			return Profile{}, fmt.Errorf("profile %q has invalid env file path %q", profile, file)
		}
		if filepath.IsAbs(clean) {
			return Profile{}, fmt.Errorf("profile %q has absolute env file path %q", profile, file)
		}
		if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return Profile{}, fmt.Errorf("profile %q has path traversal env file path %q", profile, file)
		}
	}
	return cfg, nil
}

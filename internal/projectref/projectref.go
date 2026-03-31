package projectref

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func DefaultConfigHome() string {
	if v := strings.TrimSpace(os.Getenv("PROFY_CONFIG_HOME")); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".profy"
	}
	return filepath.Join(home, ".profy")
}

func ReadProjectID(projectFile string) (string, error) {
	data, err := os.ReadFile(projectFile)
	if err != nil {
		return "", fmt.Errorf("read project config file: %w", err)
	}
	projectID, err := parseProjectIDYAML(data)
	if err != nil {
		return "", fmt.Errorf("parse project config file %q: %w", projectFile, err)
	}
	if strings.Contains(projectID, "/") || strings.Contains(projectID, `\`) {
		return "", fmt.Errorf("invalid project id %q", projectID)
	}
	if projectID == "." || projectID == ".." {
		return "", fmt.Errorf("invalid project id %q", projectID)
	}
	return projectID, nil
}

type fileConfig struct {
	ProjectID string `yaml:"project_id"`
}

func parseProjectIDYAML(data []byte) (string, error) {
	content := strings.TrimSpace(string(data))
	if content == "" {
		return "", fmt.Errorf("file is empty")
	}

	var cfg fileConfig
	if err := yaml.Unmarshal(data, &cfg); err == nil {
		id := strings.TrimSpace(cfg.ProjectID)
		if id != "" {
			return id, nil
		}
		if strings.Contains(content, ":") || strings.Contains(content, "\n") {
			return "", fmt.Errorf(`missing required key "project_id"`)
		}
	}

	// Backward compatible fallback: allow plain scalar value in file.
	id := strings.Trim(strings.TrimSpace(content), `"'`)
	if id == "" {
		return "", fmt.Errorf(`"project_id" is empty`)
	}
	return id, nil
}

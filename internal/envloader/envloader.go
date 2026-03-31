package envloader

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func BuildMergedEnv(projectDir string, files []string, override bool) ([]string, error) {
	current := envSliceToMap(os.Environ())
	originalKeys := make(map[string]struct{}, len(current))
	for k := range current {
		originalKeys[k] = struct{}{}
	}

	fileMerged := make(map[string]string)
	for _, rel := range files {
		parsed, err := parseEnvFile(filepath.Join(projectDir, rel))
		if err != nil {
			return nil, fmt.Errorf("load env for profile failed: %w", err)
		}
		for k, v := range parsed {
			fileMerged[k] = v
		}
	}
	expandMergedValues(fileMerged, current)
	for k, v := range fileMerged {
		_, existedInOS := originalKeys[k]
		if existedInOS && !override {
			continue
		}
		current[k] = v
	}
	return envMapToSlice(current), nil
}

func ValidateRequiredKeys(env []string, required []string) error {
	if len(required) == 0 {
		return nil
	}
	m := envSliceToMap(env)
	missing := make([]string, 0)
	for _, k := range required {
		if strings.TrimSpace(m[k]) == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("required env keys missing: %s", strings.Join(missing, ", "))
	}
	return nil
}

func PrintEnv(w io.Writer, env []string) {
	items := append([]string(nil), env...)
	sort.Strings(items)
	for _, item := range items {
		fmt.Fprintln(w, item)
	}
}

func parseEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open env file: %w", err)
	}
	defer file.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("invalid env format at line %d", lineNo)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return nil, fmt.Errorf("empty env key at line %d", lineNo)
		}
		if !isValidEnvKey(key) {
			return nil, fmt.Errorf("invalid env key %q at line %d", key, lineNo)
		}
		parsedValue, err := parseEnvValue(value)
		if err != nil {
			return nil, fmt.Errorf("invalid env value for key %q at line %d: %w", key, lineNo, err)
		}
		result[key] = parsedValue
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func parseEnvValue(v string) (string, error) {
	if v == "" {
		return "", nil
	}
	if len(v) >= 2 {
		if v[0] == '"' && v[len(v)-1] == '"' {
			return strconv.Unquote(v)
		}
		if v[0] == '\'' && v[len(v)-1] == '\'' {
			return v[1 : len(v)-1], nil
		}
	}
	if idx := strings.Index(v, " #"); idx >= 0 {
		v = strings.TrimSpace(v[:idx])
	}
	return v, nil
}

func isValidEnvKey(k string) bool {
	for i, r := range k {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case r == '_':
		case i > 0 && r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return k != ""
}

func envSliceToMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, item := range env {
		if k, v, ok := strings.Cut(item, "="); ok {
			m[k] = v
		}
	}
	return m
}

func envMapToSlice(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]string, 0, len(m))
	for _, k := range keys {
		out = append(out, k+"="+m[k])
	}
	return out
}

func expandValue(value string, fileMerged map[string]string, current map[string]string) string {
	return os.Expand(value, func(key string) string {
		if v, ok := fileMerged[key]; ok {
			return v
		}
		if v, ok := current[key]; ok {
			return v
		}
		return ""
	})
}

func expandMergedValues(fileMerged map[string]string, current map[string]string) {
	if len(fileMerged) == 0 {
		return
	}
	keys := make([]string, 0, len(fileMerged))
	for k := range fileMerged {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	maxPasses := len(keys) + 1
	for i := 0; i < maxPasses; i++ {
		changed := false
		for _, k := range keys {
			expanded := expandValue(fileMerged[k], fileMerged, current)
			if expanded != fileMerged[k] {
				fileMerged[k] = expanded
				changed = true
			}
		}
		if !changed {
			return
		}
	}
}

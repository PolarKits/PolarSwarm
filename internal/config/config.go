package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// sectionKeys defines known keys per section for unknown field detection.
var sectionKeys = map[string]map[string]bool{
	"github":      {"owner": true, "repo": true},
	"repository": {"owner": true, "name": true},
	"workflow":   {"target_label": true, "dry_run": true, "confirm_writes": true},
	"runtime":    {"dry_run": true, "confirm_writes": true},
}

const DefaultPath = ".polarswarm/core.toml"

type Config struct {
	GitHub   GitHubConfig
	Workflow WorkflowConfig
}

type GitHubConfig struct {
	Owner string
	Repo  string
}

type WorkflowConfig struct {
	TargetLabel   string
	DryRun        bool
	ConfirmWrites bool
}

func Load(path string) (Config, error) {
	if path == "" {
		path = DefaultPath
	}

	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("config file %q not found", path)
		}
		return Config{}, fmt.Errorf("read config file %q: %w", path, err)
	}

	cfg, err := parse(path, string(content))
	if err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(path); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate(path string) error {
	if strings.TrimSpace(c.GitHub.Owner) == "" {
		return fmt.Errorf("%s: [github].owner is required", path)
	}
	if strings.TrimSpace(c.GitHub.Repo) == "" {
		return fmt.Errorf("%s: [github].repo is required", path)
	}
	if strings.TrimSpace(c.Workflow.TargetLabel) == "" {
		return fmt.Errorf("%s: [workflow].target_label is required", path)
	}
	return nil
}

func (c Config) Summary() string {
	return fmt.Sprintf("github=%s/%s target_label=%s dry_run=%t confirm_writes=%t", c.GitHub.Owner, c.GitHub.Repo, c.Workflow.TargetLabel, c.Workflow.DryRun, c.Workflow.ConfirmWrites)
}

func parse(path, content string) (Config, error) {
	cfg := Config{
		Workflow: WorkflowConfig{
			TargetLabel: "status:new",
			DryRun:      true,
		},
	}

	section := ""
	var unknownWarnings []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := stripComment(scanner.Text())
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			if section == "" {
				return Config{}, fmt.Errorf("%s:%d: section name is required", path, lineNo)
			}
			continue
		}

		key, raw, ok := strings.Cut(line, "=")
		if !ok {
			return Config{}, fmt.Errorf("%s:%d: expected key = value", path, lineNo)
		}
		key = strings.TrimSpace(key)
		raw = strings.TrimSpace(raw)
		if key == "" {
			return Config{}, fmt.Errorf("%s:%d: key is required", path, lineNo)
		}

		if known, ok := sectionKeys[section]; ok && !known[key] {
			unknownWarnings = append(unknownWarnings, fmt.Sprintf("%s:%d: unknown field [%s].%s", path, lineNo, section, key))
			continue
		}

		if err := assign(&cfg, path, lineNo, section, key, raw); err != nil {
			return Config{}, err
		}
	}
	if err := scanner.Err(); err != nil {
		return Config{}, fmt.Errorf("%s: scan config: %w", path, err)
	}

	// Emit unknown field warnings (non-fatal).
	for _, w := range unknownWarnings {
		fmt.Fprintf(os.Stderr, "WARNING: %s\n", w)
	}

	return cfg, nil
}

func assign(cfg *Config, path string, lineNo int, section, key, raw string) error {
	switch section {
	case "github":
		switch key {
		case "owner":
			value, err := parseString(path, lineNo, raw)
			if err != nil {
				return err
			}
			cfg.GitHub.Owner = value
		case "repo":
			value, err := parseString(path, lineNo, raw)
			if err != nil {
				return err
			}
			cfg.GitHub.Repo = value
		}
	case "repository":
		switch key {
		case "owner":
			value, err := parseString(path, lineNo, raw)
			if err != nil {
				return err
			}
			cfg.GitHub.Owner = value
		case "name":
			value, err := parseString(path, lineNo, raw)
			if err != nil {
				return err
			}
			cfg.GitHub.Repo = value
		}
	case "workflow":
		switch key {
		case "target_label":
			value, err := parseString(path, lineNo, raw)
			if err != nil {
				return err
			}
			cfg.Workflow.TargetLabel = value
		case "dry_run":
			value, err := parseBool(path, lineNo, raw)
			if err != nil {
				return err
			}
			cfg.Workflow.DryRun = value
		case "confirm_writes":
			value, err := parseBool(path, lineNo, raw)
			if err != nil {
				return err
			}
			cfg.Workflow.ConfirmWrites = value
		}
	case "runtime":
		switch key {
		case "dry_run":
			value, err := parseBool(path, lineNo, raw)
			if err != nil {
				return err
			}
			cfg.Workflow.DryRun = value
		case "confirm_writes":
			value, err := parseBool(path, lineNo, raw)
			if err != nil {
				return err
			}
			cfg.Workflow.ConfirmWrites = value
		}
	}

	return nil
}

func parseString(path string, lineNo int, raw string) (string, error) {
	value, err := strconv.Unquote(raw)
	if err != nil {
		return "", fmt.Errorf("%s:%d: expected quoted string", path, lineNo)
	}
	return value, nil
}

func parseBool(path string, lineNo int, raw string) (bool, error) {
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s:%d: expected boolean", path, lineNo)
	}
	return value, nil
}

func stripComment(line string) string {
	inString := false
	escaped := false
	for i, r := range line {
		switch {
		case escaped:
			escaped = false
		case r == '\\' && inString:
			escaped = true
		case r == '"':
			inString = !inString
		case r == '#' && !inString:
			return line[:i]
		}
	}
	return line
}

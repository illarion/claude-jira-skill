// Package config loads and stores the .jiraskillrc credentials file.
//
// The file lives at the root of a company/project tree; it is found by walking
// up from the working directory, so each directory tree resolves to its own
// Jira instance automatically.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DotFile is the credentials file name.
const DotFile = ".jiraskillrc"

// Config is the on-disk schema of .jiraskillrc.
type Config struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Email    string `json:"email"`
	Token    string `json:"token"`
	Projects string `json:"projects"`
}

// Find walks up from the working directory looking for DotFile.
func Find() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for {
		candidate := filepath.Join(dir, DotFile)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// Load finds and reads credentials, printing the instance banner to stderr.
// It exits the process when no config exists or the file is unreadable.
func Load() *Config {
	path, ok := Find()
	if !ok {
		fmt.Fprintf(os.Stderr, "No %s found in any parent directory.\n", DotFile)
		fmt.Fprintln(os.Stderr, "Run: jira auth login")
		os.Exit(1)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot read %s: %v\n", path, err)
		os.Exit(1)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid %s: %v\n", path, err)
		os.Exit(1)
	}
	name := cfg.Name
	if name == "" {
		name = filepath.Base(filepath.Dir(path))
	}
	fmt.Fprintf(os.Stderr, "[jira: %s — %s]\n", name, cfg.URL)
	return &cfg
}

// ProjectList returns the configured default projects as a slice.
func (c *Config) ProjectList() []string {
	var out []string
	for _, p := range strings.Split(c.Projects, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Save writes cfg to path as pretty JSON with 0600 permissions.
func Save(path string, cfg *Config) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

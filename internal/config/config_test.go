package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFindWalksUp(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "company", "repo")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, DotFile), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)
	path, ok := Find()
	if !ok {
		t.Fatal("expected to find the dotfile in a parent directory")
	}
	if path != filepath.Join(root, DotFile) {
		t.Errorf("got %q", path)
	}
}

func TestFindMissing(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, ok := Find(); ok {
		t.Fatal("expected no dotfile")
	}
}

func TestSaveAndSchema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", DotFile)
	cfg := &Config{Name: "acme", URL: "https://acme.atlassian.net", Email: "e@x.io", Token: "tok", Projects: "PROJ,CORE"}
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("permissions %v, want 0600", info.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var back map[string]any
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"name", "url", "email", "token", "projects"} {
		if _, ok := back[key]; !ok {
			t.Errorf("missing key %q in %s", key, data)
		}
	}
	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}
	if len(loaded.ProjectList()) != 2 || loaded.ProjectList()[0] != "PROJ" {
		t.Errorf("ProjectList = %v", loaded.ProjectList())
	}
}

func TestProjectListFiltersEmpty(t *testing.T) {
	c := &Config{Projects: " A , ,B, "}
	got := c.ProjectList()
	if len(got) != 2 || got[0] != "A" || got[1] != "B" {
		t.Errorf("got %v", got)
	}
	if len((&Config{}).ProjectList()) != 0 {
		t.Error("empty projects should give an empty list")
	}
}

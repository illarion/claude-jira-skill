package main

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
)

var versionTag = regexp.MustCompile(`v\d+\.\d+\.\d+`)

func readVersion(t *testing.T, path string, v any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}

// SKILL.md names the binary version that the skill needs and downloads. It
// must be the plugin version, or a plugin update leaves the old binary in use.
func TestSkillPinsPluginVersion(t *testing.T) {
	var plugin struct {
		Version string `json:"version"`
	}
	readVersion(t, ".claude-plugin/plugin.json", &plugin)
	var market struct {
		Plugins []struct {
			Version string `json:"version"`
		} `json:"plugins"`
	}
	readVersion(t, ".claude-plugin/marketplace.json", &market)
	if len(market.Plugins) != 1 || market.Plugins[0].Version != plugin.Version {
		t.Errorf("marketplace.json plugins = %+v, want one plugin of version %s", market.Plugins, plugin.Version)
	}

	data, err := os.ReadFile("skills/jira/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	skill := string(data)
	want := "v" + plugin.Version
	if !strings.Contains(skill, "needs version `"+want+"`") {
		t.Errorf("SKILL.md does not state that it needs version %s", want)
	}
	if n := strings.Count(skill, "releases/download/"+want+"/"); n != 2 {
		t.Errorf("SKILL.md has %d download URLs of %s, want 2", n, want)
	}
	if strings.Contains(skill, "releases/latest/download") {
		t.Error("SKILL.md downloads the latest release, not the pinned version")
	}
	for _, tag := range versionTag.FindAllString(skill, -1) {
		if tag != want {
			t.Errorf("SKILL.md names version %s, want %s", tag, want)
		}
	}
}

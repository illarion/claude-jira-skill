package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"unicode/utf8"

	"github.com/illarion/claude-jira-skill/internal/client"
	"github.com/illarion/claude-jira-skill/internal/config"
)

// Fail prints a message to stderr and exits with status 1.
func Fail(msg string, a ...any) {
	fmt.Fprintf(os.Stderr, msg+"\n", a...)
	os.Exit(1)
}

func die(err error) {
	Fail("%v", err)
}

// start loads credentials (printing the instance banner) and builds a client.
func start() (*config.Config, *client.Client) {
	cfg := config.Load()
	return cfg, client.New(cfg.URL, cfg.Email, cfg.Token)
}

func mustGet(c *client.Client, path string) any {
	v, err := c.Get(path)
	if err != nil {
		die(err)
	}
	return v
}

// printJSON writes indented JSON to stdout without HTML escaping,
// matching the former Python json.dumps output.
func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		die(err)
	}
}

// warnSummary prints a warning when a summary exceeds 70 characters.
func warnSummary(summary string) {
	n := utf8.RuneCountInString(summary)
	if n <= 70 {
		return
	}
	fmt.Fprintf(os.Stderr, "Warning: summary is %d chars, limit is 70. Use short product language.\n", n)
}

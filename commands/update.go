package commands

import (
	"flag"
	"fmt"
	"strings"

	"github.com/illarion/claude-jira-skill/internal/adf"
	"github.com/illarion/claude-jira-skill/internal/jx"
	"github.com/illarion/claude-jira-skill/internal/users"
)

// Update changes fields on an issue and prints what was updated.
func Update(args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	summary := fs.String("summary", "", "New summary")
	priority := fs.String("priority", "", "Priority name (e.g. Major)")
	description := fs.String("description", "", "Description in light markup (converted to ADF)")
	descriptionFile := fs.String("description-file", "", "Path to a file with light markup or raw ADF JSON")
	fixversion := fs.String("fixversion", "", "Fix version name (e.g. 1.118.2)")
	assignee := fs.String("assignee", "", "Assignee display name")
	pos := parseFlags(fs, args)
	if len(pos) < 1 {
		Fail("Usage: jira update ISSUE-KEY [--summary S] [--priority P] [--description markup] [--description-file F] [--fixversion V] [--assignee \"Name\"]")
	}
	key := strings.ToUpper(pos[0])
	_, c := start()

	fields := map[string]any{}
	var updated []string
	if *summary != "" {
		warnSummary(*summary)
		fields["summary"] = *summary
		updated = append(updated, "summary")
	}
	if *priority != "" {
		fields["priority"] = map[string]any{"name": *priority}
		updated = append(updated, "priority")
	}
	if *descriptionFile != "" {
		d, err := adf.LoadDescription(*descriptionFile, c.BaseURL)
		if err != nil {
			die(err)
		}
		fields["description"] = d
		updated = append(updated, "description")
	}
	if _, exists := fields["description"]; !exists && *description != "" {
		fields["description"] = adf.TextToADF(*description, c.BaseURL)
		updated = append(updated, "description")
	}
	if *fixversion != "" {
		fields["fixVersions"] = []any{map[string]any{"name": *fixversion}}
		updated = append(updated, "fixVersions")
	}
	if *assignee != "" {
		u, err := users.Find(c, *assignee)
		if err != nil {
			die(err)
		}
		fields["assignee"] = map[string]any{"accountId": jx.Str(u["accountId"])}
		updated = append(updated, "assignee")
	}
	if len(fields) == 0 {
		Fail("No fields to update. Use --summary, --priority, --description, --description-file, --fixversion, or --assignee.")
	}

	if _, err := c.Put("/rest/api/3/issue/"+key, map[string]any{"fields": fields}); err != nil {
		die(err)
	}
	fmt.Printf("%s: Updated %s\n", key, strings.Join(updated, ", "))
}

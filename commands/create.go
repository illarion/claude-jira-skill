package commands

import (
	"flag"
	"strings"

	"github.com/illarion/claude-jira-skill/internal/adf"
	"github.com/illarion/claude-jira-skill/internal/jx"
	"github.com/illarion/claude-jira-skill/internal/users"
)

// Create files a new issue and prints {key, url}.
func Create(args []string) {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	typeFlag := fs.String("type", "Task", "Issue type (default: Task)")
	priority := fs.String("priority", "", "Priority name (e.g. High)")
	description := fs.String("description", "", "Description in light markup (converted to ADF)")
	descriptionFile := fs.String("description-file", "", "Path to a file with light markup or raw ADF JSON")
	assignee := fs.String("assignee", "", "Assignee display name")
	fixversion := fs.String("fixversion", "", "Fix version name (e.g. 1.118.2)")
	pos := parseFlags(fs, args)
	if len(pos) < 2 {
		Fail(`Usage: jira create PROJECT "Summary" [--type Bug] [--priority High] [--description markup] [--description-file F] [--assignee "Name"] [--fixversion 1.2]`)
	}
	project, summary := strings.ToUpper(pos[0]), pos[1]
	_, c := start()
	warnSummary(summary)

	fields := map[string]any{
		"project":   map[string]any{"key": project},
		"summary":   summary,
		"issuetype": map[string]any{"name": *typeFlag},
	}
	if *priority != "" {
		fields["priority"] = map[string]any{"name": *priority}
	}
	if *descriptionFile != "" {
		d, err := adf.LoadDescription(*descriptionFile, c.BaseURL)
		if err != nil {
			die(err)
		}
		fields["description"] = d
	} else if *description != "" {
		fields["description"] = adf.TextToADF(*description, c.BaseURL)
	}
	if *fixversion != "" {
		fields["fixVersions"] = []any{map[string]any{"name": *fixversion}}
	}
	if *assignee != "" {
		u, err := users.Find(c, *assignee)
		if err != nil {
			die(err)
		}
		fields["assignee"] = map[string]any{"accountId": jx.Str(u["accountId"])}
	}

	res, err := c.Post("/rest/api/3/issue", map[string]any{"fields": fields})
	if err != nil {
		die(err)
	}
	key := jx.Str(jx.Obj(res)["key"])
	printJSON(struct {
		Key string `json:"key"`
		URL string `json:"url"`
	}{key, c.BaseURL + "/browse/" + key})
}

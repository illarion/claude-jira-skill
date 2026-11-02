package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/illarion/claude-jira-skill/internal/client"
	"github.com/illarion/claude-jira-skill/internal/digest"
)

// Digest generates the period team digest, or lists workflow statuses and
// custom fields for classification.
func Digest(args []string) {
	if len(args) > 0 {
		switch args[0] {
		case "fields":
			digestFields()
			return
		case "statuses":
			digestStatuses(args[1:])
			return
		}
	}

	fs := flag.NewFlagSet("digest", flag.ExitOnError)
	jsonFlag := fs.Bool("json", false, "Output JSON format")
	outputFile := fs.String("output-file", "", "Write full JSON to file, print team_stats to stdout")
	devStatuses := fs.String("dev-statuses", "", "Comma-separated dev statuses")
	reviewStatuses := fs.String("review-statuses", "", "Comma-separated review statuses")
	testStatuses := fs.String("test-statuses", "", "Comma-separated test statuses")
	testedStatuses := fs.String("tested-statuses", "", "Comma-separated tested statuses")
	executorFieldID := fs.String("executor-field-id", "", "Custom field ID for executor")
	pos := parseFlags(fs, args)
	if len(pos) < 1 || len(pos) > 2 {
		Fail("Usage: jira digest START_DATE [PROJECTS] [--json] [--output-file F] [--dev-statuses ...] [--review-statuses ...] [--test-statuses ...] [--tested-statuses ...] [--executor-field-id ID]")
	}
	startDate := pos[0]

	cfg, c := start()
	projects := cfg.ProjectList()
	if len(pos) == 2 {
		projects = strings.Split(pos[1], ",")
	}
	if len(projects) == 0 {
		Fail("No projects specified and none configured in .jiraskillrc")
	}
	devS := statusSet(*devStatuses)
	reviewS := statusSet(*reviewStatuses)
	testS := statusSet(*testStatuses)
	testedS := statusSet(*testedStatuses)

	jql := fmt.Sprintf(`project in (%s) AND updated >= "%s" ORDER BY project, updated DESC`,
		strings.Join(projects, ", "), startDate)
	fieldsParam := "summary,status,assignee,priority,updated,issuetype,resolution,fixVersions"
	expand := ""
	if *jsonFlag {
		fieldsParam += ",description"
		if *executorFieldID != "" {
			fieldsParam += "," + *executorFieldID
		}
		expand = "changelog"
	}
	issues, err := client.Search(c, jql, fieldsParam, 0, expand)
	if err != nil {
		die(err)
	}

	if !*jsonFlag {
		fmt.Printf("## Team Digest — week of %s\n\n%s\n", startDate, digest.BuildReport(issues))
		return
	}

	tickets, err := digest.BuildJSON(c, issues, startDate, *executorFieldID, devS, reviewS, testS, testedS)
	if err != nil {
		die(err)
	}
	stats := digest.BuildTeamStats(tickets, reviewS)
	output := struct {
		Tickets   []digest.Ticket   `json:"tickets"`
		TeamStats *digest.TeamStats `json:"team_stats"`
	}{tickets, stats}

	if *outputFile == "" {
		printJSON(output)
		return
	}
	f, err := os.OpenFile(*outputFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		die(err)
	}
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		die(err)
	}
	if err := f.Close(); err != nil {
		die(err)
	}
	// The file may have existed with wider permissions.
	if err := os.Chmod(*outputFile, 0o600); err != nil {
		die(err)
	}
	printJSON(stats)
}

func digestStatuses(args []string) {
	cfg, c := start()
	projects := cfg.ProjectList()
	if len(args) > 0 && args[0] != "" {
		projects = strings.Split(args[0], ",")
	}
	if len(projects) == 0 {
		Fail("No projects specified and none configured in .jiraskillrc")
	}
	names, err := digest.FetchProjectStatuses(c, projects)
	if err != nil {
		die(err)
	}
	printJSON(names)
}

func digestFields() {
	_, c := start()
	fields, err := digest.FetchCustomFields(c)
	if err != nil {
		die(err)
	}
	printJSON(fields)
}

func statusSet(s string) map[string]bool {
	out := map[string]bool{}
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out[part] = true
		}
	}
	return out
}

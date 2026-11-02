package commands

import (
	"flag"
	"strings"

	"github.com/illarion/claude-jira-skill/internal/client"
	"github.com/illarion/claude-jira-skill/internal/jx"
)

// Search runs a JQL search and prints a simplified JSON array.
func Search(args []string) {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	fieldsFlag := fs.String("fields", "summary,status,assignee,priority", "Comma-separated fields")
	maxFlag := fs.Int("max", 50, "Maximum results")
	pos := parseFlags(fs, args)
	if len(pos) < 1 {
		Fail(`Usage: jira search "JQL query" [--fields summary,status] [--max 20]`)
	}
	_, c := start()

	// --max 0 (or less) asks for nothing, as the former script did.
	if *maxFlag <= 0 {
		printJSON([]any{})
		return
	}
	issues, err := client.Search(c, pos[0], *fieldsFlag, *maxFlag, "")
	if err != nil {
		die(err)
	}
	results := []any{}
	for _, iss := range issues {
		results = append(results, simplifyIssue(jx.Obj(iss), *fieldsFlag))
	}
	printJSON(results)
}

// simplifyIssue keeps the key first, then the requested fields in order.
func simplifyIssue(issue map[string]any, fields string) *jx.Ordered {
	f := jx.Obj(issue["fields"])
	out := jx.NewOrdered().Set("key", jx.Str(issue["key"]))
	for _, fl := range strings.Split(fields, ",") {
		fl = strings.TrimSpace(fl)
		switch fl {
		case "summary":
			out.Set("summary", jx.Str(f["summary"]))
		case "status":
			out.Set("status", jx.Str(jx.Obj(f["status"])["name"]))
		case "assignee":
			if a := jx.Obj(f["assignee"]); a != nil {
				out.Set("assignee", jx.Str(a["displayName"]))
				continue
			}
			out.Set("assignee", "Unassigned")
		case "priority":
			if p := jx.Obj(f["priority"]); p != nil {
				out.Set("priority", jx.Str(p["name"]))
				continue
			}
			out.Set("priority", "None")
		default:
			if v, ok := f[fl]; ok {
				out.Set(fl, v)
				continue
			}
			out.Set(fl, "")
		}
	}
	return out
}

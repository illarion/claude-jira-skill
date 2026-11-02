package commands

import (
	"fmt"
	"strings"

	"github.com/illarion/claude-jira-skill/internal/jx"
	"github.com/illarion/claude-jira-skill/internal/users"
)

// Assign sets or clears the assignee of an issue.
func Assign(args []string) {
	if len(args) < 2 {
		Fail("Usage: jira assign ISSUE-KEY \"Display Name\"\n       jira assign ISSUE-KEY --unassign")
	}
	key := strings.ToUpper(args[0])
	_, c := start()

	if args[1] == "--unassign" {
		if _, err := c.Put("/rest/api/3/issue/"+key+"/assignee", map[string]any{"accountId": nil}); err != nil {
			die(err)
		}
		fmt.Printf("%s: Unassigned\n", key)
		return
	}

	u, err := users.Find(c, args[1])
	if err != nil {
		die(err)
	}
	name := jx.Str(u["displayName"])
	if name == "" {
		name = args[1]
	}
	if _, err := c.Put("/rest/api/3/issue/"+key+"/assignee",
		map[string]any{"accountId": jx.Str(u["accountId"])}); err != nil {
		die(err)
	}
	fmt.Printf("%s: Assigned to %s\n", key, name)
}

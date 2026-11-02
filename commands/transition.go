package commands

import (
	"fmt"
	"strings"

	"github.com/illarion/claude-jira-skill/internal/jx"
)

// Transition lists or applies a workflow status transition.
func Transition(args []string) {
	if len(args) < 2 {
		Fail("Usage: jira transition ISSUE-KEY \"Status Name\"\n       jira transition ISSUE-KEY --list")
	}
	key := strings.ToUpper(args[0])
	_, c := start()

	data := jx.Obj(mustGet(c, "/rest/api/3/issue/"+key+"/transitions"))
	transitions := jx.Arr(data["transitions"])

	if args[1] == "--list" {
		names := []any{}
		for _, t := range transitions {
			names = append(names, jx.Str(jx.Obj(t)["name"]))
		}
		printJSON(names)
		return
	}

	target := strings.ToLower(args[1])
	var match map[string]any
	for _, t := range transitions {
		tm := jx.Obj(t)
		if strings.ToLower(jx.Str(tm["name"])) == target {
			match = tm
			break
		}
	}
	if match == nil {
		var names []string
		for _, t := range transitions {
			names = append(names, jx.Str(jx.Obj(t)["name"]))
		}
		Fail("No transition matching \"%s\". Available: %s", args[1], strings.Join(names, ", "))
	}

	if _, err := c.Post("/rest/api/3/issue/"+key+"/transitions",
		map[string]any{"transition": map[string]any{"id": match["id"]}}); err != nil {
		die(err)
	}
	fmt.Printf("%s: Status → %s\n", key, jx.Str(match["name"]))
}

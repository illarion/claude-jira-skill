package commands

import (
	"fmt"
	"strings"

	"github.com/illarion/claude-jira-skill/internal/jx"
)

type linkType struct {
	Name    string `json:"name"`
	Inward  string `json:"inward"`
	Outward string `json:"outward"`
}

// Link lists link types or links two issues: "ISSUE-1 <outward phrase> ISSUE-2".
func Link(args []string) {
	if len(args) == 1 && args[0] == "--list" {
		_, c := start()
		data := jx.Obj(mustGet(c, "/rest/api/3/issueLinkType"))
		types := []linkType{}
		for _, t := range jx.Arr(data["issueLinkTypes"]) {
			tm := jx.Obj(t)
			types = append(types, linkType{
				Name:    jx.Str(tm["name"]),
				Inward:  jx.Str(tm["inward"]),
				Outward: jx.Str(tm["outward"]),
			})
		}
		printJSON(types)
		return
	}

	if len(args) < 3 {
		Fail("Usage: jira link ISSUE-1 \"Link Type\" ISSUE-2\n       jira link --list")
	}
	source := strings.ToUpper(args[0])
	typeName := args[1]
	target := strings.ToUpper(args[2])
	_, c := start()

	data := jx.Obj(mustGet(c, "/rest/api/3/issueLinkType"))
	linkTypes := jx.Arr(data["issueLinkTypes"])

	lower := strings.ToLower(typeName)
	var match map[string]any
	for _, lt := range linkTypes {
		lm := jx.Obj(lt)
		if strings.ToLower(jx.Str(lm["name"])) == lower {
			match = lm
			break
		}
	}
	if match == nil {
		var names []string
		for _, lt := range linkTypes {
			names = append(names, jx.Str(jx.Obj(lt)["name"]))
		}
		Fail("No link type matching \"%s\". Available: %s", typeName, strings.Join(names, ", "))
	}

	if _, err := c.Post("/rest/api/3/issueLink", linkBody(jx.Str(match["name"]), source, target)); err != nil {
		die(err)
	}
	fmt.Printf("%s %s %s\n", source, jx.Str(match["outward"]), target)
}

// linkBody builds the issueLink payload for "source <outward phrase> target".
// Jira shows the outward phrase on the issue sent as inwardIssue.
func linkBody(typeName, source, target string) map[string]any {
	return map[string]any{
		"type":         map[string]any{"name": typeName},
		"inwardIssue":  map[string]any{"key": source},
		"outwardIssue": map[string]any{"key": target},
	}
}

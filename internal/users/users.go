// Package users resolves a display name to a Jira account.
package users

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/illarion/claude-jira-skill/internal/client"
	"github.com/illarion/claude-jira-skill/internal/jx"
)

// Find returns the user whose display name matches exactly (case-insensitive).
// On ambiguity it returns an error listing the candidates.
func Find(c *client.Client, displayName string) (map[string]any, error) {
	res, err := c.Get("/rest/api/3/user/search?query=" + url.QueryEscape(displayName))
	if err != nil {
		return nil, err
	}
	list := jx.Arr(res)
	if len(list) == 0 {
		return nil, fmt.Errorf("No user found matching \"%s\"", displayName)
	}
	target := strings.ToLower(displayName)
	for _, u := range list {
		um := jx.Obj(u)
		if strings.ToLower(jx.Str(um["displayName"])) == target {
			return um, nil
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "No exact match for \"%s\". Candidates:", displayName)
	for _, u := range list {
		um := jx.Obj(u)
		id := jx.Str(um["emailAddress"])
		if id == "" {
			id = jx.Str(um["accountId"])
		}
		fmt.Fprintf(&b, "\n  - %s (%s)", jx.Str(um["displayName"]), id)
	}
	return nil, errors.New(b.String())
}

package commands

import (
	"fmt"
	"strings"

	"github.com/illarion/claude-jira-skill/internal/adf"
)

// Comment posts a light-markup comment on an issue.
func Comment(args []string) {
	if len(args) < 2 {
		Fail(`Usage: jira comment ISSUE-KEY "comment text"`)
	}
	key := strings.ToUpper(args[0])
	text := args[1]
	_, c := start()

	if _, err := c.Post("/rest/api/3/issue/"+key+"/comment",
		map[string]any{"body": adf.TextToADF(text, c.BaseURL)}); err != nil {
		die(err)
	}
	fmt.Printf("Commented on %s\n", key)
}

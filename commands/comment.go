package commands

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/illarion/claude-jira-skill/internal/adf"
)

const commentUsage = `Usage: jira comment ISSUE-KEY "comment text" | --description-file F`

// Comment posts a comment on an issue from positional text or a file.
func Comment(args []string) {
	key, text, file, err := commentArgs(args)
	if err != nil {
		Fail("%v\n%s", err, commentUsage)
	}
	_, c := start()

	body := map[string]any{}
	if file != "" {
		d, err := adf.LoadDescription(file, c.BaseURL)
		if err != nil {
			die(err)
		}
		body["body"] = d
	} else {
		body["body"] = adf.TextToADF(text, c.BaseURL)
	}

	if _, err := c.Post("/rest/api/3/issue/"+key+"/comment", body); err != nil {
		die(err)
	}
	fmt.Printf("Commented on %s\n", key)
}

// commentArgs parses `ISSUE-KEY "text"` or `ISSUE-KEY --description-file F`.
// Exactly one body source must be given, and the text must not look like
// a flag, so a mistyped option is never posted as a comment.
func commentArgs(args []string) (key, text, file string, err error) {
	fs := flag.NewFlagSet("comment", flag.ExitOnError)
	descriptionFile := fs.String("description-file", "", "Path to a file with light markup or raw ADF JSON")
	pos := parseFlags(fs, args)
	if len(pos) < 1 {
		return "", "", "", errors.New("missing ISSUE-KEY")
	}
	key = strings.ToUpper(pos[0])
	file = *descriptionFile
	if len(pos) > 1 {
		text = pos[1]
	}
	if text == "" && file == "" {
		return "", "", "", errors.New("no comment body: give text or --description-file")
	}
	if text != "" && file != "" {
		return "", "", "", errors.New("give either comment text or --description-file, not both")
	}
	if strings.HasPrefix(text, "--") {
		return "", "", "", fmt.Errorf("comment text %q looks like a flag", text)
	}
	return key, text, file, nil
}

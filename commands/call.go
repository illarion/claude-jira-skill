package commands

import (
	"encoding/json"
	"flag"
	"strings"
)

// Call makes an arbitrary authenticated API call without exposing credentials.
func Call(args []string) {
	fs := flag.NewFlagSet("call", flag.ExitOnError)
	dataFlag := fs.String("data", "", "JSON request body")
	pos := parseFlags(fs, args)
	if len(pos) < 2 {
		Fail("Usage: jira call METHOD /rest/api/3/... [--data '{...}']")
	}
	method, path := strings.ToUpper(pos[0]), pos[1]
	_, c := start()

	var body any
	if *dataFlag != "" {
		if err := json.Unmarshal([]byte(*dataFlag), &body); err != nil {
			Fail("Invalid --data JSON: %v", err)
		}
	}
	res, err := c.Request(method, path, body)
	if err != nil {
		die(err)
	}
	if res != nil {
		printJSON(res)
	}
}

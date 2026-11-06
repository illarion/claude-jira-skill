// jira is a self-contained CLI for the Claude Code Jira skill.
//
// It replaces the former Python scripts: every operation is a subcommand with
// the same flags, stdout JSON, stderr diagnostics, and exit codes.
package main

import (
	"fmt"
	"os"

	"github.com/illarion/claude-jira-skill/commands"
)

// version is injected at build time: -ldflags "-X main.version=v2.0.0".
var version = "dev"

const usageText = `Usage: jira <command> [args]

Commands:
  fetch ISSUE-KEY                                  fetch a ticket with comments and attachments
  search "JQL" [--fields summary,status] [--max N] search issues
  comment ISSUE-KEY "text" | --description-file F  add a comment
  create PROJECT "Summary" [--type Bug] [--priority P]
        [--description markup] [--description-file F] [--assignee "Name"] [--fixversion V]
                                                   create an issue
  update ISSUE-KEY [--summary S] [--priority P] [--description markup]
        [--description-file F] [--fixversion V] [--assignee "Name"]
                                                   update an issue
  transition ISSUE-KEY --list | "Status"           list or apply a status transition
  assign ISSUE-KEY "Display Name" | --unassign     assign or unassign
  link --list | ISSUE-1 "Type" ISSUE-2             list link types or link two issues
  call METHOD /rest/api/3/... [--data '{...}']     raw API call
  digest [statuses [P]] | [fields]                 workflow statuses / custom fields
  digest START_DATE [PROJECTS] [--json] [--output-file F]
        [--dev-statuses ...] [--review-statuses ...] [--test-statuses ...]
        [--tested-statuses ...] [--executor-field-id ID]
                                                   period digest report
  auth login [--url U] [--email E] [--token T] [--projects P] [--dir D]
        | logout                                   set up or remove credentials
  version                                          print the binary version
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usageText)
		os.Exit(1)
	}
	cmd, args := os.Args[1], os.Args[2:]
	switch cmd {
	case "fetch":
		commands.Fetch(args)
	case "search":
		commands.Search(args)
	case "comment":
		commands.Comment(args)
	case "create":
		commands.Create(args)
	case "update":
		commands.Update(args)
	case "transition":
		commands.Transition(args)
	case "assign":
		commands.Assign(args)
	case "link":
		commands.Link(args)
	case "call":
		commands.Call(args)
	case "auth":
		commands.Auth(args)
	case "digest":
		commands.Digest(args)
	case "version", "-v", "--version":
		fmt.Println(version)
	case "help", "-h", "--help":
		fmt.Print(usageText)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command %q\n\n", cmd)
		fmt.Fprint(os.Stderr, usageText)
		os.Exit(1)
	}
}

package commands

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"

	"github.com/illarion/claude-jira-skill/internal/config"
)

// Auth sets up or removes the local .jiraskillrc credentials file.
func Auth(args []string) {
	if len(args) == 0 {
		Fail("Usage: jira auth login|logout")
	}
	switch args[0] {
	case "login":
		authLogin(args[1:])
	case "logout":
		authLogout()
	default:
		Fail("Usage: jira auth login|logout")
	}
}

func authLogin(args []string) {
	fs := flag.NewFlagSet("auth login", flag.ExitOnError)
	name := fs.String("name", "", "Instance name (e.g. company-prod)")
	urlFlag := fs.String("url", "", "Jira URL (e.g. https://company.atlassian.net)")
	email := fs.String("email", "", "Account email")
	token := fs.String("token", "", "API token")
	projects := fs.String("projects", "", "Default projects for digest (comma-separated, optional)")
	dir := fs.String("dir", "", "Directory to place .jiraskillrc in (default: current)")
	fs.Parse(args)

	target := *dir
	if target == "" {
		wd, err := os.Getwd()
		if err != nil {
			die(err)
		}
		target = wd
	}
	dotfile := filepath.Join(target, config.DotFile)
	if _, err := os.Stat(dotfile); err == nil {
		Fail("%s already exists in %s. Remove it first to reconfigure.", config.DotFile, target)
	}

	nameV, urlV, emailV, tokenV, projectsV := *name, *urlFlag, *email, *token, *projects
	if urlV == "" || emailV == "" || tokenV == "" {
		reader := bufio.NewReader(os.Stdin)
		ask := func(label string) string {
			fmt.Print(label)
			line, _ := reader.ReadString('\n')
			return strings.TrimSpace(line)
		}
		if nameV == "" {
			nameV = ask("Instance name (e.g. company-prod): ")
		}
		if urlV == "" {
			urlV = ask("Jira URL (e.g. https://company.atlassian.net): ")
		}
		if emailV == "" {
			emailV = ask("Email: ")
		}
		if tokenV == "" {
			tokenV = askSecret("API token: ", ask)
		}
		if projectsV == "" {
			projectsV = ask("Default projects for digest (comma-separated, optional): ")
		}
	}
	if urlV == "" || emailV == "" || tokenV == "" {
		Fail("URL, email, and token are required.")
	}
	urlV = strings.TrimRight(urlV, "/")
	if nameV == "" {
		nameV = filepath.Base(target)
	}

	if err := config.Save(dotfile, &config.Config{
		Name:     nameV,
		URL:      urlV,
		Email:    emailV,
		Token:    tokenV,
		Projects: projectsV,
	}); err != nil {
		die(err)
	}
	fmt.Printf("Created %s in %s\n", config.DotFile, target)
	fmt.Printf("Add %s to your .gitignore.\n", config.DotFile)
}

// askSecret reads a line without echo when stdin is a terminal, like
// Python's getpass; otherwise it falls back to a plain prompt.
func askSecret(label string, ask func(string) string) string {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return ask(label)
	}
	fmt.Print(label)
	b, err := term.ReadPassword(fd)
	fmt.Println()
	if err != nil {
		die(err)
	}
	return strings.TrimSpace(string(b))
}

func authLogout() {
	wd, err := os.Getwd()
	if err != nil {
		die(err)
	}
	dotfile := filepath.Join(wd, config.DotFile)
	if _, err := os.Stat(dotfile); os.IsNotExist(err) {
		Fail("No %s in this directory.", config.DotFile)
	}
	if err := os.Remove(dotfile); err != nil {
		die(err)
	}
	fmt.Printf("Removed %s from %s\n", config.DotFile, wd)
}

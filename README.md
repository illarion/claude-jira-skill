# Jira Skill for Claude Code

A Claude Code plugin that provides full Jira integration — fetch tickets, search with JQL, manage issues, and generate team digest reports.

## Features

- **Ticket lookup** — fetch full ticket details including description, comments, and image attachments
- **JQL search** — search Jira with any JQL query
- **Create issues** — create new tickets with type, priority, assignee, and fix version
- **Update issues** — modify summary, description, priority, fix version, assignee
- **Assign / unassign** — assign or unassign tickets by display name
- **Change status** — transition tickets through workflow states, list available transitions
- **Link issues** — create relationships between tickets (blocks, relates to, etc.), list available link types
- **Add comments** — post comments on tickets
- **Team digest** — generate period reports with per-developer stats, workload analysis, and categorized summaries
- **Image support** — downloads and analyzes image attachments, opens in system viewer on request
- **Custom API calls** — access any Jira REST endpoint via `call_api.py` for advanced use cases

## Installation

In Claude Code, run:

```
/plugin marketplace add illarion/claude-jira-skill
/plugin install claude-jira-skill
```

See [INSTALL.md](INSTALL.md) for manual installation options.

## Prerequisites

- Python 3.8+
- A Jira Cloud or self-hosted Jira instance with API token ([create one here](https://id.atlassian.com/manage-profile/security/api-tokens))

## Setup

Start a Claude Code session and mention a Jira ticket (e.g. `PROJ-1234`). The skill will guide you through authentication via interactive prompts.

Auth is directory-based via `.jiraskillrc` files — each company directory tree uses its own Jira instance automatically. Add `.jiraskillrc` to your `.gitignore` to avoid committing credentials.

## Usage

**Fetch a ticket:**
> Show me PROJ-1234

**Search:**
> Find all open bugs assigned to me in PROJ

**Create an issue:**
> Create a bug in PROJ: "Login page crashes on empty password"

**Assign a ticket:**
> Assign PROJ-1234 to Alice Smith

**Unassign:**
> Unassign PROJ-1234

**Change status:**
> Move PROJ-1234 to In Review

**List available transitions:**
> What transitions are available for PROJ-1234?

**Link issues:**
> PROJ-1 blocks PROJ-2

**Add a comment:**
> Comment on PROJ-1234: "Fixed in latest commit"

**Team digest:**
> Give me a team digest for last week

## License

[MIT](LICENSE)

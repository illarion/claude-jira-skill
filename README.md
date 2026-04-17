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

You'll need a Jira API token. Create one at [id.atlassian.com/manage-profile/security/api-tokens](https://id.atlassian.com/manage-profile/security/api-tokens).

Then start a Claude Code session and type `/jira login`. The skill will prompt you for your Jira URL, email, and token. Credentials are stored locally in a `.jiraskillrc` file with restricted permissions.

### Multiple Jira instances

If you work with different Jira instances, place a separate `.jiraskillrc` (by performing a `/jira login` ) in each directory tree. The skill picks up the nearest one automatically — no manual switching needed.

```
~/work/
├── a/               ← .jiraskillrc (a.atlassian.net)
│   ├── repo-1/
│   └── repo-2/
└── b/               ← .jiraskillrc (b.atlassian.net)
    └── repo-3/
```
### Multiple projects

During login, you can configure default projects (comma-separated, e.g. `FRONTEND,API,MARKETING`). These are used by the team digest to query across all your projects at once.

You can work with any project your token has access to — just use the project key in your request (e.g. `/jira search open bugs in API2`).

## Usage

| Action | Example |
|--------|---------|
| Fetch a ticket | `/jira show me PROJ-1234` |
| Search | `/jira find all open bugs assigned to me in PROJ` |
| | `/jira what's overdue in PROJ?` |
| | `/jira show unassigned tickets in PROJ` |
| Create an issue | `/jira create a bug in PROJ: "Login page crashes on empty password"` |
| | `/jira create a high priority task in PROJ: "Update API docs", assign to Alice` |
| Update an issue | `/jira change the priority of PROJ-1234 to High` |
| | `/jira set fix version of PROJ-1234 to 1.5.0` |
| Add a comment | `/jira comment on PROJ-1234: "Fixed in latest commit"` |
| Assign / unassign | `/jira assign PROJ-1234 to Alice Smith` |
| | `/jira unassign PROJ-1234` |
| Change status | `/jira move PROJ-1234 to In Review` |
| | `/jira what transitions are available for PROJ-1234?` |
| Link issues | `/jira PROJ-1 blocks PROJ-2` |
| | `/jira what link types are available?` |
| Images | `/jira show me the screenshots from PROJ-1234` |
| | `/jira open screenshot 2` |
| Team digest | `/jira give me a team digest for last week` |
| | `/jira what did the team do this week?` |

## Why this over the official Jira MCP?

The [Atlassian Rovo MCP server](https://www.atlassian.com/platform/remote-mcp-server) gives you basic Jira CRUD — search, create, update. This skill does that and more:

| | Official MCP | This skill |
|--|:--:|:--:|
| Search, create, update issues | Yes | Yes |
| Transitions, assignments, linking | No | Yes |
| Team digest reports | No | Yes |
| Developer attribution from changelog | No | Yes |
| Image attachment analysis | No | Yes |
| Self-hosted Jira support | No | Yes |
| Multi-instance auto-switching | No | Yes |
| ADF formatting handled for you | No | Yes |
| Requires admin/OAuth setup | Yes | No |
| Rate limited by Atlassian plan | Yes | No |

The official MCP is a cloud proxy that requires Atlassian admin to enable Rovo, OAuth 2.1 configuration, and optionally IP allowlisting. This skill runs locally, needs only a personal API token, and works in seconds.

**When to use the official MCP instead:** if you need Confluence or Compass access (not just Jira), your org requires OAuth/SSO and doesn't allow personal API tokens, or corporate policy mandates all API access through Atlassian's managed infrastructure.

## License

[MIT](LICENSE)

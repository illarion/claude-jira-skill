# Authentication setup

Credentials are stored in a `.jiraskillrc` file placed at the root directory of a company/project tree. The `jira` binary auto-detects credentials by traversing up from the current working directory until it finds this file. This means each company directory tree uses its own Jira instance automatically — no manual switching needed.

**Interactive setup via AskUserQuestion:**

When no `.jiraskillrc` is found (auth failure or first use), collect credentials using `AskUserQuestion` and then run the login command non-interactively. Use a single `AskUserQuestion` call with 3 questions:

1. **Jira URL** (header: `Jira URL`) — "What is your Jira instance URL?"
   - Option 1: `https://company.atlassian.net` — "Atlassian Cloud (most common)"
   - Option 2: `https://jira.company.com` — "Self-hosted Jira instance"
   - The user will select "Other" and type their actual URL.

2. **Email** (header: `Email`) — "What email is associated with your Jira account?"
   - Option 1: `user@company.com` — "Work email"
   - Option 2: `user@gmail.com` — "Personal email"
   - The user will select "Other" and type their actual email.

3. **API Token** (header: `API Token`) — "What is your Jira API token? Generate one at id.atlassian.com/manage-profile/security/api-tokens"
   - Option 1: `I have a token` — "Paste your existing API token via Other"
   - Option 2: `I need to create one` — "Visit the link above in your browser first"
   - The user will select "Other" and paste their token.

After collecting answers, optionally ask a follow-up `AskUserQuestion` for default projects (header: `Projects`) — "Which Jira projects should be included in digest reports? (comma-separated, e.g. PROJ,CORE)" with options like "Skip for now" / "Enter projects". Then run:

```
jira auth login --url URL --email EMAIL --token TOKEN --dir /path/to/company [--projects PROJECTS]
```

Use `--dir` pointing to the working directory root (typically the company root directory). The `--dir` flag determines where `.jiraskillrc` is placed.

**Manual setup:**

The user can also run the login command directly in their terminal:
```
jira auth login                          # interactive prompts
jira auth login --url URL --email EMAIL --token TOKEN --dir /path/to/company  # non-interactive
jira auth logout                         # removes .jiraskillrc from current directory
```

The binary prints the detected instance name and URL to stderr automatically. Mention it in your response so the user knows which instance is being queried.

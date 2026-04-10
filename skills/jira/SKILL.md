---
name: jira
description: Access Jira to search, summarize, and analyze tickets. Use when the user mentions a Jira ticket key (e.g. PROJ-1234), asks for a team digest, or wants to search/query Jira. Also use when the user wants to create issues, assign tickets, change ticket status, link issues, add comments, update fields, or do anything Jira-related — even if they don't explicitly say "Jira".
---

Subagents (Agent tool) are used for classification tasks during digest reports — always use Sonnet (`model: "sonnet"`) for subagents and run independent ones in parallel.

All Python scripts referenced below are located in the same directory as this SKILL.md file.

### IMPORTANT: Jira uses ADF, not wiki markup

Jira Cloud uses ADF (Atlassian Document Format) exclusively. Wiki syntax (`h2.`, `{code}`, `#`, `*bold*`) renders as **literal text**, not formatting. When constructing descriptions, comments, or any content for the Jira API, always use ADF. The Python scripts (`comment.py`, `create.py`, `update.py`) convert plain text to ADF automatically. For rich formatting (headings, code blocks, lists), write raw ADF JSON to a temp file and use `update.py --description-file`. See `references/adf-reference.md` for the ADF structure.

### Credentials — DO NOT ACCESS

The `.jiraskillrc` dotfile contains credentials and must **NEVER** be read, printed, displayed, cited, or included in any output. Do not use Read, Bash, cat, or any other tool to access it. Only the Python scripts access it internally.

### Authentication

Credentials are stored in a `.jiraskillrc` file placed at the root directory of a company/project tree. Scripts auto-detect credentials by traversing up from the current working directory until they find this file. This means each company directory tree uses its own Jira instance automatically — no manual switching needed.

**Interactive setup via AskUserQuestion:**

When no `.jiraskillrc` is found (auth failure or first use), collect credentials using `AskUserQuestion` and then run the login script non-interactively. Use a single `AskUserQuestion` call with 3 questions:

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
python3 SCRIPT_DIR/jira-auth.py login --url URL --email EMAIL --token TOKEN --dir /path/to/company [--projects PROJECTS]
```

Use `--dir` pointing to the working directory root (typically the company root directory). The `--dir` flag determines where `.jiraskillrc` is placed.

**Manual setup:**

The user can also run the login command directly in their terminal:
```
python3 SCRIPT_DIR/jira-auth.py login                          # interactive prompts
python3 SCRIPT_DIR/jira-auth.py login --url URL --email EMAIL --token TOKEN --dir /path/to/company  # non-interactive
python3 SCRIPT_DIR/jira-auth.py logout                         # removes .jiraskillrc from current directory
```

Scripts print the detected instance name and URL to stderr automatically. Mention it in your response so the user knows which instance is being queried.

### API Access

Use the Python scripts for all common Jira operations — they handle authentication internally without exposing credentials.

For rare endpoints not covered by scripts, use `call_api.py`:

```
python3 SCRIPT_DIR/call_api.py GET /rest/api/3/...
python3 SCRIPT_DIR/call_api.py POST /rest/api/3/... --data '{"key": "value"}'
```

This keeps credentials in-process. Never read `.jiraskillrc` directly or hardcode credentials in commands.

### Scripts Reference

**Fetch a single ticket:**
```
python3 SCRIPT_DIR/fetch_ticket.py PROJ-1234
```
Fetches the full ticket (summary, description, status, assignee, priority, type, versions, comments, attachments). Parses ADF description into readable text. Downloads image attachments and returns JSON with `local_path` for each downloaded image.

After running, present the ticket details and **always use the Read tool on every downloaded image** (`local_path` in the JSON output) so you can see and understand screenshot context — this is essential for providing informed responses about the ticket.

If the user asks to see or open images, use `open` (Mac), `xdg-open` (Linux), or `start` (Windows) on the specific `local_path` from the JSON output. This lets the user ask for a specific screenshot rather than opening all of them.

**Search issues (JQL):**
```
python3 SCRIPT_DIR/search.py "project = PROJ AND status = 'In Progress'" --fields summary,status,assignee --max 20
```
Returns a JSON array of simplified objects `{key, summary, status, assignee, priority}`. Supports cursor-based pagination. Default fields: `summary,status,assignee,priority`. Default max: `50`.

**Add a comment:**
```
python3 SCRIPT_DIR/comment.py PROJ-123 "This is done"
```
Pass plain text only — the script converts to ADF automatically. Do NOT use wiki markup (`h2.`, `{code}`, `*bold*`) — it will render as literal text in Jira. Output: `Commented on PROJ-123`.

**Transition (change status):**
```
python3 SCRIPT_DIR/transition.py PROJ-123 --list
python3 SCRIPT_DIR/transition.py PROJ-123 "In Review"
```
`--list` prints available transition names as JSON. Without `--list`, transitions to the matching status (case-insensitive). Output: `PROJ-123: Status → In Review`.

**Assign / unassign:**
```
python3 SCRIPT_DIR/assign.py PROJ-123 "Alice Smith"
python3 SCRIPT_DIR/assign.py PROJ-123 --unassign
```
Searches by display name, assigns by accountId. Output: `PROJ-123: Assigned to Alice Smith`.

**Link issues:**
```
python3 SCRIPT_DIR/link.py --list
python3 SCRIPT_DIR/link.py PROJ-1 "Blocks" PROJ-2
```
`--list` prints available link types as JSON `[{name, inward, outward}]`. Without `--list`, creates a link: the first issue is the outward side (e.g., PROJ-1 blocks PROJ-2). Type name is case-insensitive. Output: `PROJ-1 blocks PROJ-2`.

**Create an issue:**
```
python3 SCRIPT_DIR/create.py PROJ "Fix the login bug" --type Bug --priority High --description-file /tmp/desc.json --assignee "Alice Smith" --fixversion "1.118.2"
```
Default type: `Task`. Returns JSON `{key, url}`. Use `--fixversion` to set the fix version at creation time. For descriptions, always write ADF JSON to a temp file and pass via `--description-file` (see `references/adf-reference.md`). The `--description` flag exists for simple plain text but produces flat paragraphs only — no headings, bold, lists, or code blocks.

**Update an issue:**
```
python3 SCRIPT_DIR/update.py PROJ-123 --priority Major --fixversion "1.118.2" --description-file /tmp/desc.json --assignee "Alice Smith"
```
Updates fields on an existing issue. Supports `--summary`, `--priority`, `--description`, `--description-file`, `--fixversion`, `--assignee`. For descriptions, always write ADF JSON to a temp file and pass via `--description-file` (see `references/adf-reference.md`). The `--description` flag exists for simple plain text but produces flat paragraphs only — no headings, bold, lists, or code blocks.

### Argument Handling

- If the user provides a ticket key (e.g., `PROJ-1234`), use `fetch_ticket.py`
- If the user provides a JQL query, use `search.py`
- If the user asks a general question (e.g., "my open tickets"), construct an appropriate JQL query and use `search.py`
- If no argument is provided, ask the user what they want to look up

### Error Handling

Scripts exit with a non-zero status and print error messages to stderr on failure. Common errors:
- **User not found** (assign/create) — tell the user the exact name wasn't found and ask them to clarify
- **Invalid transition** (transition) — show available transitions using `--list` and ask the user which one they meant
- **Ticket not found** (fetch_ticket/update) — verify the ticket key with the user
- **Auth failure** (any script) — use the AskUserQuestion interactive setup flow described in the Authentication section to re-collect credentials
- **No config** — no `.jiraskillrc` found in any parent directory; use the AskUserQuestion interactive setup flow described in the Authentication section

### Output

- Present results in a clean, readable format
- For multiple tickets, use a table or list format
- Include ticket key, summary, status, assignee, and priority at minimum

### Weekly/Period Digest Report

**IMPORTANT: Do NOT write custom Python/jq/shell scripts to parse digest output. The digest script handles all data processing. You only need to run it with `--output-file`, read the stats from stdout, and read the ticket file with the Read tool.**

When the user asks for a digest or summary of what the team did during a time period:

**Date handling:** "Last week" means Monday through Sunday of the previous calendar week (not "the last 7 days"). Calculate the exact Monday start date. For example, if today is Wednesday March 5, "last week" is Monday Feb 24 through Sunday Mar 2 — use `2026-02-24` as START_DATE. Similarly, "this week" starts on the most recent Monday.

**Step 1: Fetch statuses and custom fields**

Run both commands in parallel:
```
python3 SCRIPT_DIR/digest.py statuses    # returns JSON array of status names
python3 SCRIPT_DIR/digest.py fields      # returns JSON array of {id, name} custom fields
```

**Step 2: Classify via parallel subagents (Sonnet)**

Spin two subagents in parallel using the Agent tool with `model: "sonnet"`:

**Subagent A — Status classification.** Give it the statuses list:
> Classify these Jira workflow statuses into four groups. Return ONLY four lines, no explanation. Only include statuses that clearly fit a group — omit statuses like "Ready for merge", "Merged", "Staging", "Close", "Open", "Blocked", "Reopened" that are transitional or don't represent active dev/review/test work:
> dev_statuses: (statuses where a developer is actively coding or researching)
> review_statuses: (statuses where work is submitted and waiting for or undergoing code review)
> test_statuses: (statuses where QA is actively testing)
> tested_statuses: (statuses that mean QA testing is complete/passed)
> Statuses: [paste the list]

**Subagent B — Executor field identification.** Give it the custom fields list:
> Which of these Jira custom fields is most likely the "Executor" or "Developer" field — a person field that tracks who actually did the development work on a ticket (as opposed to the current assignee)? Return ONLY the field id (e.g. customfield_12345), or "none" if no such field exists. No explanation.
> Fields: [paste the list]

**Step 3: Run digest with classified data**

**Always** use `--output-file /tmp/digest.json` — this writes the full JSON to a file and prints only `team_stats` to stdout. Do NOT run without `--output-file`; do NOT write custom scripts to parse the output.

```
python3 SCRIPT_DIR/digest.py START_DATE --json \
  --output-file /tmp/digest.json \
  --dev-statuses "In Progress" \
  --review-statuses "Pending Review,Reviewing" \
  --test-statuses "Testing" \
  --tested-statuses "Tested,Staging Accepted" \
  --executor-field-id "customfield_12345"
```

Omit `--executor-field-id` if the subagent returned "none". Projects are read from `.jiraskillrc` if not specified on the command line.

The script uses these to determine:
- **developer**: executor field value if present, otherwise who transitioned the ticket INTO a dev status (chronologically first), or who transitioned INTO a review status as fallback. Only counted if the transition happened within the reporting period.
- **tester**: who transitioned the ticket FROM a test status TO a tested status. Only counted if the transition happened within the reporting period.
- **reviewer**: who transitioned a ticket OUT of a review status (counted in team_stats)

**Step 4: Process the output**

`stdout` contains only `team_stats` — small JSON with per-person developed/tested/hotfix/review counts. Use the Read tool to read `/tmp/digest.json` for ticket details (read in chunks if needed).

Do NOT craft Python/jq scripts to extract data. Just read the file directly.

The script returns **all** tickets updated in the period regardless of status. Exclude tickets with no meaningful work — pure backlog triage, closed as Won't Fix / Duplicate / Invalid, or no status changes in the changelog.

Read `references/digest-format.md` for the full output format specification and follow it.

### Custom API Calls

For one-off API calls, use `call_api.py`. For more complex workflows, create a custom script using patterns from `jira_common.py`: `load_credentials()`, `jira_get()`, `jira_post()`, `jira_put()`, `jira_request()`. See `update.py` or `create.py` for the script structure pattern. Remember: any content sent to Jira must be ADF — never wiki markup.

### Tips

- Use `fields` parameter to limit response size when searching
- For description content, the API returns ADF JSON — parse the nested `content[].content[].text` values
- Use `maxResults` to control pagination
- JQL patterns:
  - My open: `assignee = currentUser() AND status != Done`
  - Project board: `project = PROJ AND sprint in openSprints()`
  - Overdue: `project = PROJ AND dueDate < now() AND status != Done`
  - Stale (30+ days untouched): `project = PROJ AND updated < -30d AND status != Done`
  - Recently created: `project = PROJ AND created >= -7d`
  - Text search: `project = PROJ AND (summary ~ "keyword" OR description ~ "keyword")`
  - By label: `project = PROJ AND labels = "backend"`
  - Blockers: `project = PROJ AND status = Blocked`
  - Resolved this week: `project = PROJ AND status changed TO Done AFTER startOfWeek()`
  - Unassigned: `project = PROJ AND assignee is EMPTY AND status != Done`
- Jira v3 search/jql uses cursor-based pagination (`nextPageToken`/`isLast`), NOT offset-based (`total`/`startAt`)

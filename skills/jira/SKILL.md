---
name: jira
description: Access Jira to search, summarize, and analyze tickets. Use when the user mentions a Jira ticket key (e.g. PROJ-1234), asks for a team digest, or wants to search/query Jira. Also use when the user wants to create issues, assign tickets, change ticket status, link issues, add comments, update fields, or do anything Jira-related — even if they don't explicitly say "Jira". Also use for "file a bug", "write up a ticket", "open a follow-up", "hand this to QA", "reply to QA".
---

All commands run through one prebuilt binary, `jira` — no Python or other runtime is needed. The binary prints the detected Jira instance to stderr; mention it in the reply.

### Binary

`jira` is cached at `~/.claude/jira-skill/bin/jira` (`jira.exe` on Windows). If it is missing or `<bin> version` fails, download it once:

- Linux/macOS: get OS and arch from `uname -s` and `uname -m` (`Linux`→`linux`, `Darwin`→`darwin`, `x86_64`→`amd64`, `aarch64`/`arm64`→`arm64`), then
  `curl -Lf --create-dirs -o ~/.claude/jira-skill/bin/jira https://github.com/illarion/claude-jira-skill/releases/latest/download/jira-<os>-<arch> && chmod +x ~/.claude/jira-skill/bin/jira`
- Windows: arch is `amd64` or `arm64` (from `$env:PROCESSOR_ARCHITECTURE`); in PowerShell, create `$HOME\.claude\jira-skill\bin` with `New-Item -ItemType Directory -Force`, then
  `Invoke-WebRequest https://github.com/illarion/claude-jira-skill/releases/latest/download/jira-windows-<arch>.exe -OutFile $HOME\.claude\jira-skill\bin\jira.exe`

Then run `<bin> version` to confirm. If the download fails, tell the user to download the file manually from the Releases page of `illarion/claude-jira-skill` and place it at that path. Below, `jira` means the full path to this binary.

### Credentials

Credentials live in `.jiraskillrc` at the root of a company tree; the binary finds it by walking up from the working directory. Never read, print or cite that file with any tool; only `jira` opens it. On "no config" or an auth failure, read `references/auth-setup.md` and follow it.

### Jira uses ADF, not wiki markup

Wiki syntax (`h2.`, `{code}`, `*bold*`) renders as literal text. `jira comment`, `jira create` and `jira update` convert light markup to ADF: `###` headings, `-` and `1.` lists, code fences, `**bold**`, `` `code` ``, issue keys become links. Raw ADF JSON is accepted by `--description-file` for tables and panels; see `references/adf-reference.md`.

### Writing tickets and comments

Before `jira create`, `jira comment` or `jira update` with text, read `references/writing-style.md` and copy the form of its closest sample. Write the body to `/tmp/desc.md` and pass `--description-file`. Show the draft in chat, post in the same turn unless the user objects, report key and URL. Comments: 5 lines or fewer. A scope change in a comment also updates the description.

### Commands

- Fetch: `jira fetch PROJ-1234`. Returns summary, description as text, status, assignee, type, versions, comments, attachments; image attachments are downloaded and listed with `local_path`. Present the ticket, then always use the Read tool on every downloaded image: screenshots carry context the text does not. To open one for the user: `xdg-open <local_path>` (Linux), `open` (Mac).
- Search: `jira search "<JQL>" --fields summary,status,assignee --max 20`. JSON array `{key, summary, status, assignee, priority}`, default max 50. Common JQL in `references/jql-tips.md`.
- Comment: `jira comment PROJ-123 --description-file /tmp/desc.md` or `jira comment PROJ-123 "text"`. Exactly one of the two.
- Transition: `jira transition PROJ-123 --list`, then `jira transition PROJ-123 "In Review"` (case-insensitive).
- Assign: `jira assign PROJ-123 "Alice Smith"` or `--unassign`.
- Link: `jira link --list`; `jira link PROJ-1 "Blocks" PROJ-2` (first key is the outward side).
- Create: `jira create PROJ "Summary" --type Bug --description-file /tmp/desc.md [--priority P] [--assignee "Name"] [--fixversion "1.2"]`. Default type Task. Returns `{key, url}`. A summary over 70 chars prints a warning.
- Update: `jira update PROJ-123 [--summary ...] [--description-file ...] [--priority ...] [--fixversion ...] [--assignee ...]`.
- Anything else: `jira call GET|POST|PUT|DELETE /rest/api/3/... [--data '{...}']`. Jira v3 search paginates with `nextPageToken`/`isLast`, not `startAt`.

### Routing

`login` (as in `/jira login`) → read `references/auth-setup.md` and run the setup flow. Ticket key → fetch. JQL, or a question like "my open tickets" → search. No argument → ask what to look up. Errors arrive on stderr with a non-zero exit: user not found → ask for the exact name; invalid transition → show `--list`; ticket not found → confirm the key; auth failure or no config → `references/auth-setup.md`.

For several tickets, use a table with key, summary, status, assignee.

### Digest / period summary

When asked what the team did over a period, read `references/digest.md` and follow it. Do not parse digest output with your own scripts.

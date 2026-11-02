# JQL tips

Use `jira search` for all of these: `--fields` limits the returned fields, `--max` limits the results. The command handles cursor pagination (`nextPageToken`/`isLast`); do not page by `startAt`. Ticket descriptions come back as plain text from `jira fetch`, no ADF parsing needed.

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

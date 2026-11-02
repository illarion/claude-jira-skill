# Digest / period summary

"Last week" means Monday through Sunday of the previous calendar week, not the last 7 days. "This week" starts on the most recent Monday. Compute the exact Monday and use it as START_DATE (for example `2026-02-24`).

Do not write scripts to parse digest output. `jira digest` does all data processing.

## Step 0: reuse a saved classification

If `<company root>/.jira-digest.json` exists (next to `.jiraskillrc`) and its `saved_at` is less than 8 weeks old, take the statuses and `executor_field` from it and go to Step 3. Otherwise, or when the report looks off (a status in the output that no group covers, zero developers or testers for an active week), run Steps 1–2 again and overwrite the file.

```json
{
  "saved_at": "2026-11-02",
  "dev_statuses": ["In Progress"],
  "review_statuses": ["Tech Review", "Reviewed"],
  "test_statuses": ["QA Test"],
  "tested_statuses": ["Ready for Stage", "Ready for Prod"],
  "executor_field": "customfield_10234"
}
```

`executor_field` is a field id or `null`.

## Step 1: fetch statuses and custom fields (run both in parallel)

```
jira digest statuses    # JSON array of status names
jira digest fields      # JSON array of {id, name} custom fields
```

## Step 2: classify with one Haiku subagent

One Agent call, `model: "haiku"`, both lists in one prompt:

> Return exactly five lines, no explanation:
> dev_statuses: statuses where a developer actively codes or researches
> review_statuses: work submitted, waiting for or undergoing code review
> test_statuses: QA actively testing
> tested_statuses: QA testing complete / passed
> executor_field: the custom field id (e.g. customfield_12345) that tracks who did the development work, as opposed to the current assignee, or "none"
> Only include statuses that clearly fit a group. Omit transitional statuses such as "Ready for merge", "Merged", "Staging", "Close", "Open", "Blocked", "Reopened".
> Statuses: [paste the list]
> Fields: [paste the list]

Save the five answers to `<company root>/.jira-digest.json` in the shape shown in Step 0, with today's date in `saved_at` and `null` for a missing executor field.

## Step 3: run the digest

Always pass `--output-file /tmp/digest.json`: the full JSON goes to the file, only `team_stats` to stdout.

```
jira digest START_DATE --json \
  --output-file /tmp/digest.json \
  --dev-statuses "In Progress" \
  --review-statuses "Pending Review,Reviewing" \
  --test-statuses "Testing" \
  --tested-statuses "Tested,Staging Accepted" \
  --executor-field-id "customfield_12345"
```

Omit `--executor-field-id` when there is no executor field. Never pass the literal string `none` or `null`: `jira digest` would send it to Jira as a field name and the request fails. Projects come from `.jiraskillrc` unless given on the command line.

`jira digest` derives:
- developer: executor field value if present, otherwise who transitioned the ticket INTO a dev status (chronologically first), or INTO a review status as fallback. Counted only when the transition is inside the period.
- tester: who transitioned the ticket FROM a test status TO a tested status, inside the period.
- reviewer: who transitioned a ticket OUT of a review status (team_stats only).

## Step 4: read and write the report

stdout holds `team_stats` only (per-person developed/tested/hotfix/review counts). Read `/tmp/digest.json` with the Read tool, in chunks if needed.

`jira digest` returns every ticket updated in the period. Exclude tickets with no meaningful work: pure backlog triage, closed as Won't Fix / Duplicate / Invalid, or no status changes in the changelog.

Format the report per `references/digest-format.md`.

## Digest Output Format

**Audience:** Technical team leads at a weekly standup. They're technical but care about the big picture. You're reporting on behalf of the team.

**Structure the report as follows:**

**1. Observations** — 3-5 bullet points covering:
- Patterns: multiple tickets in the same area suggest a trend
- Risks: tickets stuck in review, long-running items, single-person bottlenecks
- Connections: related tickets that form a larger effort
- Keep it factual and brief — no fluff, no speculation beyond what the data shows

**2. Team Performance** — Use `team_stats` from the JSON output. It provides pre-computed per-person ticket counts broken down by developed/tested/hotfixes/reviews.

For each team member, note:
- Developed count, tested count, hotfix count, review count (from `team_stats.per_person`)
- Brief summary of what they worked on (use `dev_keys` and `test_keys` to look up ticket summaries)
- Label as overperformer / on track / underperformer

Then add 1-2 actionable recommendations: redistribute load, pair up on a bottleneck, flag someone who may be blocked or idle. Be direct but constructive.

If a team member has zero tickets in the period, explicitly call that out — don't silently omit them.

**3. Summary** — After the full report, add a `## Summary` heading with a categorized bullet-point summary for someone who won't read the details. No ticket keys. Each category is a bold top-level bullet, with individual items as sub-bullets beneath it. Each sub-bullet should be a short plain-English statement.

Categories (use bold labels, skip empty categories):
- **Released** — what shipped to production this period
- **Staging** — what's deployed to staging and awaiting acceptance
- **Done** — tested/reviewed but not yet staged
- **In Progress** — actively being worked on
- **Waiting** — blocked, stalled, or waiting on external dependencies

Include dates where known (from fix versions, changelog timestamps, or status transition dates).

Keep this section **concise** — group related tickets into single bullets, summarize themes rather than listing every ticket. Aim for 2-4 sub-bullets per category max. The audience wants the big picture, not a ticket-by-ticket list without numbers.

Example:
```
## Summary

- **Released**
  - Server v1.2.3 (Feb 12) — user client connectivity and stability fixes,  sync, documents search improvements
  - v1.2.3, v1.2.4 hotfixes (Feb 26-Mar 5) — crash fixes for stats, scheduler, and monitoring
- **Staging** (v1.3.0)
  - Async/await migration and refactoring
  - Connection reliability and status update fixes
  - new API
- **In Progress**
  - Frontend upgrade, cleanup
```

This section is strictly factual — no recommendations, no opinions, no action items. Those belong in the Team Performance section only.

**Do NOT:**
- Copy Jira titles verbatim — rewrite them into concise, clear language
- Add filler or transition text
- Speculate beyond what the ticket data shows
- Include per-ticket themed lists (the audience doesn't need them)
- Include automated threshold warnings
- Put recommendations or opinions in the Summary section

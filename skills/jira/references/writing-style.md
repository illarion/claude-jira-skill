# Writing tickets and comments

Most tickets end up in the **Test** column, read by **black-box QA**. QA has only what a user has: the product, a browser, a device. No repo, no logs, no context. Every visible line of text has to be something a tester can act on.

Good human-written tickets look like this:

```
Position menu text is cut off
The P in Top Left gets cut off by the region below. Make sure all text has
enough padding so that it doesn't get obstructed by other UI elements.
```
```
Text should not be cut off anymore, fixed in v2.5.334
```

AI-written tickets tend to run 400-1,500 words, with sections like *Where the code is*, *Diagnostic recipes* and *Dead ends*. QA cannot find the test in them. Do not write those.

## 1. Decide who reads it

| Ticket goes to | Examples | Needs *How to test*? |
|---|---|---|
| **QA** (default) | anything a user can see, hear, or configure; any behaviour change | **Yes** |
| **Dev-only todo** | CI, build system, test harness, tooling, refactor with no behaviour change | No |

If you are not sure, it goes to QA.

## 2. Budgets

| Part | Limit |
|---|---|
| Summary | ≤ 70 chars, product language, the **symptom** not the cause (dev-only todos: plain language) |
| Visible description | ≤ ~120 words |
| Dev notes (collapsed) | ≤ ~15 lines |
| Comment | ≤ 5 lines |

Over budget usually means one of these:
- the ticket covers two problems: split it;
- the text is investigation notes: they go in the PR;
- the text is an open question: it gets its own ticket.

Summary, bad and good:

| ✗ | ✓ |
|---|---|
| `Storefront: Record still refused -- index.php:530/535 function…` | `Storefront: Record button does nothing` |
| `Inventory service publishes a stale order when a database dies - the payment service's gate…` | `Previous order is silently published when database is offline` |

## 3. Templates

Write the body in light markup (see `adf-reference.md`), save it to `/tmp/desc.md`, and pass `--description-file /tmp/desc.md`. The headings come in a fixed order. **Leave a heading out when you have nothing for it.** Never write "N/A".

### Bug

```
<One sentence: what the user sees go wrong.>

**Seen on:** <version or build>          (or: "Not yet reproduced, found by code reading.")

### Steps
1. …
2. …

**Expected:** <one line>
**Actual:** <one line>

▸ Dev notes
- …
```

### Task / Improvement / New Feature (QA-bound)

```
<One sentence: what changes for the user, and why.>

### Done when
- <checkable outcome>        (2-5 bullets)

### How to test
1. <action>
2. <action>

**PASS:** <what they see>
**FAIL:** <what the old behaviour looks like>

### Out of scope                (optional, ≤ 3 bullets, each pointing to a linked ticket)
- PROJ-1234 …

▸ Dev notes
- …
```

### Dev-only todo

```
<One sentence: what needs doing.>

### Done when
- …

▸ Dev notes
- …
```

**How to test rules**
- Write numbered actions using the names shown in the product UI (e.g. *Menu > Orders > History*).
- Name the precondition: which product, which account type, which country, and so on.
- Give one PASS line and one FAIL line. When the old behaviour is recognisable, FAIL describes it.
- A shell command belongs here **only** if QA runs it exactly as written, and it is one line.
- If QA would hit an **expected** behaviour and think it is a bug (e.g. "a recording now splits into `_2` files"), add one bullet on it. Do not add a whole *Traps* / *Not bugs* section.

## 4. Dev notes: the only place for engineering detail

Dev notes is a collapsed block at the bottom of the description (the `▸ Dev notes` line in the markup), so QA never has to scroll past it.

| Allowed | Not allowed, put in the PR instead |
|---|---|
| repo + file names (no line numbers, they go stale) | code snippets |
| suspected cause, 1-3 sentences | shell recipes, probe scripts |
| candidate fix, 1-2 sentences, plus the one thing to confirm | evidence logs, measurement tables |
| PR link or branch | dead ends, what you tried |
| | server IPs |

If a developer really needs the full investigation, attach it as a `.md` file.

## 5. Comments

One comment per event. There are only four kinds:

| Kind | Shape |
|---|---|
| **Hand-off to QA** | What they will see, plus `Test on <version or build> or later`. |
| **Reply to a QA finding** | Acknowledge it, give the cause in product terms in one line, then the version to retest on. *"Version 2.5.333 Orders worked only locally; fixed so it works in dev/stage too. Please retest on v2.5.334 or later."* |
| **Needs info** | The one question, plus what you need from them (version, video evidence). |
| **Scope change** | One line. **Also edit the description** so it matches. |

**Do not post:**
- progress diaries ("investigated X, then Y…"). The Status field already shows progress;
- "Correcting my previous comment…". **Edit the original** comment or description instead, so no stale result or wrong PR link is left behind to mislead QA;
- a second copy of anything that is in the description.

## 6. Never put these in visible text

- file paths, line numbers, function, variable or field names
- code snippets, commit SHAs, branch names, PR numbers (a PR *link* in Dev notes is fine)
- server internals, state machines
- root-cause essays, dead ends, "what I tried"
- a `Status:` header in the body. That is what the Status field is for.
- server IPs and serials used on your own bench
- open questions. **File them as their own tickets**, because a question buried in a ticket never gets answered.

## 7. Before you post

Show the draft (summary + body) to the user, then check:
1. Is the summary ≤ 70 chars and in product language?
2. Is the visible text ≤ ~120 words, and could a tester act on every line?
3. For a QA-bound ticket, is there a *How to test* with PASS/FAIL?
4. Is every ticket you mention written as a key (`PROJ-123`)? The scripts turn keys into links.

Post in the same turn unless the user objects. Report the key and URL.

## 8. Formatting

- Section labels are `###` (H3). Do not use H2, it makes a short ticket look like a long document.
- Use real lists (`-` and `1.`). Do not type a `-` at the start of a paragraph.
- Only a tester-run command goes in a code fence.
- Labels like **Seen on**, **Expected**, **Actual**, **PASS**, **FAIL** are bold, not headings.

## Worked example

**Before.** The ticket had 9 H2 sections: *Root cause · Why it matters · Evidence · Impact · Where the code is (6 file:line anchors) · Proposed fix · Diagnostic recipes (6 shell scripts) · Dead ends · Device state*. About 1,470 words. The summary was `order-manager-server forks ~100 db probes/sec at idle: polling not existing local db that do not exist`.

**After.** 75 words.

````
Summary: Idle order manager spawns ~100 processes/sec on production

Server constantly spawns background processes at idle, wasting CPU.

**Seen on:** Backend, develop build

### Steps
1. Boot a server with no local db configured (any dev server or prod) and leave it idle for 5 seconds.
2. Run:

```
a=$(awk '/^processes/{print $2}' /proc/stat); sleep 10; b=$(awk '/^processes/{print $2}' /proc/stat); echo $(( (b-a)/10 ))
```

**Expected:** under 5
**Actual:** 37

▸ Dev notes
- order-manager-service, index.php: start() opens a db connection for every declared pool entry with a hardcoded localhost address, in a forked process
- Candidate fix: do not hardcode localhost when another address is configured
````

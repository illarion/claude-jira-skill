## ADF (Atlassian Document Format) Reference

Jira Cloud uses ADF, not wiki markup. Wiki syntax (`h2.`, `{code}`, `#`) will render as literal text.

The `comment.py`, `create.py` and `update.py` scripts convert **light markup** to ADF automatically. Write the markup to a file (for example `/tmp/desc.md`) and pass it with `--description-file`. Raw ADF JSON is the fallback for nodes the markup does not cover (tables, panels, mentions).

### Light markup

| Line | ADF |
|---|---|
| `# `, `## `, `### ` … | `heading` level 1-6 (use `###` in tickets) |
| `- item` or `* item` (consecutive lines) | `bulletList` |
| `1. item`, `2. item` … (consecutive lines) | `orderedList` |
| ```` ``` ```` … ```` ``` ```` | `codeBlock` (language from the fence, e.g. ```` ```bash ````) |
| blank line | block separator, not emitted |
| anything else | `paragraph` |

Inline, inside paragraphs, list items and headings:

| Pattern | Mark |
|---|---|
| `**text**` | `strong` |
| `` `text` `` | `code` |
| `https://…` | `link` |
| `PROJ-123` | `link` to the issue on the current Jira instance |

Leading whitespace is ignored, so indented template lines are fine. Lists are flat (one level).

Example:

```
v2.5.333
The Record button on the storefront does nothing when clicked. Probably related to PROJ-1201.

Steps:
1. Open Menu > Orders > History
2. Click "Record"

Expected: a new recording starts
Actual: nothing happens
```

### Raw ADF

`--description-file` accepts raw ADF when the file is a JSON object with `"type": "doc"`. The file holds the **document only**, not a `{"fields": ...}` wrapper.

```json
{
  "version": 1, "type": "doc",
  "content": [
    { "type": "heading", "attrs": {"level": 3}, "content": [{"type": "text", "text": "Steps"}] },
    { "type": "orderedList", "content": [
        {"type": "listItem", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "Open Menu > Orders"}]}]}
    ]},
    { "type": "paragraph", "content": [
        {"type": "text", "text": "Expected:", "marks": [{"type": "strong"}]},
        {"type": "text", "text": " a new recording starts"}
    ]},
    { "type": "bulletList", "content": [
        {"type": "listItem", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "item"}]}]}
    ]},
    { "type": "codeBlock", "attrs": {}, "content": [{"type": "text", "text": "code here"}] }
  ]
}
```

For a **comment body** sent through `call_api.py`, wrap the same document as `{"body": {...}}`.

Always write ADF JSON to a temp file and pass it via `--description-file` to avoid shell escaping issues.

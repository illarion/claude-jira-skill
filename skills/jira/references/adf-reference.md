## ADF (Atlassian Document Format) Reference

Jira Cloud uses ADF, not wiki markup. Wiki syntax (`h2.`, `{code}`, `#`) will render as literal text.

The `comment.py` and `create.py` scripts wrap plain text in ADF automatically. Use raw ADF only when you need rich formatting (headings, code blocks, bullet lists) via `update.py --description-file`.

### ADF Structure

```json
{
  "fields": {
    "description": {
      "version": 1, "type": "doc",
      "content": [
        { "type": "heading", "attrs": {"level": 2}, "content": [{"type": "text", "text": "Title"}] },
        { "type": "paragraph", "content": [
            {"type": "text", "text": "normal "},
            {"type": "text", "text": "inline code", "marks": [{"type": "code"}]},
            {"type": "text", "text": " bold", "marks": [{"type": "strong"}]}
        ]},
        { "type": "bulletList", "content": [
            {"type": "listItem", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "item"}]}]}
        ]},
        { "type": "codeBlock", "attrs": {}, "content": [{"type": "text", "text": "code here"}] }
      ]
    }
  }
}
```

For a **comment body**, replace `"fields": {"description": ...}` with `"body": { "version": 1, "type": "doc", "content": [...] }`.

Always write ADF JSON to a temp file and pass via `--description-file` (for `create.py` or `update.py`) to avoid shell escaping issues.

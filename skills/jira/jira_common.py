#!/usr/bin/env python3
"""Shared Jira credential and HTTP management."""

import os
import re
import sys
import json
import urllib.request
import urllib.error
import urllib.parse
import base64

DOTFILE = ".jiraskillrc"


def _find_dotfile():
    path = os.path.abspath(os.getcwd())
    while True:
        candidate = os.path.join(path, DOTFILE)
        if os.path.isfile(candidate):
            return candidate
        parent = os.path.dirname(path)
        if parent == path:
            break
        path = parent
    return None


def load_config():
    dotfile = _find_dotfile()
    if not dotfile:
        print(f"No {DOTFILE} found in any parent directory.", file=sys.stderr)
        print("Run: python3 SCRIPT_DIR/jira-auth.py login", file=sys.stderr)
        sys.exit(1)
    with open(dotfile) as f:
        config = json.load(f)
    name = config.get("name") or os.path.basename(os.path.dirname(dotfile))
    print(f"[jira: {name} — {config['url']}]", file=sys.stderr)
    return config


def save_config(path, config):
    os.makedirs(os.path.dirname(path) or ".", exist_ok=True)
    with open(path, "w") as f:
        json.dump(config, f, indent=2)
        f.write("\n")
    os.chmod(path, 0o600)


def load_credentials():
    config = load_config()
    return config["url"], config["email"], config["token"]


def load_credentials_full():
    config = load_config()
    projects = list(filter(None, (config.get("projects") or "").split(",")))
    return config["url"], config["email"], config["token"], projects


HTTP_TIMEOUT = 30


def _die_on_http_error(e):
    try:
        body = json.loads(e.read())
    except Exception:
        print(f"HTTP {e.code}: {e.reason}", file=sys.stderr)
        sys.exit(1)

    messages = body.get("errorMessages", [])
    errors = body.get("errors", {})
    parts = list(messages)
    for field, msg in errors.items():
        parts.append(f"{field}: {msg}")

    if not parts:
        print(f"HTTP {e.code}: {json.dumps(body)}", file=sys.stderr)
        sys.exit(1)

    print("\n".join(parts), file=sys.stderr)
    sys.exit(1)


def jira_request(jira_url, email, token, method, path, data=None):
    url = f"{jira_url}{path}"
    auth = base64.b64encode(f"{email}:{token}".encode()).decode()
    headers = {
        "Accept": "application/json",
        "Authorization": f"Basic {auth}",
    }
    body = None
    if data is not None:
        headers["Content-Type"] = "application/json"
        body = json.dumps(data).encode()
    req = urllib.request.Request(url, data=body, method=method.upper(), headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=HTTP_TIMEOUT) as resp:
            raw = resp.read()
            if not raw:
                return None
            return json.loads(raw)
    except urllib.error.HTTPError as e:
        _die_on_http_error(e)


def jira_get(jira_url, email, token, path):
    return jira_request(jira_url, email, token, "GET", path)


def jira_post(jira_url, email, token, path, data):
    return jira_request(jira_url, email, token, "POST", path, data)


def jira_put(jira_url, email, token, path, data):
    return jira_request(jira_url, email, token, "PUT", path, data)


SUMMARY_LIMIT = 70

_INLINE_RE = re.compile(r"(\*\*[^*]+\*\*|`[^`]+`|https?://[^\s<>()]+|\b[A-Z][A-Z0-9]+-\d+\b)")
_HEADING_RE = re.compile(r"^(#{1,6}) (.+)$")
_BULLET_RE = re.compile(r"^[-*] (.+)$")
_ORDERED_RE = re.compile(r"^(\d+)\. (.+)$")
_EXPAND_PREFIX = "▸ "


def _inline_node(token, jira_url):
    if token.startswith("**"):
        return {"type": "text", "text": token[2:-2], "marks": [{"type": "strong"}]}
    if token.startswith("`"):
        return {"type": "text", "text": token[1:-1], "marks": [{"type": "code"}]}
    if token.startswith("http"):
        return {"type": "text", "text": token, "marks": [{"type": "link", "attrs": {"href": token}}]}
    if not jira_url:
        return {"type": "text", "text": token}
    return {"type": "text", "text": token, "marks": [{"type": "link", "attrs": {"href": f"{jira_url}/browse/{token}"}}]}


def _parse_inline(text, jira_url):
    nodes = []
    for i, part in enumerate(_INLINE_RE.split(text)):
        if not part:
            continue
        if i % 2 == 0:
            nodes.append({"type": "text", "text": part})
            continue
        nodes.append(_inline_node(part, jira_url))
    return nodes


def _paragraph(text, jira_url):
    return {"type": "paragraph", "content": _parse_inline(text, jira_url)}


def _code_block(lines, lang):
    attrs = {"language": lang} if lang else {}
    content = [{"type": "text", "text": "\n".join(lines)}] if lines else []
    return {"type": "codeBlock", "attrs": attrs, "content": content}


def _append_list_item(target, current_list, list_type, text, jira_url, order=1):
    if current_list is None or current_list["type"] != list_type:
        current_list = {"type": list_type, "content": []}
        if list_type == "orderedList":
            current_list["attrs"] = {"order": order}
        target.append(current_list)
    item = {"type": "listItem", "content": [_paragraph(text, jira_url)]}
    current_list["content"].append(item)
    return current_list


def text_to_adf(text, jira_url=None):
    """Convert light markup to ADF.

    Supported: # headings, - bullets, 1. ordered items, ``` code fences,
    "▸ Title" opens a collapsed expand block that runs to the end of the text,
    **bold**, `code`, URLs and issue keys (linked when jira_url is given).
    Plain text becomes paragraphs; blank lines separate blocks.
    """
    doc = []
    target = doc
    current_list = None
    code = None
    code_lang = ""

    for raw in text.split("\n"):
        line = raw.strip()

        if code is not None and line.startswith("```"):
            target.append(_code_block(code, code_lang))
            code = None
            continue

        if code is not None:
            code.append(raw)
            continue

        if line.startswith("```"):
            code = []
            code_lang = line[3:].strip()
            current_list = None
            continue

        if not line:
            current_list = None
            continue

        if line.startswith(_EXPAND_PREFIX):
            title = line[len(_EXPAND_PREFIX):].strip()
            expand = {"type": "expand", "attrs": {"title": title}, "content": []}
            doc.append(expand)
            target = expand["content"]
            current_list = None
            continue

        heading = _HEADING_RE.match(line)
        if heading:
            level = len(heading.group(1))
            target.append({"type": "heading", "attrs": {"level": level}, "content": _parse_inline(heading.group(2), jira_url)})
            current_list = None
            continue

        bullet = _BULLET_RE.match(line)
        if bullet:
            current_list = _append_list_item(target, current_list, "bulletList", bullet.group(1), jira_url)
            continue

        ordered = _ORDERED_RE.match(line)
        if ordered:
            current_list = _append_list_item(target, current_list, "orderedList", ordered.group(2), jira_url, int(ordered.group(1)))
            continue

        target.append(_paragraph(line, jira_url))
        current_list = None

    if code is not None:
        target.append(_code_block(code, code_lang))

    return {"version": 1, "type": "doc", "content": doc}


def load_description(path, jira_url):
    """Load a description file: raw ADF JSON if it is one, light markup otherwise."""
    with open(path) as f:
        content = f.read()
    try:
        doc = json.loads(content)
    except ValueError:
        return text_to_adf(content, jira_url)
    if isinstance(doc, dict) and doc.get("type") == "doc":
        return doc
    return text_to_adf(content, jira_url)


def warn_summary(summary):
    if len(summary) <= SUMMARY_LIMIT:
        return
    print(f"Warning: summary is {len(summary)} chars, limit is {SUMMARY_LIMIT}. Use short product language.", file=sys.stderr)


def find_user(jira_url, email, token, display_name):
    query = urllib.parse.urlencode({"query": display_name})
    users = jira_get(jira_url, email, token, f"/rest/api/3/user/search?{query}")
    if not users:
        print(f"No user found matching \"{display_name}\"", file=sys.stderr)
        sys.exit(1)
    target = display_name.lower()
    for u in users:
        if u.get("displayName", "").lower() == target:
            return u
    print(f"No exact match for \"{display_name}\". Candidates:", file=sys.stderr)
    for u in users:
        identifier = u.get("emailAddress") or u.get("accountId", "")
        print(f"  - {u.get('displayName', '')} ({identifier})", file=sys.stderr)
    sys.exit(1)

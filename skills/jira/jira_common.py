#!/usr/bin/env python3
"""Shared Jira credential and HTTP management."""

import os
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


def text_to_adf(text):
    content = []
    for line in text.split("\n"):
        if not line:
            content.append({"type": "paragraph", "content": []})
            continue
        content.append({
            "type": "paragraph",
            "content": [{"type": "text", "text": line}],
        })
    return {"version": 1, "type": "doc", "content": content}


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

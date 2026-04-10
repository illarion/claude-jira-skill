#!/usr/bin/env python3
"""Transition a Jira issue to a new status."""

import sys
import json
from jira_common import load_credentials, jira_get, jira_post


def main():
    if len(sys.argv) < 3:
        print("Usage: transition.py ISSUE-KEY \"Status Name\"", file=sys.stderr)
        print("       transition.py ISSUE-KEY --list", file=sys.stderr)
        sys.exit(1)

    issue_key = sys.argv[1].upper()
    jira_url, email, token = load_credentials()

    data = jira_get(jira_url, email, token, f"/rest/api/3/issue/{issue_key}/transitions")
    transitions = data.get("transitions", [])

    if sys.argv[2] == "--list":
        names = [t["name"] for t in transitions]
        print(json.dumps(names, indent=2))
        return

    target = sys.argv[2].lower()
    match = None
    for t in transitions:
        if t["name"].lower() == target:
            match = t
            break

    if not match:
        names = [t["name"] for t in transitions]
        print(f"No transition matching \"{sys.argv[2]}\". Available: {', '.join(names)}", file=sys.stderr)
        sys.exit(1)

    jira_post(jira_url, email, token, f"/rest/api/3/issue/{issue_key}/transitions", {"transition": {"id": match["id"]}})
    print(f"{issue_key}: Status \u2192 {match['name']}")


if __name__ == "__main__":
    main()

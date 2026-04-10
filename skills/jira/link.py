#!/usr/bin/env python3
"""Link two Jira issues together."""

import sys
import json
from jira_common import load_credentials, jira_get, jira_post


def main():
    if len(sys.argv) == 2 and sys.argv[1] == "--list":
        jira_url, email, token = load_credentials()
        data = jira_get(jira_url, email, token, "/rest/api/3/issueLinkType")
        types = [{"name": t["name"], "inward": t["inward"], "outward": t["outward"]} for t in data.get("issueLinkTypes", [])]
        print(json.dumps(types, indent=2))
        return

    if len(sys.argv) < 4:
        print("Usage: link.py ISSUE-1 \"Link Type\" ISSUE-2", file=sys.stderr)
        print("       link.py --list", file=sys.stderr)
        sys.exit(1)

    source_key = sys.argv[1].upper()
    link_type_name = sys.argv[2]
    target_key = sys.argv[3].upper()
    jira_url, email, token = load_credentials()

    data = jira_get(jira_url, email, token, "/rest/api/3/issueLinkType")
    link_types = data.get("issueLinkTypes", [])

    target = link_type_name.lower()
    match = None
    for lt in link_types:
        if lt["name"].lower() == target:
            match = lt
            break

    if not match:
        names = [lt["name"] for lt in link_types]
        print(f"No link type matching \"{link_type_name}\". Available: {', '.join(names)}", file=sys.stderr)
        sys.exit(1)

    jira_post(jira_url, email, token, "/rest/api/3/issueLink", {
        "type": {"name": match["name"]},
        "outwardIssue": {"key": source_key},
        "inwardIssue": {"key": target_key},
    })
    print(f"{source_key} {match['outward']} {target_key}")


if __name__ == "__main__":
    main()

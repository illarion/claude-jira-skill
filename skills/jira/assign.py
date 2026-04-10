#!/usr/bin/env python3
"""Assign or unassign a Jira issue."""

import sys
from jira_common import load_credentials, jira_put, find_user


def main():
    if len(sys.argv) < 3:
        print("Usage: assign.py ISSUE-KEY \"Display Name\"", file=sys.stderr)
        print("       assign.py ISSUE-KEY --unassign", file=sys.stderr)
        sys.exit(1)

    issue_key = sys.argv[1].upper()
    jira_url, email, token = load_credentials()

    if sys.argv[2] == "--unassign":
        jira_put(jira_url, email, token, f"/rest/api/3/issue/{issue_key}/assignee", {"accountId": None})
        print(f"{issue_key}: Unassigned")
        return

    user = find_user(jira_url, email, token, sys.argv[2])
    account_id = user["accountId"]
    name = user.get("displayName", sys.argv[2])
    jira_put(jira_url, email, token, f"/rest/api/3/issue/{issue_key}/assignee", {"accountId": account_id})
    print(f"{issue_key}: Assigned to {name}")


if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""Add a comment to a Jira issue."""

import sys
from jira_common import load_credentials, jira_post, text_to_adf


def main():
    if len(sys.argv) < 3:
        print("Usage: comment.py ISSUE-KEY \"comment text\"", file=sys.stderr)
        sys.exit(1)

    issue_key = sys.argv[1].upper()
    text = sys.argv[2]

    jira_url, email, token = load_credentials()
    jira_post(jira_url, email, token, f"/rest/api/3/issue/{issue_key}/comment", {"body": text_to_adf(text)})
    print(f"Commented on {issue_key}")


if __name__ == "__main__":
    main()

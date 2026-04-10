#!/usr/bin/env python3
"""Update fields on an existing Jira issue."""

import sys
import json
import argparse
from jira_common import load_credentials, jira_put, text_to_adf, find_user


def main():
    parser = argparse.ArgumentParser(description="Update a Jira issue")
    parser.add_argument("issue", help="Issue key (e.g. PROJ-123)")
    parser.add_argument("--summary", help="New summary")
    parser.add_argument("--priority", help="Priority name (e.g. Major)")
    parser.add_argument("--description", help="Description text (plain text, converted to ADF)")
    parser.add_argument("--description-file", help="Path to file with raw ADF JSON for description")
    parser.add_argument("--fixversion", help="Fix version name (e.g. 1.118.2)")
    parser.add_argument("--assignee", help="Assignee display name")
    args = parser.parse_args()

    issue_key = args.issue.upper()
    jira_url, email, token = load_credentials()

    fields = {}

    if args.summary:
        fields["summary"] = args.summary

    if args.priority:
        fields["priority"] = {"name": args.priority}

    if args.description_file:
        with open(args.description_file) as f:
            fields["description"] = json.load(f)

    if args.description and "description" not in fields:
        fields["description"] = text_to_adf(args.description)

    if args.fixversion:
        fields["fixVersions"] = [{"name": args.fixversion}]

    if args.assignee:
        user = find_user(jira_url, email, token, args.assignee)
        fields["assignee"] = {"accountId": user["accountId"]}

    if not fields:
        print("No fields to update. Use --summary, --priority, --description, --description-file, --fixversion, or --assignee.", file=sys.stderr)
        sys.exit(1)

    jira_put(jira_url, email, token, f"/rest/api/3/issue/{issue_key}", {"fields": fields})
    updated = ", ".join(fields.keys())
    print(f"{issue_key}: Updated {updated}")


if __name__ == "__main__":
    main()

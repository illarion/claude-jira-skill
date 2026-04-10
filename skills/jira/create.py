#!/usr/bin/env python3
"""Create a new Jira issue."""

import sys
import json
import argparse
from jira_common import load_credentials, jira_post, text_to_adf, find_user


def main():
    parser = argparse.ArgumentParser(description="Create a Jira issue")
    parser.add_argument("project", help="Project key (e.g. PROJ)")
    parser.add_argument("summary", help="Issue summary")
    parser.add_argument("--type", default="Task", help="Issue type (default: Task)")
    parser.add_argument("--priority", help="Priority name (e.g. High)")
    parser.add_argument("--description", help="Description text")
    parser.add_argument("--description-file", help="Path to file with raw ADF JSON for description")
    parser.add_argument("--assignee", help="Assignee display name")
    parser.add_argument("--fixversion", help="Fix version name (e.g. 1.118.2)")
    args = parser.parse_args()

    jira_url, email, token = load_credentials()

    fields = {
        "project": {"key": args.project.upper()},
        "summary": args.summary,
        "issuetype": {"name": args.type},
    }

    if args.priority:
        fields["priority"] = {"name": args.priority}

    if args.description_file:
        with open(args.description_file) as f:
            fields["description"] = json.load(f)
    elif args.description:
        fields["description"] = text_to_adf(args.description)

    if args.fixversion:
        fields["fixVersions"] = [{"name": args.fixversion}]

    if args.assignee:
        user = find_user(jira_url, email, token, args.assignee)
        fields["assignee"] = {"accountId": user["accountId"]}

    result = jira_post(jira_url, email, token, "/rest/api/3/issue", {"fields": fields})
    key = result["key"]
    url = f"{jira_url}/browse/{key}"
    print(json.dumps({"key": key, "url": url}))


if __name__ == "__main__":
    main()
